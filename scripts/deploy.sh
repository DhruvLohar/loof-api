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
ENV_FILE=${ENV_FILE:-$HOST_REPO_DIR/.env}
SWAP_IMAGE=${SWAP_IMAGE:-docker:cli}

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
log "building $IMAGE_NAME:$TAG"
docker build -t "$IMAGE_NAME:$TAG" -t "$IMAGE_NAME:latest" "$REPO_DIR"

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

# Old images pile up fast on a small EC2 disk.
docker image prune -f >/dev/null 2>&1 || true

log "done, $IMAGE_NAME:$TAG will be live in a few seconds"
