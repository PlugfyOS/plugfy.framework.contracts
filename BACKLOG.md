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

## Open (tracked on GitHub Issues — the source of truth)

| ID | Title | Type | Prio | Size | Milestone |
|---|---|---|---|---|---|
| CMN-07 | Golden ABI test that freezes the exported public signatures | TEST | P1 | S | v1.1.0 |
| CMN-09 | First-class telemetry contract on `LifecycleContext` (`Logger`/`Tracer` golden coverage) | FEAT | P1 | M | v1.1.0 |
| CMN-10 | `traceparent` as a `CloudEvent` extension (cross-unit distributed trace) | FEAT | P2 | M | v1.2.0 |

> Open debts are mirrored here for readability; GitHub Issues remain the source
> of truth. CMN-07 is filed as a GitHub issue (label `tech-debt`); CMN-09 and
> CMN-10 are planned for their target releases (v1.1.0 / v1.2.0) and are not yet
> filed as issues. Tracking follows the
> [Delivery Standard](https://github.com/PlugfyOS/plugfy-platform/blob/main/docs/DELIVERY-STANDARD.md).
