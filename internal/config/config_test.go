package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadConfig(t *testing.T) {
	tests := []struct {
		name          string
		org           string
		project       string
		repo          string
		pat           string
		envVars       map[string]string
		expectError   bool
		errorMsg      string
		expectedRepos []string
	}{
		{
			name:          "all values from flags - single repo",
			org:           "myorg",
			project:       "myproject",
			repo:          "myrepo",
			pat:           "mytoken",
			expectError:   false,
			expectedRepos: []string{"myrepo"},
		},
		{
			name:          "multiple repositories",
			org:           "myorg",
			project:       "myproject",
			repo:          "repo1,repo2,repo3",
			pat:           "mytoken",
			expectError:   false,
			expectedRepos: []string{"repo1", "repo2", "repo3"},
		},
		{
			name:          "multiple repositories with spaces",
			org:           "myorg",
			project:       "myproject",
			repo:          " repo1 , repo2 , repo3 ",
			pat:           "mytoken",
			expectError:   false,
			expectedRepos: []string{"repo1", "repo2", "repo3"},
		},
		{
			name:    "values from environment variables",
			org:     "",
			project: "",
			repo:    "myrepo",
			pat:     "",
			envVars: map[string]string{
				"AZURE_DEVOPS_ORG":     "envorg",
				"AZURE_DEVOPS_PROJECT": "envproject",
				"AZURE_DEVOPS_PAT":     "envtoken",
			},
			expectError:   false,
			expectedRepos: []string{"myrepo"},
		},
		{
			name:        "missing organization",
			org:         "",
			project:     "myproject",
			repo:        "myrepo",
			pat:         "mytoken",
			expectError: true,
			errorMsg:    "organization is required",
		},
		{
			name:        "missing project",
			org:         "myorg",
			project:     "",
			repo:        "myrepo",
			pat:         "mytoken",
			expectError: true,
			errorMsg:    "project is required",
		},
		{
			name:        "missing repository",
			org:         "myorg",
			project:     "myproject",
			repo:        "",
			pat:         "mytoken",
			expectError: true,
			errorMsg:    "repository is required",
		},
		{
			name:        "missing PAT",
			org:         "myorg",
			project:     "myproject",
			repo:        "myrepo",
			pat:         "",
			expectError: true,
			errorMsg:    "PAT is required",
		},
		{
			name:    "flags override environment variables",
			org:     "flagorg",
			project: "flagproject",
			repo:    "myrepo",
			pat:     "flagtoken",
			envVars: map[string]string{
				"AZURE_DEVOPS_ORG":     "envorg",
				"AZURE_DEVOPS_PROJECT": "envproject",
				"AZURE_DEVOPS_PAT":     "envtoken",
			},
			expectError:   false,
			expectedRepos: []string{"myrepo"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variables
			for key, value := range tt.envVars {
				os.Setenv(key, value)
				defer os.Unsetenv(key)
			}

			cfg, err := LoadConfig(tt.org, tt.project, tt.repo, tt.pat)

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, cfg)

			// Validate configuration values
			if tt.org != "" {
				assert.Equal(t, tt.org, cfg.Organization)
			} else if tt.envVars["AZURE_DEVOPS_ORG"] != "" {
				assert.Equal(t, tt.envVars["AZURE_DEVOPS_ORG"], cfg.Organization)
			}

			if tt.project != "" {
				assert.Equal(t, tt.project, cfg.Project)
			} else if tt.envVars["AZURE_DEVOPS_PROJECT"] != "" {
				assert.Equal(t, tt.envVars["AZURE_DEVOPS_PROJECT"], cfg.Project)
			}

			assert.Equal(t, tt.expectedRepos, cfg.Repositories)

			if tt.pat != "" {
				assert.Equal(t, tt.pat, cfg.PAT)
			} else if tt.envVars["AZURE_DEVOPS_PAT"] != "" {
				assert.Equal(t, tt.envVars["AZURE_DEVOPS_PAT"], cfg.PAT)
			}
		})
	}
}

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name        string
		config      *Config
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid config",
			config: &Config{
				Organization: "myorg",
				Project:      "myproject",
				Repositories: []string{"myrepo"},
				PAT:          "mytoken",
			},
			expectError: false,
		},
		{
			name: "empty organization",
			config: &Config{
				Organization: "",
				Project:      "myproject",
				Repositories: []string{"myrepo"},
				PAT:          "mytoken",
			},
			expectError: true,
			errorMsg:    "organization cannot be empty",
		},
		{
			name: "empty project",
			config: &Config{
				Organization: "myorg",
				Project:      "",
				Repositories: []string{"myrepo"},
				PAT:          "mytoken",
			},
			expectError: true,
			errorMsg:    "project cannot be empty",
		},
		{
			name: "empty repositories",
			config: &Config{
				Organization: "myorg",
				Project:      "myproject",
				Repositories: []string{},
				PAT:          "mytoken",
			},
			expectError: true,
			errorMsg:    "repositories cannot be empty",
		},
		{
			name: "empty PAT",
			config: &Config{
				Organization: "myorg",
				Project:      "myproject",
				Repositories: []string{"myrepo"},
				PAT:          "",
			},
			expectError: true,
			errorMsg:    "PAT cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestGetValueOrEnv(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		envKey   string
		envValue string
		expected string
	}{
		{
			name:     "value provided",
			value:    "myvalue",
			envKey:   "TEST_ENV",
			envValue: "envvalue",
			expected: "myvalue",
		},
		{
			name:     "value from environment",
			value:    "",
			envKey:   "TEST_ENV",
			envValue: "envvalue",
			expected: "envvalue",
		},
		{
			name:     "no value or environment",
			value:    "",
			envKey:   "TEST_ENV",
			envValue: "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				os.Setenv(tt.envKey, tt.envValue)
				defer os.Unsetenv(tt.envKey)
			}

			result := getValueOrEnv(tt.value, tt.envKey)
			assert.Equal(t, tt.expected, result)
		})
	}
}
