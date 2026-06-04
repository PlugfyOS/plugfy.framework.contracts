<!-- markdownlint-disable MD013 -->

# Plugfy — The Three Layers (canonical)

> The canonical boundary model for the whole codebase. It defines what belongs in
> each of the three layers, why, and where the line is drawn. Every package lands
> in **exactly one** layer. The relocations that bring the as-built tree into line
> with this model are tracked in
> [`boundary-refactor-backlog.md`](boundary-refactor-backlog.md).
>
> **On the status of these rules.** The architecture and its rules are the *current
> documented decisions*, not frozen law. They are written down precisely so they can
> be **analyzed, discussed, improved, and adjusted** whenever the context or the
> needs change. Where this document states a layer responsibility, a boundary, or a
> dependency direction, read it as "the decision we hold today, and the reasoning
> behind it" — kept open to revision, not declared permanent.

## The ruler

**Three layers, three verbs: Framework DEFINES & RUNS pipelines · Foundations BUILDS apps/services/scripts · Platform SCALES them into a governed ecosystem.**

That single sentence is the canonical project definition and the discriminator for
every package. To place any concept, ask which verb it serves:

- Does it **define or run a pipeline** (the unit contract, the pipeline contract,
  the engine and runner that execute them)? → **L1 Framework**.
- Is it something you need to **build** a complete, modern, manageable app,
  service, or script on the framework (an extension, module, plugin, provider,
  adapter, capability — the SDK, the UI/SDUI engine, the persistence seam, an
  AI/agent contract, a communication module, a connector)? → **L2 Foundations**.
  **"Capability" is a Foundations concept, not a Framework one.**
- Is it host-side **operation that scales those apps/services/agents into a governed
  ecosystem** (enterprise governance, the multi-app platform, marketplace,
  automatic updates, accounts/identity, themes/skins, per-edition config,
  install-as-OS-service, supervision, observability, the micro-kernel
  host-composition/loader)? → **L3 Platform**.

## L1 Framework — DEFINES & RUNS pipelines

L1 is **exactly three concepts — Unit + Pipeline + Execution.** You **define**
units, **compose** them into pipelines, and **execute** those pipelines —
**asynchronously**. It is pure, **stdlib-only**, and domain-agnostic: it knows
nothing of capabilities, providers, persistence, UI, hosts, editions, accounts, or
triggers. With unit + pipeline + execution alone you can define and run a complete,
complex, async pipeline.

The framework **knows nothing domain-specific**: there are **no webhooks, no HTTP,
no gRPC, no WebSockets, no UI, no persistence, no accounts, no triggers** in it.
Those are not framework concerns — **communication modules (gRPC, WebSockets,
HTTP/REST) are Foundations (L2)** and **trigger/webhook hosting is Platform (L3)**.
The framework's remaining domain remnants (`node_llm`/`node_ui` handlers, the
trigger node, the in-engine CEL implementation) are not L1 and are being removed
(wave SW-7); see [`boundary-refactor-backlog.md`](boundary-refactor-backlog.md).

### The two ways to use the framework

The framework is consumed **asynchronously** in exactly two ways:

1. **As a Go library you import** — embed the engine in your own Go program, build
   units and pipelines in code, and run them in-process.
2. **Via the `plugfy` CLI** — pass a pipeline document plus parameters to the
   standalone job runner and execute it from the command line.

Both paths drive the same `core.Unit` = `{Describe, Invoke}` brick, the same
`Pipeline` that composes units recursively, and the same engine + runner.

### The empirical proof: `plugfy run <doc>`

The claim that *unit + pipeline + execution alone* are enough to define and run a
complete, complex, async pipeline is not aspirational — it is demonstrated by the
standalone job runner (the **CLI** path) that depends on nothing but L1:

```
plugfy run <pipeline.v1.json> [--input key=value ...]
```

