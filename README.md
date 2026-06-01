# plugfy-common

> **L1 — ABI / Contracts (the baseplate of PlugfyOS).** The one module that
> **every** unit and host links. It imports nobody; it is imported by everyone.

[![Layer](https://img.shields.io/badge/layer-L1_ABI-blue)]() [![Deps](https://img.shields.io/badge/deps-stdlib--only-green)]() [![Version](https://img.shields.io/badge/version-1.1.0-informational)]() [![ABI](https://img.shields.io/badge/ABI-frozen-success)]()

PlugfyOS is an **AI Operation Framework** — a guest platform that installs into a
host environment (desktop, server, or cloud tenant) and operates AI agents and
capabilities on top of it. `plugfy-common` is the contract layer at the bottom
of that framework: the normative interfaces everything else agrees on.

## What it is

`plugfy-common` publishes the generic, stable primitives that hold up the
PlugfyOS micro-kernel — the provider SPI, the unit lifecycle, the event-bus
contract, identifiers, the canonical error model, idempotency, and resilience.
It is **stdlib-only** by design: keeping the root of the dependency tree free of
external modules guarantees that the baseplate never drags a domain, a backend,
or a heavyweight dependency into everything above it.

These are **contracts, not implementations**. The micro-kernel core knows only
this module; capabilities, drivers, and apps are self-contained units that
implement these interfaces and self-register at runtime — they are never
compiled into the core. Because the framework is a *guest* in its host, every
external dependency (model, store, identity, event bus, …) sits behind one of
these SPIs so the concrete backend can be selected per edition without the
upper layers knowing which host they landed in.

| Package | Contents |
|---|---|
| `spi` | `Provider`, `Lifecycle` (+ `DefaultLifecycle`, `LifecycleContext`), `EventBus`, and the 14 `Kind*` provider categories (model, embedding, vectorstore, storage, identity, connector, notification, secret, eventbus, database, rag, authorizer, registry, api) — the base SPI that units extend |
| `api` | api.v1 route-contribution contract: `RouteSet` → `RouteContribution` → `Route` with the `AuthScope` enum (none/user/admin). Pure data — what a route-provider returns and the API host mounts; imports **no** `net/http` |
| `installed` | installed-manifest.v1 + system-layout.v1: `InstalledModule`/`InstalledIndex`, the `RenderPath` (declarative/custom, matching the ui-engine enum) and `Compatibility` UX shape, the `PlatformSpine` and `SystemLayout` (the 9 `Area`s), plus parse/validate helpers. The single shape ops-packaging **writes** and platform-api **reads** |
| `persistence` | the dialect-aware data plane `SQLDB`/`Tx`/`Rows`/`Row`/`Result` over the stdlib `database/sql`, with `Dialect` (postgres/sqlite), `Rebind` and the `Now`/`JSONExtract`/`Upsert` fragment helpers, plus the namespaced control-plane `RegistryStore`. Contracts only — drivers stay in provider repos |
| `events` | the `CloudEvent` envelope (CloudEvents 1.0, JSON mode) + 18 canonical `Type*` constants (IAM, runtime, agent, marketplace, jobs, notifications, audit) |
| `ids` | a lexicographically-sortable ULID generator (Crockford base32, 26-char) with a `Prefixed` helper for kind-tagged IDs |
| `errs` | the canonical error model: 9 classes (validation/unauthorized/forbidden/not_found/conflict/rate_limit/upstream/timeout/internal) → HTTP family, stable reverse-DNS codes, structured details, unwrap-aware `Wrap` |
| `idempotency` | the `Store` contract + an in-memory implementation (`MemStore`) for replay protection, keyed on (subject, path, idempotency-key) |
| `resilience` | `Breaker` (circuit breaker), `RetryPolicy` (capped exponential backoff), `Bulkhead` (bounded concurrency), composed into a single `Guard` |

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

## ABI stability (the frozen public surface)

Because every unit pins `^1.x` of this baseplate, an accidental change to an
exported signature would silently break the whole polyrepo. The `abi` package
guards against that: `abi.TestGoldenABI` snapshots the entire exported public
surface — types, struct fields (with JSON tags), interface method sets, function
and method signatures, and typed constant values — of every public package into
the committed golden file `abi/testdata/api.golden`, and fails CI on any drift.

A failing `Golden ABI freeze` step is the signal that a public signature changed.
If the change is intentional and you have weighed its backward-compatibility
impact (a break warrants a major-version bump), regenerate the golden and commit
it alongside the change:

```bash
GOWORK=off go test ./abi -run TestGoldenABI -update
```

The test is stdlib-only (`go/ast`, `go/parser`, `go/types`, `go/importer`), so it
adds no module dependency and the decoupling gate still holds.

## Rule (non-negotiable)

`plugfy-common` is **L1**: the root of the dependency arrow. It **imports only
the standard library** and **no** other `PlugfyOS/*` repo. Any `require` in
`go.mod` or import of a unit **fails CI**. The bar is *standard library*, not
*fewer packages*: the `persistence` contract may import `database/sql` (it is
stdlib), but the concrete **driver** (`pgx`, the SQLite driver) and `net/http`
are third-party / runtime concerns that live in the provider repos implementing
these contracts — never here. Anything with a domain, a schema, or a concrete
backend lives **above**.

## License

Proprietary — see [LICENSE](LICENSE).
