#!/usr/bin/env bash

if [ -n "$DOCKER_USERNAME" ] && [ -n "$DOCKER_PASSWORD" ]; then
	echo "Login to the docker..."
	echo "$DOCKER_PASSWORD" | docker login -u "$DOCKER_USERNAME" --password-stdin "$DOCKER_REGISTRY"
fi

if [ -n "$GITHUB_TOKEN" ]; then
	# Log into GitHub package registry
	echo "$GITHUB_TOKEN" | docker login docker.pkg.github.com -u docker --password-stdin
	echo "$GITHUB_TOKEN" | docker login ghcr.io -u docker --password-stdin
fi

# prevents git from complaining about unsafe dir. especially when using github actions
git config --global --add safe.directory .

# shellcheck disable=SC2068
exec otelgen $@
