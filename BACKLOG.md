# Backlog — plugfy-common

Items from v1.0.0 onward. Labels: `[type][priority][size]`.

## Delivered (v1.0.0)

| ID | Title | Type | Prio | Size | Milestone |
|---|---|---|---|---|---|
| CMN-01 | `spi`: `Provider`, `Kind`, `Lifecycle`, `DefaultLifecycle`, `LifecycleContext`, `EventBus` | FEAT | P0 | M | v1.0.0 |
| CMN-02 | `events`: `CloudEvent` envelope (CloudEvents 1.0) + canonical type constants | FEAT | P0 | S | v1.0.0 |
| CMN-03 | `ids`: ULID generator | FEAT | P0 | S | v1.0.0 |
| CMN-04 | `errs`: canonical error model (class/code/details/wrap) | FEAT | P0 | S | v1.0.0 |
| CMN-05 | `idempotency`: `Store` contract + in-memory `MemStore` | FEAT | P1 | S | v1.0.0 |
| CMN-06 | `resilience`: `Breaker` / `RetryPolicy` / `Bulkhead` / `Guard` | FEAT | P1 | M | v1.0.0 |
| CMN-08 | `decouple-check` in CI (stdlib-only + zero unit imports) | CI | P0 | S | v1.0.0 |

## Delivered (v1.0.1)

| ID | Title | Type | Prio | Size | Milestone |
|---|---|---|---|---|---|
| CMN-07 | Golden ABI test that freezes the exported public signatures (`abi` package + `testdata/api.golden`) | TEST | P1 | S | v1.0.1 |

> CMN-07 is test-only and leaves the published ABI byte-for-byte unchanged, so it
> ships as a patch (v1.0.1) rather than consuming the v1.1.0 milestone.

## Delivered (v1.1.0)

| ID | Title | Type | Prio | Size | Milestone |
|---|---|---|---|---|---|
| CMN-11 | `api`: api.v1 route-contribution contract (`RouteSet` / `RouteContribution` / `Route` / `AuthScope`), pure data, no `net/http` | FEAT | P0 | S | v1.1.0 |
| CMN-12 | `installed`: installed-manifest.v1 + system-layout.v1 (`InstalledModule`/`InstalledIndex`/`PlatformSpine`/`SystemLayout`) with UX `RenderPath` + `Compatibility` and parse/validate | FEAT | P0 | M | v1.1.0 |
| CMN-13 | `persistence`: dialect-aware `SQLDB`/`Tx`/`Rows`/`Row`/`Result` over stdlib `database/sql`, `Rebind` + `Now`/`JSONExtract`/`Upsert` fragments, `RegistryStore` control plane | FEAT | P0 | M | v1.1.0 |
| CMN-14 | `spi`: `KindRegistry` + `KindAPI` Kind constants (additive) | FEAT | P0 | S | v1.1.0 |

> The v1.1.0 contracts are the Wave-0 shared foundation: ops-packaging writes the
> `installed` shape that platform-api reads, route-providers return `api`
> bundles, and the data/control planes program against `persistence`. The change
> is purely additive — the golden ABI diff adds three package blocks and two Kind
> constants and removes nothing — so it is backward-compatible within `^1.x`. The
> charter now permits the stdlib `database/sql` (drivers stay in provider repos);
> see ARCHITECTURE.md.

## Open (tracked on GitHub Issues — the source of truth)

| ID | Title | Type | Prio | Size | Milestone |
|---|---|---|---|---|---|
| CMN-09 | First-class telemetry contract on `LifecycleContext` (`Logger`/`Tracer` golden coverage) | FEAT | P1 | M | v1.2.0 |
| CMN-10 | `traceparent` as a `CloudEvent` extension (cross-unit distributed trace) | FEAT | P2 | M | v1.2.0 |

> Open debts are mirrored here for readability; GitHub Issues remain the source
> of truth. CMN-09 and CMN-10 are planned for v1.2.0 and are not yet filed as
> issues. Tracking follows the
> [Delivery Standard](https://github.com/PlugfyOS/plugfy-platform/blob/main/docs/DELIVERY-STANDARD.md).
