# Hardpoint Connect Protocol Spec

*Version*: 1.0
*Date* April 2026

---

## Overview

This details the spec for the wire protocol used by the agent (otherwise known as `hardpointd`) to establish a tunnel and handle traffic between Hardpoint's managed infrastructure.

The protocol is a simple, framed, text-based state machine sitting on top of [QUIC](https://en.wikipedia.org/wiki/QUIC) primitives.

## Terminology

* Agent, client, `hardpointd`: used interchangeably to refer to a daemon running inside a private network which constitutes one end of a Hardpoint Connect Protocol tunnel
* Relay: the touchpoint within Hardpoint's managed network mesh which constitutes the receiving end of a Hardpoint Connect Protocol tunnel
* Service: TCP-based server running somewhere within the same network as the agent, but not publicly accessible. Agents relay traffic to services when they receive traffic requests from the Hardpoint network

## Details

### Versioning

The protocol heavily uses QUIC primitives. Since QUIC mandates transport encryption, a protocol value is required for NPN to complete properly. The value is set to `hp-<version>` where `<version>` is the latest protocol version, i.e. `hp-1.0`.

Forward & backward compatibility guarantees are not defined at this time.

### Commands

The protocol consists of the following commands:


| Command | Description |
| :--- | :----: |
| `HELLO` | Sent by an agent on startup |
| `WAITPING` | Sent by an agent polling for approval status |
| `OK` | |
| `ERROR` | |

### Framing