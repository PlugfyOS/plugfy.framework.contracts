# Backlog — plugfy-common

Itens da v1.0.0 em diante. Labels: `[tipo][prioridade][tamanho]`.

| ID | Título | Tipo | Prio | Tam | Milestone |
|---|---|---|---|---|---|
| CMN-01 | `spi`: `Provider`, `Kind`, `Lifecycle`, `LifecycleContext`, `EventBus` | FEAT | P0 | M | v1.0.0 |
| CMN-02 | `events`: envelope `CloudEvent` (CloudEvents 1.0) + helpers `New` | FEAT | P0 | S | v1.0.0 |
| CMN-03 | `ids`: gerador ULID | FEAT | P0 | S | v1.0.0 |
| CMN-04 | `errs`: modelo de erro canônico (code/category/wrap) | FEAT | P0 | S | v1.0.0 |
| CMN-05 | `idempotency`: contrato `Store` + `MemStore` | FEAT | P1 | S | v1.0.0 |
| CMN-06 | `resilience`: `Breaker`/`RetryPolicy`/`Bulkhead`/`Guard` | FEAT | P1 | M | v1.0.0 |
| CMN-07 | Teste golden de ABI (congela assinaturas públicas) | TEST | P1 | S | v1.0.0 |
| CMN-08 | `decouple-check` no CI (stdlib-only + zero import de unit) | CI | P0 | S | v1.0.0 |
| CMN-09 | Contrato de telemetria no `LifecycleContext` (`Logger`/`Tracer`) | FEAT | P1 | M | v1.1.0 |
| CMN-10 | `traceparent` como extension do `CloudEvent` (trace cross-unit) | FEAT | P2 | M | v1.2.0 |
