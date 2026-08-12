package deploy

import (
	"strconv"
	"strings"
	"time"

	"loof/internal/config"
)

// Config holds everything the webhook needs to decide whether a push should
// trigger a redeploy, and how to run the deploy itself.
type Config struct {
	Secret     string        // GITHUB_WEBHOOK_SECRET, shared with GitHub
	Branch     string        // DEPLOY_BRANCH, only pushes to this branch deploy
	ScriptPath string        // DEPLOY_SCRIPT, the shell script that rebuilds and restarts
	Timeout    time.Duration // DEPLOY_TIMEOUT_MINUTES, hard limit on a deploy run
}

func LoadConfig() Config {
	branch := strings.TrimSpace(config.GetEnv("DEPLOY_BRANCH"))
	if branch == "" {
		branch = "main"
	}

	script := strings.TrimSpace(config.GetEnv("DEPLOY_SCRIPT"))
	if script == "" {
		script = "/root/scripts/deploy.sh"
	}

	timeout := 15 * time.Minute
	if raw := strings.TrimSpace(config.GetEnv("DEPLOY_TIMEOUT_MINUTES")); raw != "" {
		if minutes, err := strconv.Atoi(raw); err == nil && minutes > 0 {
			timeout = time.Duration(minutes) * time.Minute
		}
	}

	return Config{
		Secret:     strings.TrimSpace(config.GetEnv("GITHUB_WEBHOOK_SECRET")),
		Branch:     branch,
		ScriptPath: script,
		Timeout:    timeout,
	}
}

// Ref is the fully qualified git ref a push must carry to trigger a deploy.
func (c Config) Ref() string {
	return "refs/heads/" + c.Branch
}
