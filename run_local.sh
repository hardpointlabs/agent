#!/bin/bash

set -euo pipefail

./agent listen --relay localhost:8080 --skip-tls --config ./local.json
