package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

// parseRepoArg parses repository argument with default fallback
// Supports: PROJECT/REPO, REPO, or empty (uses defaults)
func parseRepoArg(arg string) (project, repo string, err error) {
	if arg == "" {
		// Use defaults from config
		project = viper.GetString("bitbucket.default_project")
		repo = viper.GetString("bitbucket.default_repo")
		if project == "" || repo == "" {
			return "", "", fmt.Errorf("no default project/repo configured, use PROJECT/REPO")
		}
		return project, repo, nil
	}

	parts := strings.Split(arg, "/")
	switch len(parts) {
	case 2:
		return parts[0], parts[1], nil
	case 1:
		// Only repo name provided, use default project
		project = viper.GetString("bitbucket.default_project")
		repo = parts[0]
		if project == "" {
			return "", "", fmt.Errorf("no default project configured, use PROJECT/REPO format")
		}
		return project, repo, nil
	default:
		return "", "", fmt.Errorf("invalid format: use PROJECT/REPO or REPO")
	}
}

// getDefaultProject returns the default project for a service
func getDefaultProject(service string) string {
	return viper.GetString(fmt.Sprintf("%s.default_project", service))
}

// getDefaultSpace returns the default space for Confluence
func getDefaultSpace() string {
	return viper.GetString("confluence.default_space")
}

// getDefaultRepo returns formatted default repo for display
func getDefaultRepo() string {
	project := viper.GetString("bitbucket.default_project")
	repo := viper.GetString("bitbucket.default_repo")
	if project != "" && repo != "" {
		return fmt.Sprintf("%s/%s", project, repo)
	}
	return ""
}
