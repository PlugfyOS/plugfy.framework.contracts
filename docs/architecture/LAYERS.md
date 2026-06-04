<!-- markdownlint-disable MD013 -->

# Plugfy — The Three Layers (canonical)

> The canonical boundary model for the whole codebase. It defines what belongs in
> each of the three layers, why, and where the line is drawn. Every package lands
> in **exactly one** layer. The relocations that bring the as-built tree into line
> with this model are tracked in
> [`boundary-refactor-backlog.md`](boundary-refactor-backlog.md).

## The ruler

**Three layers, three verbs: Framework RUNS pipelines · Foundations BUILDS apps · Platform SCALES apps.**

That single sentence is the discriminator. To place any concept, ask which verb it
serves:

- Does it **run a pipeline** (the unit contract, the pipeline contract, the engine
  that executes them)? → **L1 Framework**.
- Is it something you need to **build an app** on the framework (a provider, the
  SDK, the UI engine, the persistence seam, an AI/agent contract, a connector)? →
  **L2 Foundations**. **"Capability" is a Foundations concept, not a Framework one.**
- Is it host-side **operation that scales apps** (per-edition config,
  install-as-OS-service, auto-update, supervision, observability, the micro-kernel
  loader, marketplace/distribution, multi-tenant governance)? → **L3 Platform**.

## L1 Framework — RUNS pipelines

L1 is **only** the `unit` contract + the `pipeline` contract + the minimal engine
that executes a pipeline. It is pure, **stdlib-only**, and domain-agnostic: it
knows nothing of capabilities, providers, persistence, UI, hosts, or editions.

### The empirical proof: `plugfy run <doc>`

The claim that *unit + pipeline alone* are enough to run a complete, complex
pipeline is not aspirational — it is demonstrated by a standalone job runner that
depends on nothing but L1:

```
plugfy run <pipeline.v1.json> [--input key=value ...]
```

The runner lives in `plugfy.framework.runtime`'s **nested `framework/` module** (the
inner of the repo's two go.mod modules):

- `framework/cmd/plugfy/main.go` — the `plugfy` binary entry point.
- `framework/cli/cli.go` — the `run` subcommand: load a `pipeline.v1` document,
  resolve its unit references against the builtin bricks, run it to completion, and
  print the result JSON.
- `framework/job/runner.go` + `document.go` + `context.go` + `sink.go` — the job:
  parse the document, build the graph, execute it through the engine.
- `framework/builtin` — a self-contained `UnitResolver` over demo bricks
  (`upper`/`exclaim`) so the runner is runnable with zero external wiring.

A `pipeline.v1` document plus the two-method `Unit` contract is the entire input;
the engine threads values node→node, honors control-flow nodes, classifies errors,
and emits the final result — with **no** provider registry, persistence, host,
loader, or capability anywhere in the dependency graph. That is the L1 boundary,
proven by a binary you can run.

### The crisp post-relocation L1 surface

After the relocations in the backlog, L1 contains exactly:

| Concern | Package |
|---|---|
| The `unit` contract — `Unit`/`UnitContext`/`UnitDescriptor`/`ParamDef`/`RetryPolicy`/`DefaultUnit` | `contracts/spi/core` |
| Lifecycle support — `LifecycleContext` (which `UnitContext` extends) | `contracts/spi/lifecycle` |
| The `Evaluator` **port** | `contracts/spi/evaluator` |
| The `pipeline` contract — `Pipeline`/`Node`/`Edge`/`NodeType`/`PipelineEngine`/`UnitResolver`/`NodeRunner` + generic collaborators `ModuleDispatcher`/`JobsQueue` | `pipeline/contracts/spi` |
| The pipeline engine — generic nodes + `pipelineunit` + the `domain/pipeline` graph + `errclass` | `pipeline/application/engine`, `pipeline/domain/pipeline` |
| The per-`Invoke` Runner | `pipeline` Runner |
| The standalone job runner + CLI + demo builtin | `runtime/framework` (`cmd/plugfy`, `cli`, `job`, `builtin`) |
| Pure support leaves (zero third-party deps) — `events.CloudEvent`, `errs` error-class, `ids.ULID`, `resilience` reference impl, `idempotency.Store` port | `contracts/events`, `contracts/errs`, `contracts/ids`, `contracts/resilience`, `contracts/idempotency` |

> Note on "contracts, not implementations": L1 is **contracts + stdlib-only
> reference implementations**. `ids` (ULID), `resilience` (Breaker/Retry/Bulkhead),
> and `idempotency` (MemStore) are real implementations, but they have **no
> third-party dependencies** and exist to make unit+pipeline runnable on their own.

## L2 Foundations — BUILDS apps

L2 is everything you need to **build an app** on the framework — UI + backend + all
the necessary resources. It extends L1 with the concrete machinery that L1
deliberately excludes. **A "capability" is an L2 concept**: L1 has no notion of one.

L2 owns:

