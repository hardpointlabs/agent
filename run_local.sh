#!/bin/bash

set -euo pipefail

./agent --relay localhost:8080 --skip-tls --config ./local.json
