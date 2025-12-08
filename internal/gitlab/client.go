package gitlab

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/EspenTeigen/lazylab/internal/config"
)

// Client is a GitLab API client
type Client struct {
	baseURL    string
	token      string
	httpClient *http.Client
	perPage    int
}

// ClientOption allows configuring the client
type ClientOption func(*Client)

// WithPerPage sets the default per_page for list requests
func WithPerPage(n int) ClientOption {
	return func(c *Client) {
		c.perPage = n
	}
}

// NewClient creates a new GitLab client
func NewClient(baseURL, token string, opts ...ClientOption) *Client {
	c := &Client{
		baseURL: baseURL,
		token:   token,
		httpClient: &http.Client{
			Timeout: config.DefaultTimeout,
		},
		perPage: config.DefaultPerPage,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// NewPublicClient creates a client for gitlab.com public repos (no auth)
func NewPublicClient() *Client {
	return NewClient("https://"+config.DefaultHost, "")
}

// isRetryableStatus returns true if the status code should trigger a retry
func isRetryableStatus(statusCode int) bool {
	return statusCode >= config.ServerErrorMin || statusCode == config.RateLimitStatus
}

// doWithRetry executes an HTTP request with retry logic
func (c *Client) doWithRetry(req *http.Request) (*http.Response, error) {
	var lastErr error
	backoff := config.InitialBackoff

	for attempt := 0; attempt <= config.MaxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-req.Context().Done():
				return nil, req.Context().Err()
			default:
			}
			// Use a simple sleep - in production you'd want a timer
			sleepDuration := backoff
			backoff *= 2
			if backoff > config.MaxBackoff {
				backoff = config.MaxBackoff
			}
			// Simple blocking sleep
			<-sleepChan(sleepDuration)
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("request failed (attempt %d/%d): %w", attempt+1, config.MaxRetries+1, err)
			continue
		}

		if !isRetryableStatus(resp.StatusCode) {
			return resp, nil
		}

		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		lastErr = fmt.Errorf("API error %d (attempt %d/%d): %s", resp.StatusCode, attempt+1, config.MaxRetries+1, string(body))
	}

	return nil, lastErr
}

// sleepChan returns a channel that closes after the duration
func sleepChan(d time.Duration) <-chan time.Time {
	return time.After(d)
}

func (c *Client) get(path string, result interface{}) error {
	reqURL := c.baseURL + "/api/v4" + path

	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	if c.token != "" {
		req.Header.Set("PRIVATE-TOKEN", c.token)
	}

	resp, err := c.doWithRetry(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
		return fmt.Errorf("decoding response: %w", err)
	}

	return nil
}

// GetProject fetches a single project by ID or path
func (c *Client) GetProject(projectID string) (*Project, error) {
	var project Project
	path := fmt.Sprintf("/projects/%s", url.PathEscape(projectID))
	if err := c.get(path, &project); err != nil {
		return nil, err
	}
	return &project, nil
}

// GetTree fetches the repository tree for a project
func (c *Client) GetTree(projectID, ref, treePath string) ([]TreeEntry, error) {
	var entries []TreeEntry
	path := fmt.Sprintf("/projects/%s/repository/tree?ref=%s&per_page=%d",
		url.PathEscape(projectID),
		url.QueryEscape(ref),
		c.perPage)

	if treePath != "" {
		path += "&path=" + url.QueryEscape(treePath)
	}

	if err := c.get(path, &entries); err != nil {
		return nil, err
	}
	return entries, nil
}

// GetFileContent fetches raw file content
func (c *Client) GetFileContent(projectID string, filePath string, ref string) (string, error) {
	reqURL := fmt.Sprintf("%s/api/v4/projects/%s/repository/files/%s/raw?ref=%s",
		c.baseURL,
		url.PathEscape(projectID),
		url.PathEscape(filePath),
		url.QueryEscape(ref))

	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		return "", fmt.Errorf("creating request: %w", err)
	}

	if c.token != "" {
		req.Header.Set("PRIVATE-TOKEN", c.token)
	}

	resp, err := c.doWithRetry(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("reading response: %w", err)
	}

	return string(content), nil
}

// ListBranches fetches branches for a project
func (c *Client) ListBranches(projectID string) ([]Branch, error) {
	var branches []Branch
	path := fmt.Sprintf("/projects/%s/repository/branches?per_page=%d", url.PathEscape(projectID), c.perPage)
	if err := c.get(path, &branches); err != nil {
		return nil, err
	}
	return branches, nil
}

