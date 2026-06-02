# Architecture — plugfy-common

Plugfy Framework is an **Operation Framework** — a guest platform that installs into a
host (desktop, server, or cloud tenant) and operates AI agents and capabilities
on top of it. Its runtime is a domain-agnostic micro-kernel that knows only
generic contracts; everything domain-specific is a self-registering unit above.
`plugfy-common` is where those generic contracts live. It is the one fixed point
the whole framework agrees on, so it must stay small, stable, and dependency-free.

## Position

- **Layer:** L1 — ABI / SPI contracts (the baseplate).
- **Dimension:** Foundations.
- **Kind:** `shared-baseplate` — it is **not a unit** (it does not self-register,
  carries no manifest or capability) and **not a host** (it composes nothing). It
  is the primitive ABI every other module links at build time.

## Boundaries

**Does:** define interfaces (the provider SPI, the unit lifecycle, the event bus),
the `CloudEvent` envelope, the pure-data contracts shared across the polyrepo —
`api` (api.v1 route contributions), `installed` (installed-manifest.v1 +
system-layout.v1, including the UX render-path/compatibility shape) and
`persistence` (the dialect-aware `SQLDB`/`Tx` surface over the standard library's
`database/sql`, plus the namespaced `RegistryStore`) — and pure utilities
(`ids` / `errs` / `idempotency` / `resilience`). Nothing more.

**Does not:** concrete backends (Postgres / SQLite / NATS / S3), database
**drivers** (`pgx`, the SQLite driver), HTTP servers/routers (`net/http`), domain
logic, or UI. The `persistence` package names the SQL *contract* over the stdlib
`database/sql`; the driver and the network wiring live in the provider repos that
implement `spi.KindDatabase` / `spi.KindRegistry`. Likewise `api` declares routes
as data and never imports `net/http`. Concrete backends and runtimes live in the
layers above — drivers (L2), runtime (L3), kernel (L4), the API host (L6), and
system services (L7).

## Dependency inversion

The domain defines the port; the adapter implements it. The adapter therefore
depends on **this** contract — never the reverse. The dependency arrow always
points here.

```
L2 platform-provider-*  ──implements──►  spi.* (here)  ◄──defines/uses──  L7 system-*
                                          ▲
                          every host (L3 runtime / L4 kernel / L6 api) also links
```

## Boundary gate (decouple-check)

`scripts/decouple-check.sh` fails the build if:

1. there is any `require` in `go.mod` (it must be **stdlib-only**); or
2. any package imports another `PlugfyOS/*` repo.

This preserves the invariant that the root of the dependency tree is stable and
zero-domain. The bar is *standard-library only*, not *fewer packages*:
`database/sql` is part of the standard library, so the `persistence` contract may
import it. The forbidden surface is third-party modules — concrete database
**drivers** (`pgx`, the SQLite driver) and HTTP machinery (`net/http` in `api`) —
which would add a `require` to `go.mod` and trip rule 1. Those stay in the
provider repos that implement the contracts.

## Versioning

The ABI is stable: any break in a public signature is a **major** bump.
Consumers pin `^1.x`. A golden ABI test (`abi/`, frozen in `testdata/api.golden`)
locks every exported signature, so an accidental breaking change is caught in CI;
a deliberate, additive extension regenerates the golden in the same review (e.g.
the v1.1.0 `api` / `installed` / `persistence` contracts were a purely additive
golden diff). The decouple gate plus `go vet`/`go build` guard the rest.

## Canonical layer rule

A unit at `Lx` depends only on layers `< x`, always through a contract. The
master layer model lives in
[`PlugfyOS/plugfy-platform`](https://github.com/PlugfyOS/plugfy-platform) (see
`docs/PLATFORM-ARCHITECTURE.md`).
