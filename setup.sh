#!/bin/bash
# setup.sh - hosted at pkg.hardpoint.dev/setup.sh

set -euo pipefail

# Parse account_id from query string or args
ACCOUNT_ID="${1}"

if [ -z "$ACCOUNT_ID" ]; then
    echo "Usage: curl -s https://pkg.hardpoint.dev/setup.sh | bash -s YOUR_ACCOUNT_ID"
    exit 1
fi

# Add GPG key
curl -fsSL https://pkg.hardpoint.dev/apt/hardpoint.gpg | sudo gpg --dearmor -o /usr/share/keyrings/hardpoint-archive-keyring.gpg

# Add repository
echo "deb [signed-by=/usr/share/keyrings/hardpoint-archive-keyring.gpg] https://pkg.hardpoint.dev/apt/ stable main" | sudo tee /etc/apt/sources.list.d/hardpoint.list

# Update apt
sudo apt-get update

# Preseed debconf to avoid interactive prompt
echo "hardpointd hardpointd/account_id string ${ACCOUNT_ID}" | sudo debconf-set-selections

# Install non-interactively
sudo DEBIAN_FRONTEND=noninteractive apt-get install -y hardpointd

echo "✓ Hardpoint agent installed with account ID: ${ACCOUNT_ID}"
echo "✓ Config written to /etc/hardpointd/config.yaml"
FINGERPRINT=$(hardpointd fingerprint)
echo "✓ Agent fingerprint is ${FINGERPRINT}"