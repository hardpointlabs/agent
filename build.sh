#!/bin/bash
set -euo pipefail

COMMIT=${1:-$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")}
VERSION=${2:-dev}

SHORT_COMMIT=${COMMIT:0:7}
echo "Building version $VERSION (commit $SHORT_COMMIT)"

go build -ldflags="-X github.com/hardpointlabs/agent/config.Version=$VERSION -X github.com/hardpointlabs/agent/config.Commit=$SHORT_COMMIT" -o agent ./main.go
