# Architecture — plugfy-common

## Position

- **Layer:** L1 — ABI / SPI contracts (the baseplate).
- **Dimension:** Foundations.
- **Kind:** `shared-baseplate` — it is **not a unit** (it does not self-register,
  carries no manifest or capability) and **not a host** (it composes nothing). It
  is the primitive ABI every other module links at build time.

## Boundaries

**Does:** define interfaces (the provider SPI, the unit lifecycle, the event bus),
the `CloudEvent` envelope, and pure utilities (`ids` / `errs` / `idempotency` /
`resilience`). Nothing more.

**Does not:** persistence, HTTP, concrete backends (Postgres / NATS / S3), domain
logic, UI. Those live in the layers above — drivers (L2), runtime (L3), kernel
(L4), the API host (L6), and system services (L7).

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
zero-domain.

## Versioning

The ABI is stable: any break in a public signature is a **major** bump.
Consumers pin `^1.x`. A golden ABI test freezes the exported signatures so an
accidental breaking change is caught in CI.

## Canonical layer rule

A unit at `Lx` depends only on layers `< x`, always through a contract. The
master layer model lives in
[`PlugfyOS/plugfy-platform`](https://github.com/PlugfyOS/plugfy-platform) (see
`docs/PLATFORM-ARCHITECTURE.md`).
