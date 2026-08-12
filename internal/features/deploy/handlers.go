package deploy

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"log"
	"strings"

	"github.com/gofiber/fiber/v3"
)

const zeroSHA = "0000000000000000000000000000000000000000"

// pushPayload is the subset of GitHub's push event we actually care about.
type pushPayload struct {
	Ref        string `json:"ref"`
	After      string `json:"after"`
	Deleted    bool   `json:"deleted"`
	HeadCommit *struct {
		ID      string `json:"id"`
		Message string `json:"message"`
	} `json:"head_commit"`
	Repository struct {
		FullName string `json:"full_name"`
	} `json:"repository"`
	Pusher struct {
		Name string `json:"name"`
	} `json:"pusher"`
}

// HandleGithubWebhook verifies the GitHub signature and kicks off a rebuild for
// pushes to the configured branch. It always answers fast: GitHub times the
// delivery out after 10 seconds, and a build takes minutes.
func HandleGithubWebhook(deployer *Deployer) fiber.Handler {
	cfg := deployer.cfg

	return func(c fiber.Ctx) error {
		if cfg.Secret == "" {
			log.Println("[deploy] GITHUB_WEBHOOK_SECRET is not set, refusing webhook")
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
				"success": false,
				"message": "deploy webhook is not configured",
			})
		}

		body := c.Body()
		if !validSignature(body, c.Get("X-Hub-Signature-256"), cfg.Secret) {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"success": false,
				"message": "invalid signature",
			})
		}

		switch event := c.Get("X-GitHub-Event"); event {
		case "ping":
			return c.Status(fiber.StatusOK).JSON(fiber.Map{
				"success": true,
				"message": "pong",
			})
		case "push":
			// handled below
		default:
			return c.Status(fiber.StatusOK).JSON(fiber.Map{
				"success": true,
				"message": "ignored event: " + event,
			})
		}

		var payload pushPayload
		if err := json.Unmarshal(body, &payload); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"success": false,
				"message": "invalid push payload",
			})
		}

		if cfg.Repository != "" && !strings.EqualFold(cfg.Repository, payload.Repository.FullName) {
			return c.Status(fiber.StatusOK).JSON(fiber.Map{
				"success": true,
				"message": "ignored repository: " + payload.Repository.FullName,
			})
		}

		if payload.Ref != cfg.Ref() {
			return c.Status(fiber.StatusOK).JSON(fiber.Map{
				"success": true,
				"message": "ignored ref: " + payload.Ref,
			})
		}

		// Branch deletions and tag-only pushes carry no code to build.
		if payload.Deleted || payload.After == zeroSHA || payload.HeadCommit == nil {
			return c.Status(fiber.StatusOK).JSON(fiber.Map{
				"success": true,
				"message": "nothing to deploy",
			})
		}

		commit := payload.HeadCommit.ID
		log.Printf("[deploy] push to %s by %s: %s", payload.Ref, payload.Pusher.Name, commit)

		if err := deployer.Trigger(commit); err != nil {
			return c.Status(fiber.StatusOK).JSON(fiber.Map{
				"success": true,
				"message": err.Error() + ", it will pick up this commit",
				"commit":  commit,
			})
		}

		return c.Status(fiber.StatusAccepted).JSON(fiber.Map{
			"success": true,
			"message": "deploy started",
			"commit":  commit,
		})
	}
}

// HandleStatus reports the last deploy. It is signature-free but requires the
// webhook secret as a bearer token so it can be curled from the EC2 box.
func HandleStatus(deployer *Deployer) fiber.Handler {
	cfg := deployer.cfg

	return func(c fiber.Ctx) error {
		token := strings.TrimPrefix(c.Get("Authorization"), "Bearer ")
		if cfg.Secret == "" || !hmac.Equal([]byte(strings.TrimSpace(token)), []byte(cfg.Secret)) {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"success": false,
				"message": "unauthorized",
			})
		}

		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"success": true,
			"data":    deployer.Status(),
		})
	}
}

// validSignature checks GitHub's "sha256=<hex>" HMAC of the raw request body.
func validSignature(body []byte, header, secret string) bool {
	const prefix = "sha256="
	if !strings.HasPrefix(header, prefix) {
		return false
	}

	expected, err := hex.DecodeString(strings.TrimPrefix(header, prefix))
	if err != nil {
		return false
	}

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)

	return hmac.Equal(expected, mac.Sum(nil))
}
