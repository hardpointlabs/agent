#!/bin/bash

set -euo pipefail

./agent connect --relay localhost:8080 --key-dir ~/.config/hardpointd --skip-tls --config ./local.yaml
