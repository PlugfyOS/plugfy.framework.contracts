<!-- markdownlint-disable MD013 -->

# Plugfy Framework — Boundary-Refactor Backlog (boundaries + implementation)

> Actionable items derived from the boundary audit (framework ⇄ foundation ⇄ platform) and the as-built implementation analysis. Each item is ready to become a GitHub issue. Severity: **P1** (fix), **P2** (important), **P3** (smell/debt).

## The ruler: three layers, three verbs

**Three layers, three verbs: Framework RUNS pipelines · Foundations BUILDS apps · Platform SCALES apps.**

This is the canonical boundary decision and the discriminator for every package below. Each concept lands in exactly one layer.

- **L1 Framework** = ONLY the `unit` contract + the `pipeline` contract + the minimal engine that executes a pipeline. With unit+pipeline alone the engine runs a complete, complex pipeline — proven by the standalone `plugfy run <doc>` job runner in `plugfy.framework.runtime`'s nested `framework/` module (`framework/cmd/plugfy/main.go` + `framework/cli/cli.go` + `framework/job/runner.go`). Pure, stdlib-only, domain-agnostic. It knows nothing of capabilities, providers, persistence, UI, hosts, or editions.
- **L2 Foundations** = everything needed to BUILD an app on the framework — UI + backend + all necessary resources. It extends the framework with providers, adapters, capabilities, the SDK, the UI engine, the persistence seam, AI/agent contracts, and connectors. **"Capability" is a Foundations concept, not a Framework one.**
- **L3 Platform** = the ecosystem that SCALES those apps — host-side operation: per-edition configuration, install-as-OS-service, auto-update, supervision, observability, the micro-kernel host-composition/loader, marketplace/distribution, and multi-tenant governance.

The sharpened ruler supersedes the prior phrasing ("framework contains only the generic mechanism; domain category/contract/implementation lives in Foundation/Platform"): the generic mechanism that STAYS in L1 is now defined positively and minimally as **unit + pipeline + the engine that runs them** — everything else (every provider, every capability, every host concern) relocates out.

### Responsibility map (each concept → exactly one layer)

| Concept | Layer | Where (real package path) |
|---|---|---|
| `unit` contract (Unit/UnitContext/UnitDescriptor/ParamDef/RetryPolicy/DefaultUnit) | **L1** | `contracts/spi/core` |
| lifecycle support (LifecycleContext) | **L1** | `contracts/spi/lifecycle` |
| `Evaluator` PORT | **L1** | `contracts/spi/evaluator` |
| `pipeline` contract (Pipeline/Node/Edge/NodeType/PipelineEngine/UnitResolver/NodeRunner + ModuleDispatcher/JobsQueue) | **L1** | `pipeline/contracts/spi` |
| pipeline engine (generic nodes + pipelineunit + graph + errclass) | **L1** | `pipeline/application/engine`, `pipeline/domain/pipeline` |
| per-Invoke Runner | **L1** | `pipeline` Runner |
| standalone job runner + CLI + demo builtin | **L1** | `runtime/framework` (`cmd/plugfy`, `cli`, `job`, `builtin`) |
| pure support leaves (events.CloudEvent, errs error-class, ids.ULID, resilience reference impl, idempotency.Store PORT) | **L1** | `contracts/events`, `contracts/errs`, `contracts/ids`, `contracts/resilience`, `contracts/idempotency` |
| Provider / Kind / registry | **L2** | `contracts/spi/provider.go` + `runtime/registry` → Foundation |
| transport adapters (native/subprocess plugin tiers + wasm) | **L2** | `runtime/plugin`, `runtime/wasm` → Foundation |
| capabilities catalog (domain Kind/capability vocabulary) | **L2** | **NEW** Foundation `capabilities` module |
| persistence seam (SQLDB/MigrationSet/RegistryStore) | **L2** | `contracts/persistence` → Foundation |
| concrete EventBus SPI + adapters | **L2** | `contracts/spi/eventbus` + adapters → Foundation |
| api route contract | **L2** | `contracts/api` → Foundation |
| agent / AI contracts | **L2** | ✅ DONE — relocated `contracts/agent` → `foundation.sdk/agent` (BR-02, v1.12.12) |
| UI RenderPath / SDUI | **L2** | `foundation.ui.engine` (already there); `RenderPath` leaves `contracts/installed` |
| marketplace contract | **L2** | Foundation/`platform.system.marketplace` (BR-09) |
| CEL `Evaluator` impl | **L2** | `pipeline/application/expr` → Foundation |
| ModelGateway + node_llm/node_ui handlers | **L2** | `pipeline/application/engine` (LLM/UI) + `contracts/spi/collaborators.go` ModelGateway → Foundation |
| action hub | **L2** | `pipeline/application/action` → Foundation |
| MVS version parser (shared) | **L2** | Foundation shared |
| installed/admissibility/manifest/layout | **L3** | `contracts/installed` → Platform |
| micro-kernel loader | **L3** | `runtime/loader` → Platform |
| supervisor | **L3** | `runtime/supervisor` → Platform |
| capability resolver + reconciler | **L3** | `runtime/resolver` → Platform |
| entire kernel repo (config/edition, updater/auto-update, svcmgr/OS-service, obs/observability) | **L3** | `plugfy.platform.kernel` (relocated, NR-03 ✓) |
| trigger hosting (cron/webhook/HMAC) | **L3** | `pipeline/application/trigger` → Platform |

