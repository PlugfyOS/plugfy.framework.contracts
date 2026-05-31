# plugfy-common

> **L1 — ABI / Contracts (the baseplate of PlugfyOS).** The one module that
> **every** unit and host links. It imports nobody; it is imported by everyone.

[![Layer](https://img.shields.io/badge/layer-L1_ABI-blue)]() [![Deps](https://img.shields.io/badge/deps-stdlib--only-green)]() [![Version](https://img.shields.io/badge/version-1.0.0-informational)]()

## What it is

`plugfy-common` publishes the generic, stable primitives that hold up the
PlugfyOS micro-kernel — the provider SPI, the unit lifecycle, the event-bus
contract, identifiers, the canonical error model, idempotency, and resilience.
It is **stdlib-only** by design: keeping the root of the dependency tree free of
external modules guarantees that the baseplate never drags a domain, a backend,
or a heavyweight dependency into everything above it.

| Package | Contents |
|---|---|
| `spi` | `Provider`, `Lifecycle` (+ `DefaultLifecycle`, `LifecycleContext`), `EventBus`, and the `Kind*` constants — the base SPI that units extend |
| `events` | the `CloudEvent` envelope (CloudEvents 1.0, JSON mode) + canonical event-type constants |
| `ids` | a lexicographically-sortable ULID generator (Crockford base32) |
| `errs` | the canonical error model (class → HTTP family, stable codes, details, wrap) |
| `idempotency` | the `Store` contract + an in-memory implementation (replay protection) |
| `resilience` | `Breaker`, `RetryPolicy`, `Bulkhead`, composed into a single `Guard` |

## How to consume

```go
import (
    "github.com/PlugfyOS/plugfy-common/spi"
    "github.com/PlugfyOS/plugfy-common/events"
)
```

A driver/provider implements an SPI port and self-registers; a system service
defines its own port and exposes it. Neither imports the other's implementation —
only this contract. The dependency arrow always points here.

> **App authors** normally import the SDK
> ([`plugfy-sdk`](https://github.com/PlugfyOS/plugfy-sdk)), which re-exports these
> contracts behind a single ergonomic surface, rather than depending on
> `plugfy-common` directly.

## The lifecycle contract (`spi.Lifecycle`)

Every runnable unit runs through four ordered phases per execution; embed
`spi.DefaultLifecycle` and override only the ones you need:

| Phase | Purpose |
|---|---|
| `OnInit` | acquire resources (open connections, fetch credentials, prepare buffers) |
| `OnProcessParameters` | validate, normalize, and resolve template expressions in the inputs |
| `OnExecute` | the actual work — receives processed inputs, returns outputs |
| `OnFinalize` | always-runs cleanup (close connections, report metrics, redact outputs) |

Each phase receives a rich `LifecycleContext` carrying the run/unit identity,
tenant scope, structured logger, tracer, scoped state, and credential accessor.

## Build & test

```bash
GOWORK=off go build ./...
GOWORK=off go test -race ./...
bash scripts/decouple-check.sh   # enforces stdlib-only + zero unit imports
```

## Rule (non-negotiable)

`plugfy-common` is **L1**: the root of the dependency arrow. It **imports only
the standard library** and **no** other `PlugfyOS/*` repo. Any `require` in
`go.mod` or import of a unit **fails CI**. Anything with a domain, a schema,
persistence, or a concrete backend lives **above**, never here.

## License

Proprietary — see [LICENSE](LICENSE).
