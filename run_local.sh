#!/bin/bash

set -euo pipefail

./agent connect --relay localhost:8080 --skip-tls --config ./local.yaml
