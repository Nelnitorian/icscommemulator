#!/usr/bin/env sh
set -eu

ROOT_DIR="$(cd "$(dirname "$0")/../../.." && pwd)"
IMAGE_TAG="${1:-icscommemulator-dnp3-master:local}"

docker image build -f "$ROOT_DIR/protocols/dnp3/master/Dockerfile.master" -t "$IMAGE_TAG" "$ROOT_DIR"
