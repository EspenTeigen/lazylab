package config

import "time"

// API configuration
const (
	DefaultPerPage   = 50
	DefaultTimeout   = 30 * time.Second
	MaxRetries       = 3
	InitialBackoff   = 500 * time.Millisecond
	MaxBackoff       = 5 * time.Second
	DefaultHost      = "gitlab.com"
	DefaultBranch    = "main"
	DefaultAPIPath   = "/api/v4"
	RateLimitStatus  = 429
	ServerErrorMin   = 500
)

// Environment variable names
const (
	EnvGitLabToken = "GITLAB_TOKEN"
	EnvGitLabHost  = "GITLAB_HOST"
	EnvGitLabGroup = "GITLAB_GROUP"
)

// UI layout ratios
const (
	NavigatorWidthRatio = 0.15
	ContentWidthRatio   = 0.55
	DetailWidthRatio    = 0.30
	ReadmeHeightRatio   = 0.60
)

// Popup configuration
const (
	PopupWidthRatio  = 0.6
	PopupHeightRatio = 0.7
	PopupMinWidth    = 40
	PopupMinHeight   = 10
	PopupPadding     = 4
)

// Search configuration
const (
	SearchMinQueryLength = 2
)

// UI element sizes
const (
	BorderSize   = 2
	StatusBarHeight = 1
)
