package config

// Mode represents the TAKL operation mode
type Mode string

const (
	ModeStandalone Mode = "standalone"
	ModeEmbedded   Mode = "embedded"
)

// Project represents project-level configuration
type Project struct {
	Name string `yaml:"name"`
	Path string `yaml:"path,omitempty"`
}

// Notifications represents notification configuration
type Notifications struct {
	Enabled       bool                `yaml:"enabled"`
	GitHubActions GitHubActionsConfig `yaml:"github_actions"`
}

// GitHubActionsConfig represents GitHub Actions integration
type GitHubActionsConfig struct {
	Workflow     string   `yaml:"workflow"`
	OnCreate     bool     `yaml:"on_create"`
	OnTransition []string `yaml:"on_transition"`
}

// UI represents user interface configuration
type UI struct {
	DateFormat string `yaml:"date_format"`
}

// Paradigm represents paradigm configuration
type Paradigm struct {
	ID      string         `yaml:"id"`      // e.g., "kanban", "scrum"
	Options map[string]any `yaml:"options"` // paradigm-owned configuration
}

// GitConfig represents git-related configuration (legacy support)
type GitConfig struct {
	AutoCommit    bool   `yaml:"auto_commit"`
	CommitMessage string `yaml:"commit_message"`
	AuthorName    string `yaml:"author_name,omitempty"`
	AuthorEmail   string `yaml:"author_email,omitempty"`
}

// Config represents the complete TAKL configuration
type Config struct {
	// Core configuration
	Project       Project       `yaml:"project"`
	Paradigm      Paradigm      `yaml:"paradigm"`
	Notifications Notifications `yaml:"notifications"`
	UI            UI            `yaml:"ui"`

	// Legacy fields for backward compatibility
	Mode      Mode      `yaml:"mode"`
	IssuesDir string    `yaml:"issues_dir,omitempty"`
	WebPort   int       `yaml:"web_port,omitempty"`
	Git       GitConfig `yaml:"git"`
}

// Defaults returns a Config with sensible default values
func Defaults() Config {
	return Config{
		Project: Project{
			Name: "",
			Path: ".",
		},
		Paradigm: Paradigm{
			ID:      "kanban",
			Options: map[string]any{},
		},
		Notifications: Notifications{
			Enabled: false,
			GitHubActions: GitHubActionsConfig{
				Workflow:     ".github/workflows/takl.yml",
				OnCreate:     false,
				OnTransition: []string{},
			},
		},
		UI: UI{
			DateFormat: "2006-01-02 15:04",
		},
		// Legacy defaults
		Mode:    ModeEmbedded,
		WebPort: 3000,
		Git: GitConfig{
			AutoCommit:    true,
			CommitMessage: "Update issue: %s",
		},
	}
}
