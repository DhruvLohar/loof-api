#!/bin/sh
# Rebuilds the API from the tip of the deploy branch and swaps the running
# container for the new image.
#
# This runs *inside* the API container, which means it cannot restart itself:
# the moment `docker rm -f` lands, this script's process is gone. So it does the
# slow part (fetch + build) itself, then hands the swap to a short-lived helper
# container that outlives us.
#
# Requirements on the API container:
#   -v /var/run/docker.sock:/var/run/docker.sock   (talk to the host daemon)
#   -v $HOST_REPO_DIR:/repo                        (source to build from)

set -eu

REPO_DIR=${REPO_DIR:-/repo}
HOST_REPO_DIR=${HOST_REPO_DIR:-$REPO_DIR}
BRANCH=${DEPLOY_BRANCH:-main}
IMAGE_NAME=${IMAGE_NAME:-loof-api}
CONTAINER_NAME=${CONTAINER_NAME:-loof-api}
HOST_PORT=${HOST_PORT:-8000}
APP_PORT=${APP_PORT:-8000}
ENV_FILE=${ENV_FILE:-$HOST_REPO_DIR/.env.prod}
SWAP_IMAGE=${SWAP_IMAGE:-docker:cli}
RUN_MIGRATIONS=${RUN_MIGRATIONS:-1}
BUILDER_IMAGE=${BUILDER_IMAGE:-golang:1.26-alpine}

log() { echo "[deploy.sh] $*"; }

log "repo=$REPO_DIR branch=$BRANCH image=$IMAGE_NAME container=$CONTAINER_NAME"

cd "$REPO_DIR"

# The repo is bind-mounted from the host and owned by another uid, which modern
# git refuses to touch without this.
git config --global --add safe.directory "$REPO_DIR"

# A read-only token lets a private repo pull without an SSH key in the image.
if [ -n "${GITHUB_TOKEN:-}" ] && [ -n "${GITHUB_REPOSITORY:-}" ]; then
	git remote set-url origin "https://x-access-token:${GITHUB_TOKEN}@github.com/${GITHUB_REPOSITORY}.git"
fi

log "fetching origin/$BRANCH"
git fetch --prune origin "$BRANCH"
git reset --hard "origin/$BRANCH"
git clean -fd

TAG=$(git rev-parse --short HEAD)

free_space() { df -Ph / 2>/dev/null | awk 'NR==2 {print $4" free, "$5" used"}'; }

sweep() {
	# Images backing a running container must survive: one of them is the image
	# this very script is executing inside of. Docker refuses to remove those
	# anyway, but skipping them keeps the log free of expected failures.
	in_use=""
	for c in $(docker ps -q 2>/dev/null); do
		in_use="$in_use $(docker inspect --format '{{.Image}}' "$c" 2>/dev/null || true)"
	done

	# Stopped containers first: they hold image references that would otherwise
	# block the image sweep below.
	docker container prune -f >/dev/null 2>&1 || true

	docker images --no-trunc --format '{{.ID}}|{{.Repository}}:{{.Tag}}' 2>/dev/null |
		while IFS='|' read -r id ref; do
			case "$ref" in
				"$IMAGE_NAME":latest | "$BUILDER_IMAGE") continue ;;
			esac
			case "$in_use" in
				*"$id"*) continue ;;
			esac
			# Untagged builder stages, superseded $IMAGE_NAME:<sha> tags, and any
			# one-off image someone pulled by hand.
			docker rmi -f "$id" >/dev/null 2>&1 || true
		done

	docker builder prune -af >/dev/null 2>&1 || true
	docker network prune -f >/dev/null 2>&1 || true
	# Unused volumes only; nothing here mounts one, the DB is external.
	docker volume prune -f >/dev/null 2>&1 || true
}

log "sweeping docker before build ($(free_space))"
sweep
log "swept, keeping $BUILDER_IMAGE + $IMAGE_NAME ($(free_space))"

log "building $IMAGE_NAME:$TAG"
docker build -t "$IMAGE_NAME:$TAG" -t "$IMAGE_NAME:latest" "$REPO_DIR"

# Migrate after the build proves the new code compiles, but before the swap, so
# the incoming container never boots against an un-migrated schema. set -eu
# aborts the deploy on failure, which leaves the old container serving.
if [ "$RUN_MIGRATIONS" = "1" ] && [ -d "$REPO_DIR/migrations" ]; then
	missing=
	for v in DB_USER DB_PASS DB_HOST DB_PORT DB_NAME; do
		eval "val=\${$v:-}"
		[ -n "$val" ] || missing="$missing $v"
	done
	if [ -n "$missing" ]; then
		log "ERROR: migrations need these unset vars in $ENV_FILE:$missing"
		exit 1
	fi

	# Same vars the app builds its DSN from. A literal @ # / or ? in DB_PASS
	# corrupts this URL, so that password must be percent-encoded in .env.prod.
	DSN="postgresql://$DB_USER:$DB_PASS@$DB_HOST:$DB_PORT/$DB_NAME?sslmode=${DB_SSLMODE:-require}"

	log "applying migrations"
	goose -dir "$REPO_DIR/migrations" postgres "$DSN" up
	log "migrations up to date"
else
	log "skipping migrations"
fi

# Build succeeded, so the new image is safe to run. Everything below happens in
# a detached helper because it kills the container this script lives in.
RUN_CMD="docker run -d \
	--name $CONTAINER_NAME \
	--restart unless-stopped \
	-p $HOST_PORT:$APP_PORT \
	--env-file $ENV_FILE \
	-v /var/run/docker.sock:/var/run/docker.sock \
	-v $HOST_REPO_DIR:/repo \
	-e REPO_DIR=/repo \
	-e HOST_REPO_DIR=$HOST_REPO_DIR \
	-e IMAGE_NAME=$IMAGE_NAME \
	-e CONTAINER_NAME=$CONTAINER_NAME \
	-e HOST_PORT=$HOST_PORT \
	-e APP_PORT=$APP_PORT \
	$IMAGE_NAME:$TAG"

log "pulling $SWAP_IMAGE for the swap"
docker pull "$SWAP_IMAGE" >/dev/null

log "handing container swap to helper"
docker run -d --rm \
	--name "${CONTAINER_NAME}-swap" \
	-v /var/run/docker.sock:/var/run/docker.sock \
	-v "$HOST_REPO_DIR:$HOST_REPO_DIR" \
	"$SWAP_IMAGE" \
	sh -c "sleep 2; \
		docker rm -f $CONTAINER_NAME >/dev/null 2>&1 || true; \
		$RUN_CMD >>$HOST_REPO_DIR/deploy.log 2>&1 || \
		docker run -d --name $CONTAINER_NAME --restart unless-stopped \
			-p $HOST_PORT:$APP_PORT --env-file $ENV_FILE \
			-v /var/run/docker.sock:/var/run/docker.sock \
			-v $HOST_REPO_DIR:/repo -e REPO_DIR=/repo \
			-e HOST_REPO_DIR=$HOST_REPO_DIR \
			$IMAGE_NAME:latest >>$HOST_REPO_DIR/deploy.log 2>&1"


docker image prune -f >/dev/null 2>&1 || true

log "done, $IMAGE_NAME:$TAG will be live in a few seconds"
