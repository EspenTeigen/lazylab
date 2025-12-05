package gitlab

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// Client is a GitLab API client
type Client struct {
	baseURL    string
	token      string
	httpClient *http.Client
}

// NewClient creates a new GitLab client
func NewClient(baseURL string, token string) *Client {
	return &Client{
		baseURL: baseURL,
		token:   token,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// NewPublicClient creates a client for gitlab.com public repos (no auth)
func NewPublicClient() *Client {
	return NewClient("https://gitlab.com", "")
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

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("executing request: %w", err)
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
func (c *Client) GetTree(projectID string, ref string, treePath string) ([]TreeEntry, error) {
	var entries []TreeEntry
	path := fmt.Sprintf("/projects/%s/repository/tree?ref=%s&per_page=100",
		url.PathEscape(projectID),
		url.QueryEscape(ref))

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

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("executing request: %w", err)
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
	path := fmt.Sprintf("/projects/%s/repository/branches?per_page=20", url.PathEscape(projectID))
	if err := c.get(path, &branches); err != nil {
		return nil, err
	}
	return branches, nil
}

// ListMergeRequests fetches open MRs for a project
func (c *Client) ListMergeRequests(projectID string) ([]MergeRequest, error) {
	var mrs []MergeRequest
	path := fmt.Sprintf("/projects/%s/merge_requests?state=opened&per_page=20", url.PathEscape(projectID))
	if err := c.get(path, &mrs); err != nil {
		return nil, err
	}
	return mrs, nil
}

// ListPipelines fetches recent pipelines for a project
func (c *Client) ListPipelines(projectID string) ([]Pipeline, error) {
	var pipelines []Pipeline
	path := fmt.Sprintf("/projects/%s/pipelines?per_page=20", url.PathEscape(projectID))
	if err := c.get(path, &pipelines); err != nil {
		return nil, err
	}
	return pipelines, nil
}

// ListGroupProjects fetches projects from a group
func (c *Client) ListGroupProjects(groupID string) ([]Project, error) {
	var projects []Project
	path := fmt.Sprintf("/groups/%s/projects?per_page=50&order_by=last_activity_at", url.PathEscape(groupID))
	if err := c.get(path, &projects); err != nil {
		return nil, err
	}
	return projects, nil
}
