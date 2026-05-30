# plugfy-common

> **L1 — ABI / Contratos (a baseplate do PlugfyOS).** O único módulo que **todo** unit e host linka. Não importa ninguém; é importado por todos.

[![Layer](https://img.shields.io/badge/layer-L1_ABI-blue)]() [![Deps](https://img.shields.io/badge/deps-stdlib--only-green)]() [![Version](https://img.shields.io/badge/version-1.0.0-informational)]()

## O que é

`plugfy-common` publica as primitivas genéricas e estáveis que sustentam o micro-kernel do PlugfyOS — registro de provider, ciclo de vida de unit, transporte de eventos, IDs, erros, idempotência e resiliência. É **stdlib-only** por design: mantê-lo sem dependências externas garante que a raiz da árvore de dependência nunca arraste um domínio.

| Pacote | Conteúdo |
|---|---|
| `spi` | `Provider`, `Lifecycle`, `EventBus` e os `Kind*` — a SPI base que units estendem |
| `events` | envelope `CloudEvent` (CloudEvents 1.0) + tipos de evento |
| `ids` | gerador de ULID |
| `errs` | modelo de erro canônico |
| `idempotency` | contrato `Store` + impl in-memory |
| `resilience` | `Breaker`, `RetryPolicy`, `Bulkhead`, `Guard` |

## Como consumir

```go
import (
    "github.com/PlugfyOS/plugfy-common/spi"
    "github.com/PlugfyOS/plugfy-common/events"
)
```

Um driver/provider implementa uma porta SPI e se auto-registra; um system service define a sua porta e a expõe. Nenhum deles importa a implementação do outro — só este contrato.

## Build & test

```bash
go build ./...
go test -race ./...
bash scripts/decouple-check.sh   # garante stdlib-only + zero import de unit
```

## Regra (inegociável)

`plugfy-common` é **L1**: a raiz da seta de dependência. **Importa apenas a stdlib** e **nenhum** outro repo `PlugfyOS/*`. Qualquer `require` no `go.mod` ou import de unit **falha o CI**. Tudo que tem domínio, schema, persistência ou backend concreto vive **acima**, nunca aqui.

## Licença

Proprietário — ver [LICENSE](LICENSE).
