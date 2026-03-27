package command

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/AgentDrasil/ssh-wrapper/lib/config"
)

var (
	ErrAccessDenied = errors.New("access denied")

	namespaceRegex1 = regexp.MustCompile(`'/?([a-zA-Z0-9\-_]+)/[a-zA-Z0-9\-_.]+'`)
	namespaceRegex2 = regexp.MustCompile(`'[^']*[:/]([a-zA-Z0-9\-_]+)/[a-zA-Z0-9\-_.]+'`)
	hostRegex       = regexp.MustCompile(`@([a-zA-Z0-9\-\.]+)`)
)

func parseNamespace(cmd string) string {
	matches := namespaceRegex1.FindStringSubmatch(cmd)
	if len(matches) > 1 {
		return matches[1]
	}
	matches = namespaceRegex2.FindStringSubmatch(cmd)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

func parseHost(cmd string) string {
	matches := hostRegex.FindStringSubmatch(cmd)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

func isHostAllowed(cmd string, conf *config.Config) bool {
	host := parseHost(cmd)
	if host == "" {
		return false
	}
	for _, entry := range conf.Allowed {
		if entry.Host == host {
			return true
		}
	}
	return false
}

func IsGitCommand(cmd string) bool {
	return strings.Contains(cmd, "git-")
}

func IsBasicHandshake(cmd string) bool {
	return !IsGitCommand(cmd)
}

func VerifyAccess(cmd string, conf *config.Config) error {
	namespace := parseNamespace(cmd)
	host := parseHost(cmd)

	if namespace != "" {
		for _, entry := range conf.Allowed {
			if entry.Host == host {
				for _, u := range entry.Users {
					if u == namespace {
						return nil
					}
				}
			}
		}
		return fmt.Errorf("%w: host '%s', repo namespace '%s' is not in allowlist", ErrAccessDenied, host, namespace)
	} else if IsBasicHandshake(cmd) {
		if isHostAllowed(cmd, conf) {
			return nil
		}
		return fmt.Errorf("%w: host '%s' is not in allowlist", ErrAccessDenied, host)
	}

	return ErrAccessDenied
}
