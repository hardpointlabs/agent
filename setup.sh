#!/bin/bash
# setup.sh - hosted at pkg.hardpoint.dev/setup.sh
# Copyright 2026 Hardpoint Labs. All rights reserved.
#
# Consult the documentation at https://docs.hardpoint.dev for more information

set -euo pipefail

is_container() {
	[ -f /.dockerenv ] || grep -qE '(docker|lxc|containerd)' /proc/1/cgroup 2>/dev/null
}

# Bail if we're running from a container
if is_container; then
	echo "ERROR: Running the setup script inside a container is not supported."
	echo ""
	echo "Please use the dedicated container images instead."
	echo "Consult the docs: https://docs.hardpoint.dev/hardpoint-connect/getting-started/set-up-the-agent"
	exit 1
fi

# Check if we're running as root; if not, check if we have sudo
if [ "$EUID" -eq 0 ]; then
	SUDO=""
else
	if command -v sudo >/dev/null 2>&1; then
		SUDO="sudo"
	else
		echo "This script requires root privileges or sudo"
		exit 1
	fi
fi

# Warn about running as root
if [ "$EUID" -eq 0 ]; then
	echo "Warning: running as root. This is not recommended but will proceed."
fi

# Parse ORG_ID from query string or args
if [ "$#" -lt 1 ] || [ -z "$1" ]; then
	echo "Usage: curl -s https://pkg.hardpoint.dev/setup.sh | bash -s YOUR_ORG_ID"
	exit 1
fi

ORG_ID="$1"

echo "Checking for dependencies"

check_cmd() { command -v "$1" >/dev/null 2>&1; }

missing_pkgs=()

check_cmd curl || missing_pkgs+=("curl")
check_cmd gpg || missing_pkgs+=("gnupg")
# ca-certificates doesn't have an executable to check
dpkg -s ca-certificates >/dev/null 2>&1 || missing_pkgs+=("ca-certificates")
# although we need sudo, we're handling it separately

if [ ${#missing_pkgs[@]} -gt 0 ]; then
	echo "Installing missing packages: ${missing_pkgs[*]}"

	# avoid forcing update if not needed
	if [ ! -f /var/lib/apt/periodic/update-success-stamp ]; then
		$SUDO apt-get update
	fi
	$SUDO apt-get install -y "${missing_pkgs[@]}"
fi

# Add GPG key
if [ ! -f /usr/share/keyrings/hardpoint-archive-keyring.gpg ]; then
	curl -fsSL https://pkg.hardpoint.dev/apt/hardpoint.gpg | $SUDO gpg --dearmor -o /usr/share/keyrings/hardpoint-archive-keyring.gpg
fi

# Add repository
echo "deb [signed-by=/usr/share/keyrings/hardpoint-archive-keyring.gpg] https://pkg.hardpoint.dev/apt/ stable main" | $SUDO tee /etc/apt/sources.list.d/hardpoint.list

# Update apt
$SUDO apt-get update

# Preseed debconf to avoid interactive prompt
echo "hardpointd hardpointd/org_id string ${ORG_ID}" | $SUDO debconf-set-selections

# Install non-interactively
$SUDO DEBIAN_FRONTEND=noninteractive apt-get install -y hardpointd

echo "✓ Hardpoint agent installed with Org ID: ${ORG_ID}"
echo "✓ Config written to /etc/hardpointd/config.yaml"
FINGERPRINT=$(hardpointd fingerprint)
echo "✓ Agent fingerprint is ${FINGERPRINT}"
