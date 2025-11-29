package config

import (
	"fmt"
	"os"
	"strings"
)

type Config struct {
	Organization string
	Project      string
	Repositories []string
	PAT          string
}

func LoadConfig(org, project, repo, pat string) (*Config, error) {
	var repositories []string
	if repo != "" {
		// Split comma-separated repositories
		for _, r := range strings.Split(repo, ",") {
			r = strings.TrimSpace(r)
			if r != "" {
				repositories = append(repositories, r)
			}
		}
	}

	cfg := &Config{
		Organization: getValueOrEnv(org, "AZURE_DEVOPS_ORG"),
		Project:      getValueOrEnv(project, "AZURE_DEVOPS_PROJECT"),
		Repositories: repositories,
		PAT:          getValueOrEnv(pat, "AZURE_DEVOPS_PAT"),
	}

	if cfg.Organization == "" {
		return nil, fmt.Errorf("organization is required (use -o flag or AZURE_DEVOPS_ORG env var)")
	}
	if cfg.Project == "" {
		return nil, fmt.Errorf("project is required (use -p flag or AZURE_DEVOPS_PROJECT env var)")
	}
	if len(cfg.Repositories) == 0 {
		return nil, fmt.Errorf("at least one repository is required (use -r flag, comma-separated for multiple)")
	}
	if cfg.PAT == "" {
		return nil, fmt.Errorf("PAT is required (use --pat flag or AZURE_DEVOPS_PAT env var)")
	}

	return cfg, nil
}

func getValueOrEnv(value, envKey string) string {
	if value != "" {
		return value
	}
	return os.Getenv(envKey)
}

func (c *Config) Validate() error {
	if c.Organization == "" {
		return fmt.Errorf("organization cannot be empty")
	}
	if c.Project == "" {
		return fmt.Errorf("project cannot be empty")
	}
	if len(c.Repositories) == 0 {
		return fmt.Errorf("repositories cannot be empty")
	}
	if c.PAT == "" {
		return fmt.Errorf("PAT cannot be empty")
	}
	return nil
}