// ListMergeRequests fetches open MRs for a project
func (c *Client) ListMergeRequests(projectID string) ([]MergeRequest, error) {
	var mrs []MergeRequest
	path := fmt.Sprintf("/projects/%s/merge_requests?state=opened&per_page=%d", url.PathEscape(projectID), c.perPage)
	if err := c.get(path, &mrs); err != nil {
		return nil, err
	}
	return mrs, nil
}

// ListPipelines fetches recent pipelines for a project
func (c *Client) ListPipelines(projectID string) ([]Pipeline, error) {
	var pipelines []Pipeline
	path := fmt.Sprintf("/projects/%s/pipelines?per_page=%d", url.PathEscape(projectID), c.perPage)
	if err := c.get(path, &pipelines); err != nil {
		return nil, err
	}
	return pipelines, nil
}

// filterActiveProjects removes projects that are marked for deletion
func filterActiveProjects(projects []Project) []Project {
	result := make([]Project, 0, len(projects))
	for _, p := range projects {
		// GitLab renames projects to include "deletion_scheduled" but doesn't always
		// set marked_for_deletion_at, so check both
		markedForDeletion := p.MarkedForDeletionAt != nil && *p.MarkedForDeletionAt != ""
		if !markedForDeletion && !strings.Contains(p.Name, "deletion_scheduled") {
			result = append(result, p)
		}
	}
	return result
}

// ListGroupProjects fetches projects from a group
func (c *Client) ListGroupProjects(groupID string) ([]Project, error) {
	var projects []Project
	path := fmt.Sprintf("/groups/%s/projects?per_page=%d&order_by=last_activity_at", url.PathEscape(groupID), c.perPage)
	if err := c.get(path, &projects); err != nil {
		return nil, err
	}
	return filterActiveProjects(projects), nil
}

// ListProjects fetches all accessible projects (for self-hosted instances)
func (c *Client) ListProjects() ([]Project, error) {
	var projects []Project
	path := fmt.Sprintf("/projects?per_page=%d&order_by=last_activity_at&membership=true", c.perPage)
	if err := c.get(path, &projects); err != nil {
		return nil, err
	}
	return filterActiveProjects(projects), nil
}

// ListGroups fetches all accessible groups
func (c *Client) ListGroups() ([]Group, error) {
	var groups []Group
	path := fmt.Sprintf("/groups?per_page=%d&order_by=name", c.perPage)
	if err := c.get(path, &groups); err != nil {
		return nil, err
	}
	return groups, nil
}

// ListPipelineJobs fetches jobs for a specific pipeline
func (c *Client) ListPipelineJobs(projectID string, pipelineID int) ([]Job, error) {
	var jobs []Job
	path := fmt.Sprintf("/projects/%s/pipelines/%d/jobs?per_page=%d", url.PathEscape(projectID), pipelineID, c.perPage)
	if err := c.get(path, &jobs); err != nil {
		return nil, err
	}
	return jobs, nil
}

// SearchProjects searches for projects by name
func (c *Client) SearchProjects(query string) ([]Project, error) {
	var projects []Project
	path := fmt.Sprintf("/projects?search=%s&per_page=%d&order_by=last_activity_at", url.QueryEscape(query), c.perPage)
	if err := c.get(path, &projects); err != nil {
		return nil, err
	}
	return filterActiveProjects(projects), nil
}

// SearchGroups searches for groups by name
func (c *Client) SearchGroups(query string) ([]Group, error) {
	var groups []Group
	path := fmt.Sprintf("/groups?search=%s&per_page=%d&order_by=name", url.QueryEscape(query), c.perPage)
	if err := c.get(path, &groups); err != nil {
		return nil, err
	}
	return groups, nil
}

// GetJobLog fetches the log/trace for a specific job
func (c *Client) GetJobLog(projectID string, jobID int) (string, error) {
	reqURL := fmt.Sprintf("%s/api/v4/projects/%s/jobs/%d/trace",
		c.baseURL,
		url.PathEscape(projectID),
		jobID)

	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		return "", fmt.Errorf("creating request: %w", err)
	}

	if c.token != "" {
		req.Header.Set("PRIVATE-TOKEN", c.token)
	}

	resp, err := c.doWithRetry(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("reading response: %w", err)
	}

	return string(content), nil
}