> Original ruler (kept for traceability): *the framework contains only the **generic mechanism**; domain category/contract/implementation lives in Foundation/Platform; generic mechanism found in Foundation/Platform descends to the framework.* The map above is the sharpened, positive form.

## NR — Sharpened relocations

The relocations that crisp L1 down to **unit + pipeline + engine**. These are sharper than the existing BR items (which they supersede/sharpen — see the annotations on each BR below). Priorities: **P1** (foundational, high blast radius), **P2** (structural), **P3** (consolidation/hygiene).

### NR-01 (P1) · Relocate Provider/Kind/registry out of L1 to Foundation
- **Where:** `contracts/spi/provider.go` (`Provider`, `Kind`, `CapabilityRequirement`), `runtime/registry`.
- **What:** Provider, the provider-category `Kind`, and the registry are L2 (Foundations) concepts — every Unit is a Provider *today* only because `core.Unit` embeds `commonspi.Provider` (`spi/core/unit.go:16-17`). **BREAK that coupling first** — un-embed `spi.Provider` from `core.Unit` (highest blast radius: it changes what a Unit *is*) — then move Provider/Kind/registry to Foundation.
- **Sequencing:** the un-embed precedes the registry move; both land in wave **R5**.
- **ABI:** breaking on `spi/core` and `spi/provider`. Golden ABI re-scoped by NR-08.
- **Accept:** `core.Unit` no longer embeds `spi.Provider`; `contracts` exports no Provider/Kind/registry; Foundation owns the provider contract + registry.

