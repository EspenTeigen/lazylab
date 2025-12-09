package gitlab

import "time"

// Group represents a GitLab group
type Group struct {
	ID          int       `json:"id"`
	WebURL      string    `json:"web_url"`
	Name        string    `json:"name"`
	Path        string    `json:"path"`
	Description string    `json:"description"`
	Visibility  string    `json:"visibility"`
	FullName    string    `json:"full_name"`
	FullPath    string    `json:"full_path"`
	CreatedAt   time.Time `json:"created_at"`
	ParentID    *int      `json:"parent_id"`
	AvatarURL   *string   `json:"avatar_url"`
}

// Namespace represents a GitLab namespace (group or user)
type Namespace struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Path     string `json:"path"`
	Kind     string `json:"kind"`
	FullPath string `json:"full_path"`
}

// Project represents a GitLab project
type Project struct {
	ID                  int        `json:"id"`
	Name                string     `json:"name"`
	NameWithNamespace   string     `json:"name_with_namespace"`
	Path                string     `json:"path"`
	PathWithNamespace   string     `json:"path_with_namespace"`
	Description         string     `json:"description"`
	Visibility          string     `json:"visibility"`
	CreatedAt           time.Time  `json:"created_at"`
	DefaultBranch       string     `json:"default_branch"`
	SSHURLToRepo        string     `json:"ssh_url_to_repo"`
	HTTPURLToRepo       string     `json:"http_url_to_repo"`
	WebURL              string     `json:"web_url"`
	Topics              []string   `json:"topics"`
	StarCount           int        `json:"star_count"`
	ForksCount          int        `json:"forks_count"`
	LastActivityAt      time.Time  `json:"last_activity_at"`
	Namespace           *Namespace `json:"namespace"`
	MarkedForDeletionAt *string    `json:"marked_for_deletion_at"`
}

// Pipeline represents a GitLab CI/CD pipeline
type Pipeline struct {
	ID        int       `json:"id"`
	IID       int       `json:"iid"`
	ProjectID int       `json:"project_id"`
	SHA       string    `json:"sha"`
	Ref       string    `json:"ref"`
	Status    string    `json:"status"`
	Source    string    `json:"source"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	WebURL    string    `json:"web_url"`
	Name      string    `json:"name"`
	User      User      `json:"user"`
}

// User represents a GitLab user
type User struct {
	ID        int    `json:"id"`
	Username  string `json:"username"`
	Name      string `json:"name"`
	State     string `json:"state"`
	AvatarURL string `json:"avatar_url"`
	WebURL    string `json:"web_url"`
}

// MergeRequest represents a GitLab merge request
type MergeRequest struct {
	ID             int       `json:"id"`
	IID            int       `json:"iid"`
	ProjectID      int       `json:"project_id"`
	Title          string    `json:"title"`
	Description    string    `json:"description"`
	State          string    `json:"state"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
	TargetBranch   string    `json:"target_branch"`
	SourceBranch   string    `json:"source_branch"`
	UserNotesCount int       `json:"user_notes_count"`
	Upvotes        int       `json:"upvotes"`
	Downvotes      int       `json:"downvotes"`
	Author         User      `json:"author"`
	Assignees      []User    `json:"assignees"`
	Reviewers      []User    `json:"reviewers"`
	Labels         []string  `json:"labels"`
	Draft          bool      `json:"draft"`
	WebURL         string    `json:"web_url"`
	MergeStatus    string    `json:"merge_status"`
	HasConflicts   bool      `json:"has_conflicts"`
}

// Commit represents a Git commit
type Commit struct {
	ID             string    `json:"id"`
	ShortID        string    `json:"short_id"`
	Title          string    `json:"title"`
	Message        string    `json:"message"`
	AuthorName     string    `json:"author_name"`
	AuthorEmail    string    `json:"author_email"`
	AuthoredDate   time.Time `json:"authored_date"`
	CommitterName  string    `json:"committer_name"`
	CommitterEmail string    `json:"committer_email"`
	CommittedDate  time.Time `json:"committed_date"`
	WebURL         string    `json:"web_url"`
}

// Branch represents a Git branch
type Branch struct {
	Name               string `json:"name"`
	Commit             Commit `json:"commit"`
	Merged             bool   `json:"merged"`
	Protected          bool   `json:"protected"`
	DevelopersCanPush  bool   `json:"developers_can_push"`
	DevelopersCanMerge bool   `json:"developers_can_merge"`
	CanPush            bool   `json:"can_push"`
	Default            bool   `json:"default"`
	WebURL             string `json:"web_url"`
}

// TreeEntry represents a file or directory in a repository tree
type TreeEntry struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Type       string `json:"type"` // "tree" for directory, "blob" for file
	Path       string `json:"path"`
	Mode       string `json:"mode"`
	LastCommit *Commit // Populated separately
}

// Runner represents a GitLab CI runner
type Runner struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Active      bool   `json:"active"`
	IsShared    bool   `json:"is_shared"`
	RunnerType  string `json:"runner_type"`
	Online      bool   `json:"online"`
	Status      string `json:"status"`
}

// Job represents a CI/CD job within a pipeline
type Job struct {
	ID         int        `json:"id"`
	Name       string     `json:"name"`
	Stage      string     `json:"stage"`
	Status     string     `json:"status"`
	Ref        string     `json:"ref"`
	CreatedAt  time.Time  `json:"created_at"`
	StartedAt  *time.Time `json:"started_at"`
	FinishedAt *time.Time `json:"finished_at"`
	Duration   float64    `json:"duration"`
	WebURL     string     `json:"web_url"`
	Runner     *Runner    `json:"runner"`
	Pipeline   struct {
		ID        int    `json:"id"`
		Ref       string `json:"ref"`
		ProjectID int    `json:"project_id"`
	} `json:"pipeline"`
	Project struct {
		ID                int    `json:"id"`
		Name              string `json:"name"`
		PathWithNamespace string `json:"path_with_namespace"`
	} `json:"project"`
}
