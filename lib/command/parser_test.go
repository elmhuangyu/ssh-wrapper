package command

import (
	"testing"

	"github.com/AgentDrasil/ssh-wrapper/lib/config"
	"github.com/stretchr/testify/assert"
)

func TestParseNamespace(t *testing.T) {
	tests := []struct {
		name     string
		cmd      string
		expected string
	}{
		{
			name:     "git clone with colon format",
			cmd:      "clone 'git@github.com:user/repo.git'",
			expected: "user",
		},
		{
			name:     "git clone with slash format",
			cmd:      "clone 'ssh://git@github.com/user/repo.git'",
			expected: "user",
		},
		{
			name:     "path without host",
			cmd:      "clone '/user/repo.git'",
			expected: "user",
		},
		{
			name:     "path without host no slash",
			cmd:      "'user/repo.git'",
			expected: "user",
		},
		{
			name:     "no match",
			cmd:      "some other command",
			expected: "",
		},
		{
			name:     "empty string",
			cmd:      "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseNamespace(tt.cmd)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseHost(t *testing.T) {
	tests := []struct {
		name     string
		cmd      string
		expected string
	}{
		{
			name:     "github.com",
			cmd:      "ssh -T git@github.com",
			expected: "github.com",
		},
		{
			name:     "bitbucket.org",
			cmd:      "ssh -T git@bitbucket.org",
			expected: "bitbucket.org",
		},
		{
			name:     "no host",
			cmd:      "git-upload-pack user/repo",
			expected: "",
		},
		{
			name:     "empty string",
			cmd:      "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseHost(tt.cmd)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsGitCommand(t *testing.T) {
	tests := []struct {
		name     string
		cmd      string
		expected bool
	}{
		{
			name:     "contains git-upload-pack",
			cmd:      "git-upload-pack user/repo",
			expected: true,
		},
		{
			name:     "contains git-receive-pack",
			cmd:      "git-receive-pack user/repo",
			expected: true,
		},
		{
			name:     "regular ssh command",
			cmd:      "ssh git@github.com",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsGitCommand(tt.cmd)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsBasicHandshake(t *testing.T) {
	tests := []struct {
		name     string
		cmd      string
		expected bool
	}{
		{
			name:     "non-git command",
			cmd:      "ssh -T git@github.com",
			expected: true,
		},
		{
			name:     "git command",
			cmd:      "git-upload-pack user/repo",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsBasicHandshake(tt.cmd)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestVerifyAccess(t *testing.T) {
	conf := &config.Config{
		Allowed: []struct {
			Host  string   `yaml:"host"`
			Users []string `yaml:"users"`
		}{
			{
				Host:  "github.com",
				Users: []string{"user1", "user2"},
			},
		},
	}

	tests := []struct {
		name         string
		cmd          string
		expectErr    bool
		errInclusive string
	}{
		{
			name:      "allowed namespace",
			cmd:       "clone 'git@github.com:user1/repo.git'",
			expectErr: false,
		},
		{
			name:         "disallowed namespace user",
			cmd:          "clone 'git@github.com:user3/repo.git'",
			expectErr:    true,
			errInclusive: "allowlist",
		},
		{
			name:         "disallowed host namespace",
			cmd:          "clone 'git@ggithub.com:user1/repo.git'",
			expectErr:    true,
			errInclusive: "ggithub.com",
		},
		{
			name:      "basic handshake allowed",
			cmd:       "ssh -T git@github.com",
			expectErr: false,
		},
		{
			name:         "disallowed host handshake",
			cmd:          "ssh -T git@ggithub.com",
			expectErr:    true,
			errInclusive: "ggithub.com",
		},
		{
			name:         "git command without namespace",
			cmd:          "git-upload-pack user3/repo",
			expectErr:    true,
			errInclusive: "access denied",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := VerifyAccess(tt.cmd, conf)
			if tt.expectErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errInclusive)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
