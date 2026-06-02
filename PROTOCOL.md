# Hardpoint Connect Protocol Spec

*Version*: 1.0
*Date* April 2026

---

## Overview

This details the spec for the wire protocol used by the agent (otherwise known as `hardpointd`), SDK and Relay mesh to establish a tunnel and handle traffic between Hardpoint's managed infrastructure.

The protocol is a simple, framed, text-based state machine sitting on top of [QUIC](https://en.wikipedia.org/wiki/QUIC), HTTP and TLS primitives.

## Terminology

Hardpoint Connect consists of 4 principal components:

* [SDK](https://github.com/hardpointlabs/sdk): The initiator of a tunneling request. Presents an identity (e.g. OIDC token) and a named service and exposes a stream for upstream client libraries such as database drivers to transparently communicate with the remote service across an established tunnel
* Relay: the rendezvous point within Hardpoint's managed network mesh. Evaluates whether SDKs with a given presented identity are allowed to access services they're requesting, and blindly forwards traffic if they are
* Agent/`hardpointd`: a daemon running inside a private network. It makes an outbound connnection to the Relay but is not publicly reachable. It's the receiving end of a Hardpoint Connect Protocol tunnel, and the reciprocal component to the SDK. The source code is contained in this repo
* Service: TCP-based server running somewhere within the same network as the agent, but not publicly accessible. Agents relay traffic to services when they receive connection requests from the Hardpoint network

## Details

### Transports

Due to the constraints of various infrastructure components, Hardpoint Connect uses a variety of transports:

```mermaid
flowchart LR
    A[SDK]
    B[Relay]
    C[Agent]
    D[Service]

    A -- HTTPS --> B
    B <-- QUIC --> C
    C -- TCP --> D
```

Connections from SDK -> Relay and Agent -> Relay mandate TLS 1.3. TLS hardens the connection against MITM interception from the open internet, although since the Relay is not a trusted component in our threat model, we use an additional ML-KEM layer on top, once a stream is established between SDK and Agent. Therefore the trust boundary is between SDK and Agent, both of which have their source code available to review.

Given Hardpoint's current emphasis on the DevEx for TypeScript/JS-based applications running in ephemeral environments such as a Vercel function or a Fly.io machine, the SDK -> Relay communication is implemented as a HTTP CONNECT proxy, which affords a level of simplicity and allows ingress hardening with common off-the-shelf tooling. However, as QUIC implementations in server-side JS runtimes improve, the goal is to build end-to-end QUIC support over time. For this reason we've elected not to lean on end-to-end tunneling capabilities of transport protocols like TLS+ClientHello extensions.

### Versioning

The protocol heavily uses QUIC primitives. Since QUIC mandates transport encryption, a protocol value is required for NPN to complete properly. The value is set to `hp-<version>` where `<version>` is the latest protocol version, i.e. `hp-1.0`.

Forward & backward compatibility guarantees are not defined at this time.

### Commands

The protocol consists of the following commands:


| Command | Description |
| :--- | :----: |
| `HELLO` | Sent by an agent on startup |
| `WAITPING` | Sent by an agent polling for approval status |
| `OK` | Acknoledgement with optional payload, depending on state |
| `ERROR` | Indicates something went wrong with the handshake |
| `CONNECT` | Request to connect to a service |

Commands are sent via a dedicated QUIC control stream between Relay and Agent. While the SDK makes `CONNECT` HTTP requests to the Relay, this is not strictly considered part of the command vocabulary.

### Framing

Both control and data messages are framed using a varint-style framing mechanism with 2 equivalent implementations in [TypeScript](https://github.com/hardpointlabs/length-prefixed-stream) and [Go](https://github.com/hardpointlabs/lpstream).

### Startup message flow

The following message sequence occurs on agent startup when it tries to connect and authenticate to the relay mesh:

```mermaid
sequenceDiagram
    participant A as Agent
    participant B as Relay

    A->>B: HELLO

    opt Auth required
        B->>A: SENDPK
        A->>B: ed25519 pubkey
        B->>A: WAIT
        A->>B: WAITPING
    end

    B-->>A: OK
```

#### Startup state transition diagram

```mermaid
stateDiagram-v2
    [*] --> HELLO

    HELLO --> AUTH_REQUIRED: Agent unknown to Relay
    HELLO --> OK: Agent known

    state AUTH_REQUIRED {
        SENDPK --> WAIT: pubkey requested
        WAIT --> WAITPING: wait
        WAITPING --> WAIT: poll
    }

    WAITPING --> OK

    OK --> [*]

    %% error paths
    HELLO --> ERROR: error
    AUTH_REQUIRED --> ERROR: auth failed
    WAITPING --> ERROR: timeout / dead peer
    ERROR --> [*]
```

### Connect message flow

The following [SDK](https://github.com/hardpointlabs/sdk)-initiated message sequence occurs when a remote SDK call requests access to a resource downstream of an agent. For brevity, lines crossing the relay have been elided, however it should be understood that any message between components on either side of the Relay [e.g. between the SDK and the reciprocal Agent] will pass through the Relay.

```mermaid
sequenceDiagram
    participant A as SDK
    participant B as Relay
    participant C as Agent
    participant D as Service

    A->>B: CONNECT <servicename>
    B->>C: CONNECT <host>:<port>
    C-->>A: OK <ML-KEM pubkey>
    A->>C: <ciphertext>
    C->>D: Open connection
    D-->>A: Tunnel Active
```
