package app

import (
	"time"

	"github.com/EspenTeigen/lazylab/internal/gitlab"
	"github.com/EspenTeigen/lazylab/internal/keymap"
)

// NewDemoScreen creates a MainScreen with mock data for demos/screenshots
func NewDemoScreen() *MainScreen {
	m := &MainScreen{
		groups:         mockGroups(),
		groupProjects:  mockGroupProjects(),
		expandedGroups: map[int]bool{1: true}, // Expand first group
		focusedPanel:   PanelNavigator,
		contentTab:     TabFiles,
		keymap:         keymap.DefaultKeyMap(),
		pipelineJobs:   make(map[int][]gitlab.Job),
	}

	// Set up mock project
	projects := m.groupProjects[1]
	if len(projects) > 0 {
		m.selectedProject = &projects[0]
		m.files = mockFiles()
		m.readmeContent = mockReadme()
		m.readmeRendered = mockReadme()
		m.pipelines = mockPipelines()
		m.mergeRequests = mockMergeRequests()
		m.branches = mockBranches()
		m.currentBranch = "main"
	}

	m.rebuildNavTree()
	return m
}

func mockGroups() []gitlab.Group {
	return []gitlab.Group{
		{
			ID:       1,
			Name:     "acme-corp",
			FullName: "Acme Corporation",
			FullPath: "acme-corp",
			Path:     "acme-corp",
		},
		{
			ID:       2,
			Name:     "internal-tools",
			FullName: "Internal Tools",
			FullPath: "internal-tools",
			Path:     "internal-tools",
		},
	}
}

func mockGroupProjects() map[int][]gitlab.Project {
	return map[int][]gitlab.Project{
		1: {
			{
				ID:                1,
				Name:              "api-gateway",
				PathWithNamespace: "acme-corp/api-gateway",
				Description:       "Central API gateway service",
				DefaultBranch:     "main",
				WebURL:            "https://gitlab.com/acme-corp/api-gateway",
			},
			{
				ID:                2,
				Name:              "web-frontend",
				PathWithNamespace: "acme-corp/web-frontend",
				Description:       "React frontend application",
				DefaultBranch:     "main",
				WebURL:            "https://gitlab.com/acme-corp/web-frontend",
			},
			{
				ID:                3,
				Name:              "auth-service",
				PathWithNamespace: "acme-corp/auth-service",
				Description:       "Authentication microservice",
				DefaultBranch:     "main",
				WebURL:            "https://gitlab.com/acme-corp/auth-service",
			},
		},
		2: {
			{
				ID:                4,
				Name:              "ci-templates",
				PathWithNamespace: "internal-tools/ci-templates",
				Description:       "Shared CI/CD templates",
				DefaultBranch:     "main",
				WebURL:            "https://gitlab.com/internal-tools/ci-templates",
			},
		},
	}
}

func mockFiles() []gitlab.TreeEntry {
	return []gitlab.TreeEntry{
		{Name: "src", Type: "tree", Path: "src"},
		{Name: "tests", Type: "tree", Path: "tests"},
		{Name: "docs", Type: "tree", Path: "docs"},
		{Name: ".gitlab-ci.yml", Type: "blob", Path: ".gitlab-ci.yml"},
		{Name: "Dockerfile", Type: "blob", Path: "Dockerfile"},
		{Name: "README.md", Type: "blob", Path: "README.md"},
		{Name: "go.mod", Type: "blob", Path: "go.mod"},
		{Name: "main.go", Type: "blob", Path: "main.go"},
	}
}

func mockReadme() string {
	return `# API Gateway

Central API gateway for Acme Corp services.

## Features

- Request routing
- Rate limiting
- Authentication
- Logging & metrics

## Quick Start

` + "```bash" + `
go run main.go
` + "```" + `

## Configuration

See ` + "`config.yaml`" + ` for options.
`
}

func mockPipelines() []gitlab.Pipeline {
	now := time.Now()
	return []gitlab.Pipeline{
		{
			ID:        1001,
			IID:       42,
			Status:    "success",
			Ref:       "main",
			SHA:       "a1b2c3d4",
			Source:    "push",
			CreatedAt: now.Add(-2 * time.Hour),
			UpdatedAt: now.Add(-1 * time.Hour),
			WebURL:    "https://gitlab.com/acme-corp/api-gateway/-/pipelines/1001",
		},
		{
			ID:        1000,
			IID:       41,
			Status:    "failed",
			Ref:       "feature/auth",
			SHA:       "e5f6g7h8",
			Source:    "push",
			CreatedAt: now.Add(-24 * time.Hour),
			UpdatedAt: now.Add(-23 * time.Hour),
			WebURL:    "https://gitlab.com/acme-corp/api-gateway/-/pipelines/1000",
		},
		{
			ID:        999,
			IID:       40,
			Status:    "success",
			Ref:       "main",
			SHA:       "i9j0k1l2",
			Source:    "merge_request",
			CreatedAt: now.Add(-48 * time.Hour),
			UpdatedAt: now.Add(-47 * time.Hour),
			WebURL:    "https://gitlab.com/acme-corp/api-gateway/-/pipelines/999",
		},
	}
}

func mockMergeRequests() []gitlab.MergeRequest {
	now := time.Now()
	return []gitlab.MergeRequest{
		{
			IID:          23,
			Title:        "Add rate limiting middleware",
			Description:  "Implements rate limiting for API endpoints",
			State:        "opened",
			SourceBranch: "feature/rate-limit",
			TargetBranch: "main",
			Author:       gitlab.User{Name: "Alice Chen", Username: "achen"},
			CreatedAt:    now.Add(-3 * time.Hour),
			WebURL:       "https://gitlab.com/acme-corp/api-gateway/-/merge_requests/23",
		},
		{
			IID:          22,
			Title:        "Fix authentication timeout",
			Description:  "Increases token refresh timeout",
			State:        "opened",
			SourceBranch: "fix/auth-timeout",
			TargetBranch: "main",
			Author:       gitlab.User{Name: "Bob Smith", Username: "bsmith"},
			CreatedAt:    now.Add(-24 * time.Hour),
			WebURL:       "https://gitlab.com/acme-corp/api-gateway/-/merge_requests/22",
		},
	}
}

func mockBranches() []gitlab.Branch {
	return []gitlab.Branch{
		{Name: "main", Default: true, Protected: true},
		{Name: "develop", Default: false, Protected: true},
		{Name: "feature/rate-limit", Default: false, Protected: false},
		{Name: "feature/auth", Default: false, Protected: false},
		{Name: "fix/auth-timeout", Default: false, Protected: false},
	}
}