The standalone job runner + CLI live in `plugfy.framework.runtime`'s **nested
`framework/` module** (the repo's sole module after WAVE SW-6 dissolved the OUTER
module). The per-`Invoke` runner envelope it drives lives in L1
`plugfy.framework.pipeline/runner` (rehomed there in SW-6):

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

> **Documented decision (v1.12.23, #102): today the L1 framework depends on nothing
> from L2 foundation or L3 platform.** This is the dependency direction we currently
> hold — open to analysis and revision if the layering needs to change. Under it, no
> L1 module — `framework.contracts`, `framework.pipeline`, the nested
> `framework.runtime/framework` engine/CLI, or the `runner` — imports
> `github.com/PlugfyOS/plugfy.foundation.*` or `github.com/PlugfyOS/plugfy.platform.*`,
> and `go list -deps` over all four proves ZERO such imports. The two violations SW-3 left in the pipeline were closed: the
> `spi.PipelineEngine` interface no longer embeds the L2 foundation `Provider` (the
> engine's L1 contract is just `Run`; `Name`/`Kind`/`Capabilities`/`HealthCheck`
> survive only as plain descriptor-derived helpers, and `KindPipeline` is the L1
> `core.Kind`), and the pipeline-engine-as-registry-Provider self-registration moved
> OUT of L1 to the composition root (the platform server wiring) — the born-correct
> dependency inversion (L1 provides, L2/L3 wires). `go mod tidy` consequently dropped
> `plugfy.foundation.{registry,sdk}` from the pipeline and the nested framework
> module graphs. The CI decouple-check that fails if any L1 engine package imports
> foundation/platform is now wired (the `plugfy.framework.runtime` CI build job greps
> the engine module for `plugfy.foundation`/`plugfy.platform` and fails on a hit), so
> this decision is machine-checked. The CEL `cel-go` third-party dep in the pipeline
> is a SEPARATE concern, SW-7c, not a foundation/platform module dep.

### The crisp post-relocation L1 surface

With the boundary relocations complete (through SW-6), L1 contains exactly:

| Concern | Package |
|---|---|
| The `unit` contract — `Unit`/`UnitContext`/`UnitDescriptor`/`ParamDef`/`RetryPolicy`/`DefaultUnit` | `contracts/spi/core` |
| Lifecycle support — `LifecycleContext` (which `UnitContext` extends) | `contracts/spi/lifecycle` |
| The `Evaluator` **port** | `contracts/spi/evaluator` |
| The `pipeline` contract — `Pipeline`/`Node`/`Edge`/`NodeType`/`PipelineEngine`/`UnitResolver`/`NodeRunner` + generic collaborators `ModuleDispatcher`/`JobsQueue` | `pipeline/contracts/spi` |
| The pipeline engine — generic nodes + `pipelineunit` + the `domain/pipeline` graph + `errclass` | `pipeline/application/engine`, `pipeline/domain/pipeline` |
| The per-`Invoke` Runner envelope | `pipeline/runner` (rehomed from the framework.runtime OUTER module in SW-6) |
| The standalone job runner + CLI + demo builtin | `runtime/framework` (`cmd/plugfy`, `cli`, `job`, `builtin`) |
| Pure support leaves (zero third-party deps) — `events.CloudEvent`, `errs` error-class, `ids.ULID`, `resilience` reference impl, `idempotency.Store` port | `contracts/events`, `contracts/errs`, `contracts/ids`, `contracts/resilience`, `contracts/idempotency` |

> Note on "contracts, not implementations": L1 is **contracts + stdlib-only
> reference implementations**. `ids` (ULID), `resilience` (Breaker/Retry/Bulkhead),
> and `idempotency` (MemStore) are real implementations, but they have **no
> third-party dependencies** and exist to make unit+pipeline runnable on their own.

## L2 Foundations — BUILDS apps/services/scripts

L2 is everything you need to **build** a complete, modern, manageable app, service,
or script on the framework — UI + backend + all the necessary resources. It extends
the framework's unit/pipeline/execution with **extensions, modules, plugins,
providers, adapters, and capabilities**: a progress reporter, the communication
modules (gRPC, WebSockets, HTTP/REST), persistence, UI/SDUI, the AI/agent contracts,
and the SDK. **A "capability" is an L2 concept**: L1 has no notion of one.
Foundations gives the Platform all the building blocks it needs to scale.

### The two ways to author on Foundations

Foundations supports **two authoring modes**:

1. **Embedded in a Go app** — import the SDK and write a compiled unit, service, or
   app in code.
2. **No/low-code** — declare the app declaratively (the `app.v1` artifact: uischema
   + pipeline + agent facets) and let the framework-as-runtime execute it.

### How apps are built

An app built on Foundations is a **context that calls pipelines defined in that
context** — but **an app is MORE than its pipelines.** A **pipeline is a FLOW for
*executing* things** (internal tasks, integrations, calls to other apps/services,
business rules, validations) — the app's execution/logic layer. The app's **logic**
**is** pipelines — they orchestrate the L2 modules, components, and connectors that
perform the app's tasks, and **pipelines call pipelines** (uniform recursion); the
app writes **no** imperative plumbing and never talks to a database, an HTTP
endpoint, or a queue directly. Every capability a pipeline node invokes is reached
through a **capability *contract*** (the SPI), never a concrete engine: a store
holds the engine-agnostic `persistence.SQLDB` seam, a connector node holds the
connector contract, and so on. **The executor decides** which concrete provider
backs each contract — memory, filestore, SQLite, Postgres, a network adapter — and
**injects** it at the composition root (the same model as .NET Entity Framework's
`DbContext` over interchangeable providers; the data realization is the two-plane
model where the composition root picks SQLite vs Postgres by edition/config). So
building an app's **logic** reduces to **defining its pipelines and declaring the
capabilities they use** — in either authoring mode above, the app code is identical
whichever provider the executor injects. On top of that logic an app **also** has
its **UI/UX/visual surface**, its **artifacts**, and its **other particularities**:
an app = **{ its pipelines (execution/flows) } + { UI/UX } + { artifacts } +
{ particularities }**, the *composition* of all of these, **not pipelines alone**.
Canonical detail:
[`governance.spine/docs/APP-MODEL.md`](https://github.com/PlugfyOS/plugfy.platform.governance.spine/blob/main/docs/APP-MODEL.md)
(companion: [`APP-DELIVERY-MODEL.md`](https://github.com/PlugfyOS/plugfy.platform.governance.spine/blob/main/docs/APP-DELIVERY-MODEL.md) for packaging/hosting,
[`EDB-PERSISTENCE.md`](https://github.com/PlugfyOS/plugfy.platform.governance.spine/blob/main/docs/EDB-PERSISTENCE.md) for the two-plane data seam).

L2 owns:

- **Provider / Kind / registry / unit-manifest** (`plugfy.foundation.registry`,
  re-exported by `foundation.sdk/spi`) — every pluggable provider, the
  provider-category `Kind` + its constants, and the registry that discovers them.
  **Relocated out of L1 here in SW-3 (v1.12.18), completing NR-01's physical move:**
  `Provider`/`Kind` are now defined canonically in the **`plugfy.foundation.registry`**
  module (the registry IS the provider machinery — it registers/builds `Provider`s by
  `Kind`, so the contract and its index are one cohesive unit), and `foundation.sdk/spi`
  re-exports them as aliases for ergonomic authoring (a one-way sdk→registry edge, so
  the registry module imports NO SDK and there is no cycle). They are no longer in
  `contracts/spi/provider.go`, which is deleted. The pure provider **index**
  (`Register`/`Build`/`Names`/`Has` + `Factory`/`Options`) and the universal
  **unit manifest** (`unit.plugfy.com/v1` + validator) live in the new stdlib-light
  Foundation module **`plugfy.foundation.registry`** (+ `/manifest`). The
  supervisor-coupled live **`ServiceIndex`** and on-disk manifest **`Discovery`**
  stay in `plugfy.platform.runtime`'s `registry` package (L3 host machinery) and
  import the L2 index + manifest one-way; they relocate to L3 with the loader in
  SW-5. (A Unit is **not** a Provider: as of v1.12.16 `core.Unit` no longer embeds
  `spi.Provider` — it is the minimal `{ Describe, Invoke }` brick, and
  identity/kind/capabilities/health derive from `Describe()`; `DefaultUnit.Kind()`
  now returns the native L1 composition `core.Kind`, leaving the L2 provider `Kind`
  to a host conversion at the boundary.) This **dissolves the SW-2 transitional
  `runtime/registry → foundation.sdk/api` edge** and the runtime↔sdk module cycle:
  the SDK now imports the L2 registry/manifest, never `plugfy.framework.runtime`.
  The dedicated `plugfy.foundation.capabilities` catalog that will own the DOMAIN
  Kind vocabulary remains a later wave (SW-8 / NR-07; the domain constants ride with
  the type in `sdk/spi` for now).
- **The unit-transport seam** (`plugfy.foundation.transport`, its own
  third-party-leaf Foundation module) — the native/subprocess **`plugin`** tiers
  (Native + go-plugin/gRPC subprocess) + the `Invoker`/`Loader` contract + the
  `InvokeAdapter`, the WASM **`wasm`** runtime (wazero Tier-3), and the generic
  **`supervisorwire`** wire contract (the `plugfy.supervisor.v1` proto + genpb).
  **Relocated here from the `plugfy.framework.runtime` OUTER module in WAVE SW-6
  (v1.12.25)**: these tiers pull third-party runtime libraries (notably wazero)
  that must not sit in the lean L1 engine, and co-locating the supervisor wire
  contract with the local transports that speak it removed the prior L2→L3 edge
  (the `plugin` adapter no longer imports the L3 kernel's proto). The module is a
  pure third-party leaf — it requires ZERO plugfy modules and imports no L1 and no
  L3 package. The L3 kernel re-publishes the generated wire types at its
  established import path as a thin re-export for its domain-service consumers.
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
- **The concrete `EventBus` SPI + adapters** and the **marketplace contract**.
  **Relocated here in SW-3 (v1.12.18):** `EventBus`/`Handler`/`Subscription` are now
  defined natively in `foundation.sdk/spi` (no longer aliased from L1
  `contracts/spi`, whose `eventbus.go` is deleted) — `EventBus` embeds the L2
  `Provider`, so keeping it in L1 while Provider lives in L2 would have inverted the
  layer direction. The L1 pipeline engine consumes only its own NARROW, host-owned
  EventBus port (`pipeline/contracts/spi`, just `Subscribe`), so L1 depends on none
  of the concrete event SPI.
- **The api.v1 route-contribution contract** (`foundation.sdk/api`) — the
  pure-data `RouteSet`/`RouteContribution`/`Route`/`AuthScope` description of the
  HTTP routes a unit contributes to the API host. **Relocated here from L1
  `contracts/api` (SW-2, v1.12.17)**: mounting HTTP routes is a BUILD-an-app
  concern — the unit/pipeline engine never declares or mounts routes — so the
  route-declaration contract belongs in Foundation, not the L1 baseplate. It is
  stdlib-only and imports nothing, so it stays a pure-data leaf the API host and
  any catalogue/OpenAPI generator read. **The SW-2 transitional edge is RESOLVED in
  SW-3:** the registry **index** relocated to `plugfy.foundation.registry` (L2), so
  the index→`foundation.sdk/api` import is now a clean L2→L2 edge. The only residual
  importer of `foundation.sdk/api` from the runtime repo is the L3-bound
  supervisor-coupled `ServiceIndex` (which dials a service's generic Describe), a
  correct L3→L2 direction dissolved when that package relocates to L3 in SW-5.
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

## L3 Platform — SCALES them into a governed ecosystem

L3 is the ecosystem that **scales** everything Foundations builds — apps, services,
**and** agents — into a **governed ecosystem**. It is host-side operation:
enterprise governance, the multi-app platform, the marketplace, automatic updates,
accounts/identity, themes/skins, per-edition config, install-as-OS-service,
supervision, observability, and the micro-kernel host-composition. It owns:

- **`installed` / admissibility / manifest / layout** (`plugfy.platform.installed`)
  — the single home of the compatibility matrix, **relocated here from L1
  `contracts/installed` in WAVE SW-4 / NR-05 (v1.12.19)** because install/update
  admissibility is a SCALE (L3) host concern, not a unit/pipeline contract.
  `system.update` now imports this one matrix and its ~600-line duplicate is
  deleted (BR-07). `RenderPath`/`RenderDeclarative`/`RenderCustom` stay here as
  OPAQUE STRING tokens whose enum meaning the L2 UI engine owns (BR-04 satisfied
  by the opaque-string boundary — no L3→L2 inversion).
- **The micro-kernel loader** (`plugfy.platform.kernel/loader`), the
  **supervisor** (`plugfy.platform.kernel/supervisor`, which IMPLEMENTS the generic
  `plugfy.supervisor.v1` contract — the contract's proto + generated wire types now
  live in L2 `plugfy.foundation.transport/supervisorwire` after WAVE SW-6; the
  kernel re-publishes them at its established `supervisor/contracts/genpb/supervisorv1`
  path as a thin re-export for its domain-service consumers), the **capability resolver
  + reconciler** (`plugfy.platform.kernel/resolver`), and the
  **supervisor-coupled service discovery** (`plugfy.platform.kernel/discovery`
  — the live `ServiceIndex` + on-disk manifest `Discovery`, package renamed
  `registry`→`discovery` to disambiguate from the L2 `plugfy.foundation.registry`
  index it imports one-way). These packages sit at the **module root** (not a
  redundant `kernel/` subdir): the import path is `plugfy.platform.kernel/loader`,
  alongside the R1 packages (`config`/`updater`/`svcmgr`/`obs`/`depsupervisor`).
  The confusing double-"kernel" path (`plugfy.platform.kernel/kernel/*`) SW-5 left
  was flattened in **v1.12.23** (#102, kernel namespace clarification).
  **Relocated here from the `plugfy.framework.runtime`
  outer module in WAVE SW-5 (v1.12.22)**, completing NR-04's host-composition half:
  this machinery is host-side dynamic composition (SCALE), so it belongs in the
  micro-kernel Platform repo, not the framework. The loader imports
  `plugfy.platform.installed` — an L3→L3 edge — and discovery imports the L2
  registry index/manifest — a correct L3→L2 edge. **Inversion-free:** no L1 package
  (contracts / pipeline / the nested `framework/` engine / `runner`) imports any of
  it; the standalone `plugfy run` L1 engine has no kernel dependency. **WAVE SW-6
  (v1.12.25) dissolved the `plugfy.framework.runtime` OUTER module entirely:** its
  `plugin`/`wasm` transport adapters + the supervisor wire contract relocated to L2
  `plugfy.foundation.transport`, and `runner` rehomed to L1
  `plugfy.framework.pipeline`. The former transitional outer-module→L3 edge
  (`plugin/adapter.go` → the kernel's supervisor genpb) is GONE — `plugin` now imports
  the L2 `supervisorwire` proto, and the L3 kernel supervisor imports that same L2
  proto, so both the transport and the supervisor implementation depend on the
  contract DOWN the layers. The repo now contains only the nested L1 `framework/`
  engine.
- **The entire `plugfy.platform.kernel` repo** (relocated here from the Framework
  engine in WAVE R1 / NR-03) — `config`/edition, `updater`/auto-update,
  `svcmgr`/OS-service, `obs`/observability. (The Ollama specialization in
  `depsupervisor` peels to Foundation/AI per BR-03; the generic "ensure dependency
  process X is ready" mechanism stays with the kernel.)
- **Trigger/webhook hosting** (`pipeline/application/trigger`) — cron/webhook/HMAC.
  Trigger hosting is a Platform concern: the framework has no triggers, and the
  communication modules a trigger listens on are Foundations.
- **Accounts/identity, themes/skins, marketplace/distribution, automatic updates,
  per-edition config**, and multi-tenant governance.

### Deployment & decoupling

Each extensible capability is an **independently-versioned, separately-compiled
artifact** (one repo = one module/service/plugin/connector/provider/app/theme/
driver) shipped in **the form that fits its nature** (`process` binary · `wasm` ·
`lib` · `data`/uischema). The **Platform loader** loads them **dynamically by
version + compatibility** — **MVS** (minimal version selection) + the **9-axis
admissibility matrix** — installed **side-by-side** (WinSxS-style), and the
**resolver/reconciler hot-swaps** a unit's version **at runtime, without
rebuilding or restarting the whole** (update-without-rebuild). The **core/host**
— the **Framework engine + contracts** (the linked execution baseplate), the
**kernel/host loader-supervisor**, the **desktop shell**, and the **SDK** — is
the thing that *does* the loading or a compile/link dependency of it; it is
**embedded/updated via the installer/auto-updater, not self-hot-swapped**.
**Concrete loading/hot-swap is Platform (L3):** the L1 Framework only defines the
`UnitResolver` **port** and runs pipelines — it never loads or hot-swaps modules.
Canonical detail:
[`governance.spine/docs/DECOUPLING-DEPLOYMENT.md`](https://github.com/PlugfyOS/plugfy.platform.governance.spine/blob/main/docs/DECOUPLING-DEPLOYMENT.md).

## Concrete examples (where a thing lands)

| Concept | Layer | Why |
|---|---|---|
| `Unit` (`{Describe, Invoke}`), `Pipeline`, the engine + runner, `plugfy run` | **L1** | define & run a pipeline — the three concepts, nothing more |
| A **progress reporter** | **L2** | a module you reach for to BUILD a richer app |
| **gRPC / WebSockets / HTTP-REST** communication modules | **L2** | communication is a build-an-app concern, never the framework |
| **Persistence**, **UI/SDUI**, **AI/agent contracts**, the **SDK**, connectors, capabilities | **L2** | building blocks Foundations gives the Platform |
| **Marketplace**, **automatic updates**, **accounts/identity** | **L3** | scaling apps into a governed ecosystem |
| **Themes / skins**, **per-edition config**, **install-as-OS-service** | **L3** | host-side operation of the ecosystem |
| **Trigger / webhook hosting**, supervision, observability, the micro-kernel loader | **L3** | host-side operation, not framework |

## Why the line falls here

The earlier ruler said "the framework contains only the generic mechanism; domain
category/contract/implementation lives in Foundation/Platform." That is correct but
under-specified — it does not say *how small* the generic mechanism is. The
sharpened ruler answers that positively and minimally: the generic mechanism that
stays in L1 is **unit + pipeline + execution** (the engine + runner that run them),
and the `plugfy run` binary proves that is a complete, self-contained whole.
Everything else — every provider, every capability, every communication module,
every host concern — is one of the other two verbs and relocates accordingly. This
is why webhooks/triggers/gRPC/WebSockets are **not** framework: communication
modules are Foundations (L2), and trigger/webhook hosting is Platform (L3).

The full per-package verdicts, the relocation items (NR-01…NR-08), the
reconciliation with the prior backlog (BR/IMP/DOC), the bug list, and the wave
sequencing live in [`boundary-refactor-backlog.md`](boundary-refactor-backlog.md).
