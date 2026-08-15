package deploy

import (
	"context"
	"errors"
	"log"
	"os"
	"os/exec"
	"sync"
	"time"
)

// ErrDeployInProgress is returned when a deploy is already running. The
// triggering commit is queued rather than dropped: deploy.sh fetches and
// resets to the branch tip at the *start* of a run, so a commit pushed while
// that run is already building would otherwise never get built. Whatever the
// latest commit is when the current run finishes gets deployed next.
var ErrDeployInProgress = errors.New("a deploy is already in progress")

type Deployer struct {
	cfg Config

	mu       sync.Mutex
	running  bool
	pending  string // latest commit queued while a deploy is running; empty when none
	lastRun  time.Time
	lastSHA  string
	lastErr  error
	lastLogs string
}

func NewDeployer(cfg Config) *Deployer {
	return &Deployer{cfg: cfg}
}

// Status is a snapshot of the most recent deploy, safe to read concurrently.
type Status struct {
	Running     bool      `json:"running"`
	Pending     string    `json:"pending_commit,omitempty"`
	LastRunAt   time.Time `json:"last_run_at"`
	LastCommit  string    `json:"last_commit"`
	LastError   string    `json:"last_error,omitempty"`
	LastOutput  string    `json:"last_output,omitempty"`
	TargetRef   string    `json:"target_ref"`
	ScriptPath  string    `json:"script_path"`
	TimeoutMins float64   `json:"timeout_minutes"`
}

func (d *Deployer) Status() Status {
	d.mu.Lock()
	defer d.mu.Unlock()

	status := Status{
		Running:     d.running,
		Pending:     d.pending,
		LastRunAt:   d.lastRun,
		LastCommit:  d.lastSHA,
		LastOutput:  d.lastLogs,
		TargetRef:   d.cfg.Ref(),
		ScriptPath:  d.cfg.ScriptPath,
		TimeoutMins: d.cfg.Timeout.Minutes(),
	}
	if d.lastErr != nil {
		status.LastError = d.lastErr.Error()
	}
	return status
}

// Trigger starts a deploy in the background. If one is already running, it
// queues commitSHA (overwriting any earlier queued commit) and returns
// ErrDeployInProgress so the webhook can answer GitHub immediately either way;
// the queued commit is deployed automatically once the current run finishes.
func (d *Deployer) Trigger(commitSHA string) error {
	d.mu.Lock()
	if d.running {
		d.pending = commitSHA
		d.mu.Unlock()
		return ErrDeployInProgress
	}
	d.running = true
	d.pending = ""
	d.lastRun = time.Now().UTC()
	d.lastSHA = commitSHA
	d.lastErr = nil
	d.lastLogs = ""
	d.mu.Unlock()

	go d.run(commitSHA)
	return nil
}

func (d *Deployer) run(commitSHA string) {
	ctx, cancel := context.WithTimeout(context.Background(), d.cfg.Timeout)
	defer cancel()

	log.Printf("[deploy] starting for commit %s (branch %s)", commitSHA, d.cfg.Branch)

	cmd := exec.CommandContext(ctx, "/bin/sh", d.cfg.ScriptPath)
	cmd.Env = append(os.Environ(),
		"DEPLOY_BRANCH="+d.cfg.Branch,
		"DEPLOY_COMMIT="+commitSHA,
	)

	output, err := cmd.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		err = errors.New("deploy timed out after " + d.cfg.Timeout.String())
	}

	// The script hands the container swap off to a detached helper, so this
	// process is usually killed before it gets here. Anything we do manage to
	// log is from a deploy that failed early, which is exactly what matters.
	if err != nil {
		log.Printf("[deploy] failed for commit %s: %v\n%s", commitSHA, err, output)
	} else {
		log.Printf("[deploy] build finished for commit %s, container swap handed off\n%s", commitSHA, output)
	}

	d.mu.Lock()
	d.running = false
	d.lastErr = err
	d.lastLogs = tail(string(output), 8000)
	next := d.pending
	d.pending = ""
	d.mu.Unlock()

	if next != "" {
		log.Printf("[deploy] commit %s landed while deploying %s, starting it now", next, commitSHA)
		if triggerErr := d.Trigger(next); triggerErr != nil {
			log.Printf("[deploy] failed to start queued deploy for %s: %v", next, triggerErr)
		}
	}
}

// tail keeps the last n bytes of the script output, which is where failures show up.
func tail(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return "...(truncated)...\n" + s[len(s)-n:]
}
