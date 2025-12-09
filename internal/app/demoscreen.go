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
		isDemo:         true,
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
	now := time.Now()
	return []gitlab.TreeEntry{
		{Name: "src", Type: "tree", Path: "src", LastCommit: &gitlab.Commit{Title: "Add rate limiting middleware", AuthorName: "Alice Chen", AuthoredDate: now.Add(-2 * time.Hour)}},
		{Name: "tests", Type: "tree", Path: "tests", LastCommit: &gitlab.Commit{Title: "Add unit tests for auth", AuthorName: "Bob Smith", AuthoredDate: now.Add(-5 * time.Hour)}},
		{Name: "docs", Type: "tree", Path: "docs", LastCommit: &gitlab.Commit{Title: "Update API documentation", AuthorName: "Carol Jones", AuthoredDate: now.Add(-3 * 24 * time.Hour)}},
		{Name: ".gitlab-ci.yml", Type: "blob", Path: ".gitlab-ci.yml", LastCommit: &gitlab.Commit{Title: "Add deploy stage to CI", AuthorName: "Alice Chen", AuthoredDate: now.Add(-24 * time.Hour)}},
		{Name: "Dockerfile", Type: "blob", Path: "Dockerfile", LastCommit: &gitlab.Commit{Title: "Optimize Docker image size", AuthorName: "Bob Smith", AuthoredDate: now.Add(-48 * time.Hour)}},
		{Name: "README.md", Type: "blob", Path: "README.md", LastCommit: &gitlab.Commit{Title: "Add quick start guide", AuthorName: "Carol Jones", AuthoredDate: now.Add(-7 * 24 * time.Hour)}},
		{Name: "go.mod", Type: "blob", Path: "go.mod", LastCommit: &gitlab.Commit{Title: "Update dependencies", AuthorName: "Alice Chen", AuthoredDate: now.Add(-12 * time.Hour)}},
		{Name: "main.go", Type: "blob", Path: "main.go", LastCommit: &gitlab.Commit{Title: "Implement graceful shutdown", AuthorName: "Bob Smith", AuthoredDate: now.Add(-4 * time.Hour)}},
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
			User:      gitlab.User{Username: "achen", Name: "Alice Chen"},
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
			User:      gitlab.User{Username: "bsmith", Name: "Bob Smith"},
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
			User:      gitlab.User{Username: "achen", Name: "Alice Chen"},
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
			Reviewers:    []gitlab.User{{Username: "bsmith", Name: "Bob Smith"}},
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
			Reviewers:    []gitlab.User{{Username: "achen", Name: "Alice Chen"}, {Username: "cjones", Name: "Carol Jones"}},
			CreatedAt:    now.Add(-24 * time.Hour),
			WebURL:       "https://gitlab.com/acme-corp/api-gateway/-/merge_requests/22",
		},
	}
}

func mockBranches() []gitlab.Branch {
	return []gitlab.Branch{
		{Name: "main", Default: true, Protected: true, Commit: gitlab.Commit{Title: "Merge branch 'feature/logging' into main", AuthorName: "Alice Chen"}},
		{Name: "develop", Default: false, Protected: true, Commit: gitlab.Commit{Title: "Add prometheus metrics endpoint", AuthorName: "Bob Smith"}},
		{Name: "feature/rate-limit", Default: false, Protected: false, Commit: gitlab.Commit{Title: "Implement token bucket algorithm", AuthorName: "Alice Chen"}},
		{Name: "feature/auth", Default: false, Protected: false, Commit: gitlab.Commit{Title: "Fix JWT validation for expired tokens", AuthorName: "Bob Smith"}},
		{Name: "fix/auth-timeout", Default: false, Protected: false, Commit: gitlab.Commit{Title: "Increase timeout to 30 seconds", AuthorName: "Bob Smith"}},
	}
}

// MockFileContent returns mock content for demo file viewing
var MockFileContent = map[string]string{
	"main.go": `package main

import (
	"log"
	"net/http"

	"github.com/acme-corp/api-gateway/internal/router"
	"github.com/acme-corp/api-gateway/internal/middleware"
)

func main() {
	r := router.New()

	// Apply middleware
	r.Use(middleware.Logger())
	r.Use(middleware.RateLimit(100))
	r.Use(middleware.Auth())

	// Register routes
	r.GET("/health", healthHandler)
	r.GET("/api/v1/*", proxyHandler)

	log.Println("Starting API Gateway on :8080")
	log.Fatal(http.ListenAndServe(":8080", r))
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func proxyHandler(w http.ResponseWriter, r *http.Request) {
	// Forward request to appropriate service
	router.Forward(w, r)
}
`,
	"go.mod": `module github.com/acme-corp/api-gateway

go 1.21

require (
	github.com/gorilla/mux v1.8.1
	github.com/prometheus/client_golang v1.17.0
	go.uber.org/zap v1.26.0
)
`,
	"Dockerfile": `FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -o api-gateway ./cmd/server

FROM alpine:3.18
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/api-gateway .
EXPOSE 8080
CMD ["./api-gateway"]
`,
	".gitlab-ci.yml": `stages:
  - test
  - build
  - deploy

test:
  stage: test
  image: golang:1.21
  script:
    - go test -v ./...
    - go vet ./...

build:
  stage: build
  image: docker:24
  services:
    - docker:dind
  script:
    - docker build -t api-gateway:$CI_COMMIT_SHA .
    - docker push $REGISTRY/api-gateway:$CI_COMMIT_SHA

deploy:
  stage: deploy
  only:
    - main
  script:
    - kubectl set image deployment/api-gateway api-gateway=$REGISTRY/api-gateway:$CI_COMMIT_SHA
`,
}
