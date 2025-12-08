package gitlab

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNewClient(t *testing.T) {
	client := NewClient("https://gitlab.com", "test-token")

	if client.baseURL != "https://gitlab.com" {
		t.Errorf("expected baseURL 'https://gitlab.com', got '%s'", client.baseURL)
	}
	if client.token != "test-token" {
		t.Errorf("expected token 'test-token', got '%s'", client.token)
	}
	if client.perPage != 50 {
		t.Errorf("expected default perPage 50, got %d", client.perPage)
	}
}

func TestNewClient_WithPerPage(t *testing.T) {
	client := NewClient("https://gitlab.com", "token", WithPerPage(100))

	if client.perPage != 100 {
		t.Errorf("expected perPage 100, got %d", client.perPage)
	}
}

func TestNewPublicClient(t *testing.T) {
	client := NewPublicClient()

	if client.token != "" {
		t.Errorf("expected empty token for public client, got '%s'", client.token)
	}
	if !strings.Contains(client.baseURL, "gitlab.com") {
		t.Errorf("expected gitlab.com in baseURL, got '%s'", client.baseURL)
	}
}

func TestClient_ListGroups(t *testing.T) {
	groups := []Group{
		{ID: 1, Name: "Group 1", FullPath: "group-1"},
		{ID: 2, Name: "Group 2", FullPath: "group-2"},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v4/groups" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("PRIVATE-TOKEN") != "test-token" {
			t.Error("expected PRIVATE-TOKEN header")
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(groups)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	result, err := client.ListGroups()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result) != 2 {
		t.Errorf("expected 2 groups, got %d", len(result))
	}
	if result[0].Name != "Group 1" {
		t.Errorf("expected 'Group 1', got '%s'", result[0].Name)
	}
}

func TestClient_ListProjects(t *testing.T) {
	projects := []Project{
		{ID: 1, Name: "Project 1", PathWithNamespace: "group/project-1"},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/api/v4/projects") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(projects)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	result, err := client.ListProjects()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result) != 1 {
		t.Errorf("expected 1 project, got %d", len(result))
	}
}

func TestClient_GetProject(t *testing.T) {
	project := Project{ID: 123, Name: "My Project", PathWithNamespace: "group/my-project"}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v4/projects/123" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(project)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	result, err := client.GetProject("123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Name != "My Project" {
		t.Errorf("expected 'My Project', got '%s'", result.Name)
	}
}

func TestClient_ListBranches(t *testing.T) {
	branches := []Branch{
		{Name: "main", Default: true, Protected: true},
		{Name: "develop", Default: false, Protected: false},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(branches)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	result, err := client.ListBranches("123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result) != 2 {
		t.Errorf("expected 2 branches, got %d", len(result))
	}
	if !result[0].Default {
		t.Error("expected first branch to be default")
	}
}

func TestClient_ListMergeRequests(t *testing.T) {
	mrs := []MergeRequest{
		{IID: 1, Title: "Fix bug", State: "opened"},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.RawQuery, "state=opened") {
			t.Error("expected state=opened query param")
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mrs)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	result, err := client.ListMergeRequests("123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result) != 1 {
		t.Errorf("expected 1 MR, got %d", len(result))
	}
}

func TestClient_ListPipelines(t *testing.T) {
	pipelines := []Pipeline{
		{ID: 1, IID: 100, Status: "success", Ref: "main"},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(pipelines)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	result, err := client.ListPipelines("123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result) != 1 {
		t.Errorf("expected 1 pipeline, got %d", len(result))
	}
	if result[0].Status != "success" {
		t.Errorf("expected status 'success', got '%s'", result[0].Status)
	}
}

func TestClient_GetTree(t *testing.T) {
	entries := []TreeEntry{
		{Name: "README.md", Type: "blob", Path: "README.md"},
		{Name: "src", Type: "tree", Path: "src"},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.RawQuery, "ref=main") {
			t.Error("expected ref=main query param")
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(entries)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	result, err := client.GetTree("123", "main", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result) != 2 {
		t.Errorf("expected 2 entries, got %d", len(result))
	}
}

func TestClient_SearchProjects(t *testing.T) {
	projects := []Project{
		{ID: 1, Name: "matching-project"},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.RawQuery, "search=test") {
			t.Error("expected search=test query param")
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(projects)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	result, err := client.SearchProjects("test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result) != 1 {
		t.Errorf("expected 1 project, got %d", len(result))
	}
}

func TestClient_ErrorResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"message": "Not found"}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	_, err := client.ListGroups()
	if err == nil {
		t.Error("expected error for 404 response")
	}
	if !strings.Contains(err.Error(), "404") {
		t.Errorf("expected error to contain '404', got '%s'", err.Error())
	}
}

func TestClient_NoToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("PRIVATE-TOKEN") != "" {
			t.Error("expected no PRIVATE-TOKEN header for public client")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("[]"))
	}))
	defer server.Close()

	client := NewClient(server.URL, "")
	_, err := client.ListGroups()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestIsRetryableStatus(t *testing.T) {
	tests := []struct {
		status   int
		expected bool
	}{
		{200, false},
		{400, false},
		{404, false},
		{429, true},  // Rate limit
		{500, true},  // Server error
		{502, true},
		{503, true},
	}

	for _, tt := range tests {
		result := isRetryableStatus(tt.status)
		if result != tt.expected {
			t.Errorf("isRetryableStatus(%d) = %v, expected %v", tt.status, result, tt.expected)
		}
	}
}