### NR-02 (P1) · Relocate the persistence seam to Foundation
- **Where:** `contracts/persistence` (`SQLDB`/`MigrationSet`/`RegistryStore`, `persistence.go`/`registry.go`/`migrate.go`).
- **What:** the persistence seam is an L2 (Foundations) resource, not an L1 contract. Move the whole `contracts/persistence` package to Foundation.
- **Sequencing:** coordinate with in-flight **EDB-F2 (#69)** — every just-migrated store imports this package, so do the move **AFTER the #69 store cutovers land**, as **one atomic re-import + retag + golden-regen**. Wave **R3** (waits for EDB-F2). Subsumes DOC-01's `migrate.go` relocation.
- **ABI:** breaking (import path moves for every store).
- **Accept:** `contracts` no longer ships `persistence`; all stores import the Foundation persistence module; goldens regenerated; `go test` green across the touched repos.

### NR-03 (P1) · Relocate the entire kernel repo to Platform L3 — DONE (WAVE R1)
> **DONE.** The kernel repo is relocated to the Platform tier as `plugfy.platform.kernel` (module `github.com/PlugfyOS/plugfy.platform.kernel`, first tag **v1.12.11**). It had **zero** Go importers (already a dependency leaf), so no consumer re-pin was needed; the L1 `plugfy run` smoke and the clean-cache qa/smoke gate stayed green. The residual item is the Ollama-specialization peel to Foundation/AI, tracked as BR-03 (P3 follow-up).
- **Where:** `plugfy.platform.kernel` (config/edition, updater/auto-update, svcmgr/OS-service, obs/observability, depsupervisor).
- **What:** the whole kernel repo is host-side **operation** (SCALE) — per-edition config, auto-update, install-as-OS-service, observability — i.e. L3 Platform. Relocated as a Platform repo. The **Ollama specialization peels to Foundation/AI** (the generic "ensure dependency process X is ready" mechanism stays; the LLM/embedding specifics go to the AI domain) — see BR-03.
- **Sequencing:** **lowest L1 coupling — moved early.** Wave **R1** ✓.
- **Accept (met):** no `plugfy.framework.*` repo carries edition/updater/svcmgr/obs; the kernel lives under Platform (`plugfy.platform.kernel`). Ollama-to-Foundation/AI remains as BR-03.

### NR-04 (P1) · Relocate the runtime repo's OUTER micro-kernel module to L2/L3
- **Where:** `plugfy.framework.runtime` — the OUTER go.mod (the repo root: `loader`, `supervisor`, `resolver`/reconciler, `plugin` tiers, `wasm`, `manifest`).
- **What:** the runtime repo has TWO go.mod modules (an undocumented L1/L3 seam — bug #9): the **nested `framework/` module** (the pipeline engine + job runner + demo builtin) is the **L1** core and STAYS; the **outer module** is host machinery — transport adapters (`plugin`, `wasm`) → **L2(adapters)**, loader + supervisor + resolver/reconciler + manifest → **L3**. Split the repo along that seam.
- **Accept:** L1 = only the nested `framework/` engine + runner; loader/supervisor/resolver land in Platform; plugin/wasm adapters land in Foundation.

### NR-05 (P2) · Relocate `contracts/installed` to L3
- **Where:** `contracts/installed` (`admissibility.go`, `manifest.go`, `layout.go`, `hostmanifest.go`).
- **What:** installed/admissibility/manifest/layout is install/update **operation** (L3 Platform). Move it to Platform, **consolidating BR-07's duplicated matrix** into that single L3 home. `RenderPath` peels to L2 UI (BR-04). Wave **R4**.
- **Accept:** `contracts` no longer ships `installed`; the admissibility matrix has one home in the L3 install/update domain; `system.update` imports it (resolves BR-07).

### NR-06 (P2) · Relocate the L2 leaf contracts + engine domain handlers to Foundation
- **Where:** `contracts/api`, `contracts/agent`, the concrete `eventbus` SPI + adapters, `contracts/grpcstatus`, `pipeline/application/action`, `pipeline/application/trigger`, `pipeline/application/expr` (CEL Evaluator impl), `ModelGateway` (`spi/collaborators.go`), `pipeline/application/engine/node_llm.go` + `node_ui.go`.
- **What:** these are domain/capability contracts + concrete handlers — L2 Foundations (trigger hosting is L3, see NR's map; folded under R6 de-domain + R1 host machinery). Wave **R2** (leaf contracts → L2) + **R6** (engine de-domain: CEL impl, ModelGateway, LLM/UI nodes leave the engine).
- **Accept:** `contracts` exports only L1 leaves; the engine's node set is purely generic (no domain-named node type); CEL/ModelGateway live in Foundation.

### NR-07 (P3) · Create the explicit Foundation capabilities catalog module
- **Where:** **NEW** `plugfy.foundation.capabilities` (or equivalent).
- **What:** the domain `Kind`/capability vocabulary (model/embedding/vectorstore/rag/identity/connector/notification/secret/storage/database/authorizer) — frozen today in L1's `spi.Kind` enum — moves to an explicit Foundation catalog module that OWNS the domain capability vocabulary. Wave **R7**.
- **Accept:** no domain category name appears in `contracts`; providers + consumers reference the Foundation catalog.

### NR-08 (P3) · Re-scope the golden ABI test to the crisp L1 surface
- **Where:** `contracts/abi/abi_test.go` (`surfacePackages`), `contracts/abi/testdata/api.golden`.
- **What:** after the relocations, freeze the golden over ONLY the crisp post-relocation L1 surface (unit/pipeline/lifecycle/evaluator + pure leaves); drop the relocated packages (provider/persistence/installed/api/agent) from the golden. Wave **R7**.
- **Accept:** the golden's `surfacePackages` matches the post-relocation L1 surface exactly; `go test ./abi` green.

## A. Fronteiras — tirar domínio do framework

### BR-01 (P1) · Tornar `Kind` opaco; mover as categorias de domínio para um catálogo no Foundation
> **SUPERSEDED by [NR-01](#nr-01-p1--relocate-providerkindregistry-out-of-l1-to-foundation) + [NR-07](#nr-07-p3--create-the-explicit-foundation-capabilities-catalog-module).** The sharpened ruling is stronger than "make `Kind` opaque": eject the **entire** Provider/Kind/registry surface from L1 to Foundation (not just the opaque domain `Kind`), with the explicit capabilities catalog (NR-07) owning the domain vocabulary. NR-01 un-embeds `spi.Provider` from `core.Unit` first.
- **Onde:** `contracts/spi/provider.go:17-39` (enum com 14 valores).
- **Problema:** `model/embedding/vectorstore/rag/identity/connector/notification/secret/storage/database/authorizer` são categorias de domínio congeladas na ABI de um L1 "domain-agnostic / open-ended". Adicionar categoria exige editar o L1 + regenerar o golden → não é open-ended.
- **Mudança:** manter `type Kind string` (opaco) + `Provider`/`registry` no framework. Criar um **catálogo de capacidades no Foundation** (ex.: `plugfy.foundation.capabilities`) que declara as constantes de categoria de domínio. **Manter no L1** apenas os `Kind` que o próprio mecanismo do framework consome: `eventbus`, `database`, `registry`, `api`.
- **Aceite:** nenhum nome de categoria de domínio aparece em `contracts`; providers e consumidores referenciam o catálogo do Foundation; golden ABI do L1 não contém mais as constantes de domínio.
- **Risco/contraponto:** perde-se o "vocabulário num lugar só" do L1 — mitigado pelo catálogo único no Foundation.

### BR-02 (P2) · Reposicionar `contracts/agent` para o domínio de IA — ✅ DONE (R2 warm-up, v1.12.12)
> **DONE.** The `agent` contract has been relocated out of L1 `contracts` into the L2 Foundation SDK (`foundation.sdk/agent`), which now holds the canonical type definitions (previously an alias re-export). `framework.contracts/agent` is deleted; `platform/system.ai/contracts/spi` re-sources the catalog from `foundation.sdk/agent`; the public surface is byte-identical so every downstream author is untouched. This **resolves IMP-03** (the public-but-ungolden `agent` package left L1). This was the lowest-risk warm-up sub-move that validates the relocation mechanics reused by the bigger R2 waves.
- **Onde (antes):** `framework.contracts/agent/{ai.go,hub.go}` (Assistant + 12 primitivas Agent-Hub). **Agora:** `foundation.sdk/agent/{ai.go,hub.go}`.
- **Problema (resolvido):** contrato de IA num L1 que se diz agnóstico; além disso era **público mas fora do golden** (ver IMP-03).
- **Mudança (feita):** definições reais movidas para o SDK do Foundation; os tipos de agente importam `framework.contracts/spi` (Provider/Kind) — esse import L2→L1 é a direção correta (a base SPI permanece em L1 até R5). `system.ai` permanece o receptor real (ADK-Go + Agent Hub) e apenas re-fonteia o contrato pelo SDK.
- **Aceite (atingido):** `contracts` não exporta mais tipos de agente; autores declaram `AgentDef` importando `foundation.sdk/agent`.
- **Contraponto (resolvido):** o motivo de estar no L1 era deixar o autor declarar agente sem depender de `system.ai`; isso continua verdadeiro — o autor agora importa o **Foundation SDK** (que já importava de qualquer forma), sem nenhuma dependência em `system.ai`.

### BR-03 (P3) · Tirar o dep-supervisor do Ollama do kernel — remaining follow-up after NR-03
> **NR-03 DONE; BR-03 is the residual P3 peel.** The kernel repo has relocated to Platform L3 as `plugfy.platform.kernel` (NR-03 ✓). The generic dep-supervisor mechanism now lives there (L3); what remains is peeling the **Ollama specialization** to Foundation/AI — a P3 hygiene follow-up, tracked as its own issue, NOT a blocker on R1.
- **Onde:** `plugfy.platform.kernel/depsupervisor/ollama.go`; `plugfy.platform.kernel/config` `ModelConfig{OllamaHost,OllamaAuto}`.
- **Problema:** o kernel genérico sabe achar/subir/baixar um servidor LLM e um modelo de embedding — domínio de IA hard-coded.
- **Mudança:** o `depsupervisor` deveria ser um **mecanismo genérico** ("garanta que o processo de dependência X esteja pronto") e a especialização Ollama virar uma unidade/extensão do domínio de IA. Tirar `ModelConfig` Ollama do god-config do kernel.
- **Aceite:** `kernel` não menciona Ollama/embedding; a garantia de Ollama vive no domínio de IA.

### BR-04 (P3) · `RenderPath` (UI) fora do `installed`
> **SHARPENED — `installed` itself leaves L1.** Beyond peeling `RenderPath` to L2 UI: the whole `contracts/installed` package relocates to **L3 Platform** ([NR-05](#nr-05-p2--relocate-contractsinstalled-to-l3)). `RenderPath`/`UISchema` peel to the L2 UI domain on the way out; the residual admissibility/manifest/layout lands in L3.
- **Onde:** `contracts/installed` (`RenderPath` declarative/custom, espelha o enum do ui-engine).
- **Problema:** conceito de UI no L1; dono canônico é `foundation.ui.engine`.
- **Mudança:** mover `RenderPath`/`UISchema` para o domínio de UI; o `installed` mantém só a admissibilidade genérica.
- **Aceite:** `installed` não referencia conceitos de renderização de UI.

### BR-05 (P3) · Parametrizar o repositório de release do updater
> **SUPERSEDED by [NR-03](#nr-03-p1--relocate-the-entire-plugfyframeworkkernel-repo-to-platform-l3).** The updater leaves L1 entirely (with the whole kernel → L3), so the hard-coded `plugfy.platform.server` default (bug #6) ceases to be an L1 boundary violation; parametrizing the release source remains a correctness fix to make in its new L3 home.
- **Onde:** `kernel/updater/updater.go:29` (`repoName = "plugfy.platform.server"`).
- **Problema:** kernel genérico nasce sabendo o nome de um daemon específico.
- **Mudança:** sem default de repo embutido; exigir `SetReleaseSource` (já existe `ErrUpdateSourceNotConfigured`).
- **Aceite:** o kernel não cita `plugfy.platform.server` no código de produção.

### BR-06 (P3) · Engine de pipeline sem tipos de nó de domínio
> **SHARPENED to mandatory (R6 de-domain).** No longer "evaluate": the `node_llm`/`node_ui` handlers + `ModelGateway` **leave the engine** for Foundation ([NR-06](#nr-06-p2--relocate-the-l2-leaf-contracts--engine-domain-handlers-to-foundation)). The engine's node set becomes purely generic (bug #8: the closed LLM/UI switch in the "agnostic" engine is removed); domain capability nodes delegate through a generic Module node.
- **Onde:** `pipeline/application/engine/node_llm.go`, `node_ui.go`; `contracts/spi/collaborators.go` (`ModelGateway`).
- **Problema:** o engine "agnóstico" tem tipos de nó nomeados por domínio (LLM/UI) num switch fechado.
- **Mudança:** as **ports** ficam (corretas); avaliar transformar LLM/UI em nós `Module` que delegam a uma capacidade, em vez de tipos de nó dedicados — mantendo o conjunto de nós puramente genérico.
- **Aceite:** adicionar uma "capacidade de domínio" não exige editar o switch de tipos de nó do engine.

## B. Fronteiras — Platform/Foundation consumir ou ceder ao framework

### BR-07 (P1) · Eliminar a duplicação da admissibilidade em `system.update`
> **Still valid; direction changes.** With [NR-05](#nr-05-p2--relocate-contractsinstalled-to-l3) moving `contracts/installed` to **L3 Platform**, the single home of the admissibility matrix becomes the **L3 install/update domain** (not the L1 `contracts`). `system.update` imports/re-exports that L3 package; the ~600 duplicated lines (bug #5) are deleted either way. Remains a standalone P1.
- **Onde:** `platform/system.update/domain/matrix.go` (+`range.go`/`version.go`/`hostos.go`) e `system.update/contracts/spi/compatibility.go` re-declaram a matriz de 9 eixos + tipos de compat que já são canônicos em `framework/contracts/installed`. `system.update` **não importa** o pacote do framework.
- **Evidência:** o próprio framework documenta a dívida — `installed/admissibility.go:184-188` ("private copy … SHOULD be retired by re-exporting").
- **Mudança:** `system.update` passa a importar/re-exportar `contracts/installed`; apagar ~600 linhas duplicadas.
- **Aceite:** `system.update` não tem cópia da matriz; `go test` verde nos dois; uma única fonte de verdade.

### BR-08 (P2) · Avaliar SDK de autoria como mecanismo do framework
> **RESOLVED — SDK stays Foundation (record ADR).** Under the run/build/scale ruler the SDK is a **BUILD** concern (authoring units/apps), which is L2 Foundations by definition. Decision: keep `foundation.sdk` in Foundation; do **not** sink it into L1. Capture the rationale in an ADR (the L1 surface is unit+pipeline+engine only; authoring ergonomics live one layer up so they can evolve on the fast clock).
- **Onde:** `foundation.sdk` (`unit` builder, `capability.Provide/Resolve`, `conformance`).
- **Problema/oportunidade:** são mecanismo genérico (resolver provider, validar qualquer Unit) que hoje vive no Foundation.
- **Mudança (decisão):** decidir explicitamente entre (a) mover para um SDK do framework, ou (b) manter no Foundation por velocidade de evolução. Documentar a escolha.
- **Aceite:** decisão registrada em ADR.

### BR-09 (P3) · Dar casa própria ao contrato `marketplace`
> **Unchanged.** The marketplace contract gets its own home (`platform.system.marketplace` or a dedicated module) and the import cycle is fixed at the source. Consistent with the ruler (marketplace = a SCALE/distribution concern, L3/L2 — never the SDK).
- **Onde:** `foundation.sdk` (colocado lá "para quebrar um ciclo de import").
- **Problema:** "colocado-aqui-para-quebrar-ciclo" é cheiro de fronteira.
- **Mudança:** mover o contrato para `platform/system.marketplace` (ou módulo próprio) e resolver o ciclo na origem.
- **Aceite:** SDK não hospeda o contrato de marketplace.

## C. Implementação (as-built)

### IMP-01 (P1) · Lacuna semântica: `Try`/`Parallel`/`ForEach` não executam corpo/sub-grafo
> **Unchanged — an L1 engine bug.** Stays in L1 (it is the engine that runs pipelines); fix in place. Not a boundary item.
- **Onde:** `pipeline/application/engine/nodes_control.go:41` (`runTry` → `resolveInputs`), `nodes.go:190` (`runParallel` branches = `resolveInputs`), `nodes.go:131`/`nodes_control.go:127` (`runForEach` = `resolveInputs`). Só `Pipeline` recursa (`nodes_control.go:69`).
- **Problema:** os nomes prometem controle-de-fluxo-sobre-um-corpo que não existe; para executar N sub-fluxos em paralelo é preciso compor `Parallel`+`Pipeline` — não óbvio.
- **Mudança:** ou (a) implementar execução de sub-grafo aninhado nesses nós, ou (b) **renomear/documentar** claramente que são "resolução de expressões", não execução de corpo. Não deixar a semântica implícita.
- **Aceite:** comportamento e nome coincidem; doc/exemplos refletem a realidade.

### IMP-02 (P2) · `framework/builtin` só tem bricks demo
> **Reframed by the ruler.** The **demo resolver STAYS in L1** — it is exactly what the standalone `plugfy run` proof needs (a self-contained `UnitResolver` over builtin bricks), and L1 must remain runnable on its own. The **production resolver** (install-root/registry-backed) is an **L3** concern (it depends on the loader/registry, which relocate to Platform). So: keep the demo in L1 labeled as such; build the production resolver in L3.
- **Onde:** `runtime/framework/builtin` (`upper`/`exclaim`).
- **Mudança:** implementar a resolução de produção (install-root/registry) que satisfaz o mesmo `UnitResolver`, ou marcar explicitamente como exemplo.
- **Aceite:** existe um resolver de produção real, ou o demo está rotulado como tal.

### IMP-03 (P2) · `agent` público fora do golden ABI — ✅ DONE (closed by BR-02, v1.12.12)
> **RESOLVED by BR-02** (the R2 warm-up agent relocation, landed v1.12.12). `contracts/agent` was moved to the L2 Foundation SDK (`foundation.sdk/agent`) and deleted from L1, so the "public-but-ungolden L1 package" gap is gone: the L1 golden's `surfacePackages` never listed `agent`, and now no public `agent` package exists in `contracts` at all. The golden remained byte-unchanged through the move (confirmed: `go test ./abi` green, `agent` absent from `surfacePackages`). [NR-08](#nr-08-p3--re-scope-the-golden-abi-test-to-the-crisp-l1-surface) still re-scopes the golden to the crisp post-relocation L1 surface as the remaining R7 cleanup.
- **Onde:** `contracts/abi/abi_test.go:65-77` (surfacePackages — 11 pacotes; `agent` ausente — agora também fisicamente ausente do módulo).
- **Problema (resolvido):** a garantia "ABI frozen" não cobria um pacote público exportado.
- **Mudança (feita):** `agent` despublicado de L1 e movido para o Foundation SDK (casou com BR-02).
- **Aceite (atingido):** todo pacote público de `contracts` está no golden; o `agent` saiu do L1.

### IMP-04 (P3) · Classificação de erro por substring de string
> **Unchanged — an L1 engine bug (bug #7).** `errclass` stays in L1; replace substring routing with `errors.Is`/`ErrorClass()`. Not a boundary item.
- **Onde:** `pipeline/.../errclass.go` (`IsTimeout/IsCancel/IsTransient` por `strings.Contains` minúsculo).
- **Problema:** frágil e dependente de texto/locale.
- **Mudança:** classificar por `errors.Is`/`ErrorClass()` (já parcialmente usado) e remover o fallback por substring, ou restringi-lo.
- **Aceite:** roteamento de erro não depende do texto da mensagem.

### IMP-05 (P3) · `StepFrame` só in-memory; `JobsQueue` sem implementação
> **Unchanged — an L1 engine concern.** The `JobsQueue` PORT stays in L1 (a generic collaborator of the engine); any production adapter is an L2/L3 implementation behind the port. Keep the L1 frontier at zero-persistence: the persistent StepFrame sink and the `JobsQueue` adapter live outside L1.
- **Onde:** `pipeline` StepFrame/FrameSink; porta `JobsQueue` em `collaborators.go` sem adapter de produção (nó `AwaitJob` aspiracional).
- **Mudança:** sink persistente opcional para StepFrame (preservando a fronteira de zero-persistência) e/ou marcar `AwaitJob`/`JobsQueue` como não-suportado até existir um adapter.
- **Aceite:** observabilidade persistível disponível; nós sem backend são sinalizados.

## D. Consistência de documentação

### DOC-01 (P3) · Alinhar "contracts, not implementations" com a realidade
> **SHARPENED — relocate `persistence/migrate.go` to L2 (subsumed by [NR-02](#nr-02-p1--relocate-the-persistence-seam-to-foundation)).** The whole `contracts/persistence` package (including `migrate.go`, the most "implementation"-like — bug #4: it runs DDL inside the "contracts" module) moves to Foundation. For the residual L1 leaves that ARE reference implementations (`ids` ULID, `resilience` Breaker/Retry/Bulkhead, `idempotency` MemStore), keep them in L1 but **rewrite the module description** to "contracts + stdlib-only reference implementations" (they are part of the unit/pipeline support surface and have zero third-party deps).
- **Onde:** README do `contracts` vs `ids` (ULID), `resilience` (Breaker/Retry/Bulkhead), `idempotency` (MemStore), `persistence/migrate.go` (ApplyMigrations roda DDL).
- **Problema:** o L1 contém implementações reais; a frase "contracts, not implementations" é imprecisa.
- **Mudança:** reescrever a descrição para "contratos + implementações de referência stdlib-only", ou mover `persistence/migrate.go` (a mais "implementação") para fora do L1.
- **Aceite:** a descrição do módulo bate com o conteúdo.

> Nota: o golden ABI hoje está **verde** (`go test ./abi` → ok). O item de "golden vermelho" de auditorias anteriores está **resolvido**.

## Bugs found

Defects surfaced while reading the as-built L1. Each is filed as a discrete bug-backlog line; several are subsumed by an NR relocation (the relocation is the right place to fix the root cause), the rest are standalone correctness fixes.

| # | Bug | Where | Disposition |
|---|---|---|---|
| 1 | **Two unrelated `Kind` types in L1** — `core.Kind` (composition ROLE: tool/agent/app/…) vs `spi.Kind` (provider CATEGORY: model/embedding/…). Same name, different meaning, both exported from L1. | `spi/core/descriptor.go:9`, `spi/provider.go:14` | **Rename `core.Kind` → `Role`** (it is a role tag); `spi.Kind` then relocates with Provider (NR-01). |
| 2 | **`core.Unit` embeds L2-bound `spi.Provider`** — every Unit IS a Provider today, coupling the L1 brick to an L2 concept. | `spi/core/unit.go:16-17` | Subsumed by **NR-01** (un-embed first, then relocate Provider). |
| 3 | **Legacy 4-hook `spi.Lifecycle` + `DefaultLifecycle`** parallel to `core.Unit`+`DefaultUnit` — two competing brick contracts in L1; likely dead. | `spi/lifecycle.go:31-36,183-192` | **Verify usage and delete** the legacy `Lifecycle`/`DefaultLifecycle` (keep `LifecycleContext`, which `UnitContext` extends). |
| 4 | **`ApplyMigrations` runs DDL inside the "contracts" module** — a real implementation in a contracts-tier package. | `contracts/persistence/migrate.go` | Subsumed by **NR-02 / DOC-01** (the whole persistence seam relocates to L2). |
| 5 | **~600 lines of admissibility matrix duplicated** in `system.update` (the 9-axis compat matrix re-declared, framework package not imported). | `platform/system.update/domain/matrix.go` (+`range.go`/`version.go`/`hostos.go`) | Subsumed by **BR-07 / NR-05** (single L3 home, delete the copy). |
| 6 | **Updater hard-codes `plugfy.platform.server`** as the release repo default — generic kernel born knowing a specific daemon. | `kernel/updater/updater.go:29` | Subsumed by **NR-03 / BR-05** (kernel→L3; require `SetReleaseSource`). |
| 7 | **`errclass` substring-based routing** — `IsTimeout/IsCancel/IsTransient` via `strings.Contains`, fragile and locale-dependent. | `pipeline/.../errclass.go` | **IMP-04** — classify via `errors.Is`/`ErrorClass()`; remove the substring fallback. |
| 8 | **`node_llm`/`node_ui` closed switch in the "agnostic" engine** — domain-named node types hard-wired into the generic engine. | `pipeline/application/engine/node_llm.go`, `node_ui.go` | Subsumed by **BR-06 / NR-06** (LLM/UI nodes + ModelGateway leave the engine). |
| 9 | **Undocumented two-go.mod L1/L3 seam** in the runtime repo — the nested `framework/` (L1) and the outer module (host machinery) share a repo with no stated boundary. | `runtime/go.mod` + `runtime/framework/go.mod` | Subsumed by **NR-04** (split the repo along the seam, document it). |
| 10 | **Third-party deps pulled into the framework tier** — `grpcstatus`/otel/redis import non-stdlib packages into a tier that claims stdlib-only. | `contracts/grpcstatus`, eventbus adapters | Relocate the offending packages to L2 (**NR-06**) so the residual L1 is genuinely stdlib-only. |

## Wave sequencing (R1–R7)

The relocations execute in dependency order. Each wave maps to the GitHub epic's sub-issues and to the ROADMAP milestone. The persistence wave (R3) interleaves with finishing **EDB-F2 (#69)**.

| Wave | Goal | Items | Depends on |
|---|---|---|---|
| **R1** | Kernel → L3 (lowest L1 coupling, move early); Ollama → Foundation/AI | NR-03, BR-03, BR-05, bug #6 | — |
| **R2** | Leaf contracts → L2 (api, agent, eventbus, grpcstatus) | NR-06 (contracts subset), BR-02, IMP-03, bug #10 | — |
| **R3** | Persistence seam → L2 (atomic re-import + retag + golden-regen) | NR-02, DOC-01, bug #4 | **EDB-F2 (#69)** store cutovers landed |
| **R4** | `installed` (admissibility/manifest/layout) → L3; `RenderPath` → L2 UI | NR-05, BR-04, BR-07, bug #5 | — |
| **R5** | Micro-kernel machinery split (loader/supervisor/resolver → L3; plugin/wasm → L2); **NR-01 un-embed Provider first** | NR-04, NR-01, BR-01, bugs #1, #2, #9 | un-embed (bug #2) precedes registry move |
| **R6** | Engine de-domain (CEL impl, ModelGateway, LLM/UI nodes leave the engine) | NR-06 (engine subset), BR-06, bug #8 | R2 |
| **R7** | Golden re-scope to the crisp L1 + Foundation capabilities catalog | NR-08, NR-07, BR-01 (catalog half) | R1–R6 |

> Standalone L1 engine fixes (IMP-01 sub-graph semantics, IMP-04 errclass, IMP-05 StepFrame/JobsQueue) ride alongside but are **not** boundary moves — they stay in L1 and can land in any wave.
