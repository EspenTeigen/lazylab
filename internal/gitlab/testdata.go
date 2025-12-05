package gitlab

import (
	_ "embed"
	"encoding/json"
)

//go:embed testdata/groups.json
var groupsJSON []byte

//go:embed testdata/projects.json
var projectsJSON []byte

//go:embed testdata/pipelines.json
var pipelinesJSON []byte

//go:embed testdata/merge_requests.json
var mergeRequestsJSON []byte

//go:embed testdata/branches.json
var branchesJSON []byte

//go:embed testdata/tree.json
var treeJSON []byte

// LoadTestGroups returns test groups from embedded JSON
func LoadTestGroups() ([]Group, error) {
	var groups []Group
	err := json.Unmarshal(groupsJSON, &groups)
	return groups, err
}

// LoadTestProjects returns test projects from embedded JSON
func LoadTestProjects() ([]Project, error) {
	var projects []Project
	err := json.Unmarshal(projectsJSON, &projects)
	return projects, err
}

// LoadTestPipelines returns test pipelines from embedded JSON
func LoadTestPipelines() ([]Pipeline, error) {
	var pipelines []Pipeline
	err := json.Unmarshal(pipelinesJSON, &pipelines)
	return pipelines, err
}

// LoadTestMergeRequests returns test merge requests from embedded JSON
func LoadTestMergeRequests() ([]MergeRequest, error) {
	var mrs []MergeRequest
	err := json.Unmarshal(mergeRequestsJSON, &mrs)
	return mrs, err
}

// LoadTestBranches returns test branches from embedded JSON
func LoadTestBranches() ([]Branch, error) {
	var branches []Branch
	err := json.Unmarshal(branchesJSON, &branches)
	return branches, err
}

// LoadTestTree returns test tree entries from embedded JSON
func LoadTestTree() ([]TreeEntry, error) {
	var entries []TreeEntry
	err := json.Unmarshal(treeJSON, &entries)
	return entries, err
}
