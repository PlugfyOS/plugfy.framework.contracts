# Arquitetura — plugfy-common

## Posição

- **Camada:** L1 — ABI / Contratos SPI (a baseplate).
- **Dimensão:** Foundations.
- **Kind:** `shared-baseplate` — não é unit (não se auto-registra, não tem manifesto/capability) e não é host (não compõe nada). É a ABI primitiva que todos linkam em build-time.

## Fronteiras

**Faz:** define interfaces, o envelope `CloudEvent`, e utilitários puros (ids/errs/idempotency/resilience). Nada mais.

**Não faz:** persistência, HTTP, backends concretos (Postgres/NATS/S3), lógica de domínio, UI. Esses vivem nas camadas acima — driver (L2), runtime (L3/L4), system services (L7).

## Inversão de dependência

O domínio define a porta; o adapter a implementa. Por isso o adapter depende **deste** contrato — nunca o contrário. A seta de dependência aponta sempre para cá.

```
L2 platform-provider-*  ──implementa──►  spi.* (aqui)  ◄──define/usa──  L7 system-*
                                          ▲
                          todos os hosts (L3/L4/L6) também linkam
```

## Gate de fronteira (decouple-check)

`scripts/decouple-check.sh` falha o build se:
1. houver qualquer `require` no `go.mod` (deve ser **stdlib-only**);
2. algum pacote importar outro repo `PlugfyOS/*`.

Isso preserva a invariante de que a raiz da árvore de dependência é estável e zero-domínio.

## Versionamento

ABI estável: qualquer quebra de assinatura pública é **major**. Os consumidores pinam `^1.x`. Um teste golden de ABI congela as assinaturas exportadas.
