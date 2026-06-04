# Plugfy Framework — Documentação

Documentação de arquitetura e análise do **Plugfy Framework (L1)** — **exatamente três conceitos: Unit + Pipeline + Execution** (`contracts` + `engine`/`runner` + a CLI `plugfy`).

## Documentos

| Documento | Conteúdo |
|---|---|
| [**The Three Layers (canonical)**](architecture/LAYERS.md) | The canonical boundary model: **Framework DEFINES & RUNS pipelines · Foundations BUILDS apps/services/scripts · Platform SCALES them into a governed ecosystem.** What belongs in L1 / L2 / L3, why the line falls there, and the empirical `plugfy run` proof of the crisp L1 surface. The placement ruler for the whole codebase. |
| [Boundary-refactor backlog](architecture/boundary-refactor-backlog.md) | The actionable backlog that brings the as-built tree into line with the three-layer model: the responsibility map, the relocation items (NR-01…NR-08), the BR/IMP/DOC reconciliation, the bug list, and the wave sequencing (R1–R7). |
| [**Documento de Arquitetura**](architecture/plugfy-framework-architecture.md) | Arquitetura de referência do framework — **o que ele é capaz de fazer e como faz**, com exemplos do simples ao complexo. *Escopo: apenas o framework (L1 — Unit + Pipeline + Execution), domain-agnostic e stdlib-only.* O contrato `Unit` de dois métodos, o DAG de nós **genéricos**, a execução assíncrona como lib Go ou pela CLI `plugfy`, e a golden ABI mínima. Estruturado em **arc42**, com diagramas **C4/Mermaid**, decisões em **ADR (Nygard)** e qualidade em **ISO/IEC 25010:2023**. |
| [Análise Comparativa (legado × atual)](analisys/comparative-complete-detailed-old-x-new-version-plugfy-framework.md) | Comparação detalhada entre a solução Plugfy legada (.NET) e a atual. *Documento de análise histórica, com escopo mais amplo.* |

## Diagramas

Todos os diagramas são **Mermaid** e renderizam no GitHub e no VS Code (extensão *Markdown Preview Mermaid Support*). Foram validados por renderizador (`@mermaid-js/mermaid-cli`).

## Como navegar o documento de arquitetura

- **Para entender o que o framework faz:** comece pela [§1.2 Catálogo de capacidades](architecture/plugfy-framework-architecture.md#12-do-que-o-framework-é-capaz-catálogo-de-capacidades) — tudo dentro de Unit + Pipeline + Execution.
- **Para ver o que NÃO é o framework:** [§3.3 A fronteira L1↔L2↔L3](architecture/plugfy-framework-architecture.md#33-o-que-não-é-o-framework-a-fronteira-l1l2l3) (loader/sandbox/supervisor/kernel/triggers/comunicação são L2/L3) — visão completa em [LAYERS.md](architecture/LAYERS.md).
- **Para ver como faz, na prática:** [§6 Exemplos trabalhados](architecture/plugfy-framework-architecture.md#63-exemplos-trabalhados) (do "olá unidade" ao processo composto, resiliente e recursivo).
- **Para as decisões e trade-offs:** [§9 ADRs](architecture/plugfy-framework-architecture.md#9-decisões-de-arquitetura-adrs).