- **Provider / Kind / registry** (`contracts/spi/provider.go` + `runtime/registry`)
  — every pluggable provider and the registry that discovers them. (A Unit is **not**
  a Provider: as of v1.12.16 `core.Unit` no longer embeds `spi.Provider` — it is the
  minimal `{ Describe, Invoke }` brick, and identity/kind/capabilities/health derive
  from `Describe()`. The physical relocation of Provider/Kind/registry into Foundation
  is the remaining part of backlog NR-01.)
- **Transport adapters** — the native/subprocess plugin tiers (`runtime/plugin`)
  and the WASM runtime (`runtime/wasm`).
- **The capabilities catalog** (NEW Foundation module) — the domain `Kind`/capability
  vocabulary (model/embedding/vectorstore/rag/identity/connector/notification/secret/
  storage/database/authorizer).
- **The persistence seam** (`plugfy.foundation.persistence`, its own stdlib-only
  Foundation module) — `SQLDB`/`MigrationSet`/`RegistryStore`. Relocated out of L1
  `contracts/persistence` (NR-02 / DOC-01, v1.12.13): a pipeline runs with no
  database, and `ApplyMigrations` literally executes DDL, so persistence is a
  capability/adapter seam, not an L1 contract. The engine driver
  (`provider.database`) and every store import it one-way. The data-plane ENGINE is
  an **edition decision** at the platform composition root, not a layer concern:
  `--edition local` opens an embedded, per-unit SQLite data plane (pure-Go modernc,
  **no Postgres child process**) for its in-process durable units, while shared/cloud
  run Postgres — the same `SQLDB`-seam stores run unchanged on either (EDB-F2 #69; see
  `governance.spine/docs/EDB-PERSISTENCE.md`).
- **The concrete `EventBus` SPI + adapters**, the **api route contract**
  (`contracts/api`), and the **marketplace contract**.
- **The agent/AI contracts** (`foundation.sdk/agent`) — the Assistant/Event chat
  surface and the twelve declarative Agent-Hub primitives + resolver. **Relocated
  here from L1 `contracts/agent` (BR-02, v1.12.12)**: this is the canonical home;
  `platform/system.ai` re-sources the catalog from the SDK. The types still import
  the L1 base SPI (`contracts/spi`, Provider/Kind) — the correct L2→L1 direction.
- **The gRPC status wire helper** (`foundation.sdk/grpcstatus`) — the
  `errs.Class`↔gRPC status-code mapping (`Code`, `CodeFor`/`ClassFor`, `Status`,
  `FromError`/`ToError`). **Relocated here from L1 `contracts/grpcstatus` (NR-06,
  v1.12.15)**: it is a transport-binding helper a service reaches for when it
  exposes its operations over gRPC, not a unit/pipeline contract or the engine. It
  imports the L1 error model (`contracts/errs`) one-way — the correct L2→L1
  direction — and stays stdlib-only (it names the canonical gRPC codes locally
  rather than importing `google.golang.org/grpc`), so its move keeps L1 genuinely
  stdlib-only (resolves bug #10's grpcstatus half).
- **The UI engine + SDUI / `RenderPath`** (`foundation.ui.engine`).
- **The CEL `Evaluator` implementation** (`pipeline/application/expr`), the
  **`ModelGateway`** + `node_llm`/`node_ui` handlers, the **action hub**
  (`pipeline/application/action`), and the **MVS version parser**.
- **The SDK** (`foundation.sdk`) — authoring is a BUILD concern; the SDK stays in
  Foundation by design (see the backlog BR-08 ADR). It now also **hosts** the
  canonical agent/AI contracts (above).

## L3 Platform — SCALES apps

L3 is the ecosystem that **scales** those apps — host-side operation. It owns:

- **`installed` / admissibility / manifest / layout** (`contracts/installed`) — the
  single home of the compatibility matrix.
- **The micro-kernel loader** (`runtime/loader`), the **supervisor**
  (`runtime/supervisor`), and the **capability resolver + reconciler**
  (`runtime/resolver`).
- **The entire `plugfy.platform.kernel` repo** (relocated here from the Framework
  engine in WAVE R1 / NR-03) — `config`/edition, `updater`/auto-update,
  `svcmgr`/OS-service, `obs`/observability. (The Ollama specialization in
  `depsupervisor` peels to Foundation/AI per BR-03; the generic "ensure dependency
  process X is ready" mechanism stays with the kernel.)
- **Trigger hosting** (`pipeline/application/trigger`) — cron/webhook/HMAC.
- Marketplace/distribution and multi-tenant governance.

## Why the line falls here

The earlier ruler said "the framework contains only the generic mechanism; domain
category/contract/implementation lives in Foundation/Platform." That is correct but
under-specified — it does not say *how small* the generic mechanism is. The
sharpened ruler answers that positively and minimally: the generic mechanism that
stays in L1 is **unit + pipeline + the engine that runs them**, and the `plugfy run`
binary proves that is a complete, self-contained whole. Everything else — every
provider, every capability, every host concern — is one of the other two verbs and
relocates accordingly.

The full per-package verdicts, the relocation items (NR-01…NR-08), the
reconciliation with the prior backlog (BR/IMP/DOC), the bug list, and the wave
sequencing live in [`boundary-refactor-backlog.md`](boundary-refactor-backlog.md).
