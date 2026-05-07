# Agent

The Agent is a small standalone daemon that runs inside a users' infrastructure and establishes an outbound-only tunnel to the [Hardpoint](https://hardpoint.dev) routing mesh so that serverless workloads from the [SDK](https://github.com/hardpointlabs/sdk) can talk to services in the users' private infrastructure.

## Overview

This is a pure golang project using recent Go and modules. Release management is handled using [goreleaser](https://goreleaser.com/) and performed only via a GitHub Actions workflow. MacOS and Linux are the only supported platforms at present.

## Dev setup

### Requirements:

* Recent go toolchain

## Workflow:

Build: `go build .`

Run locally: `./run_local.sh` (if the agent connects to a relay successfully, this will block until you send it an INT signal)
