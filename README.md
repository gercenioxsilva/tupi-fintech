# Desafio Técnico EMV - Go

Implementação de uma API em Go para processamento básico de transações EMV baseada em **vertical slice architecture**, com separação inspirada em **DDD**, princípios **SOLID**, testes unitários, observabilidade, persistência em arquivo JSON Lines, execução local em Docker e manifesto Kubernetes.

## Visão geral

O fluxo implementado atende ao desafio original:

1. Recebe uma transação EMV simulada via HTTP.
2. Decodifica TLVs (`5A`, `5F24`, `9F34`).
3. Valida PAN, data de validade e CVM.
4. Simula autorização em um gateway mock determinístico.
5. Persiste o log da transação em arquivo JSON Lines.
6. Expõe healthcheck e métricas Prometheus.

## Arquitetura

```text
cmd/api                          # bootstrap da aplicação
internal/platform                # config, observabilidade e utilitários HTTP
internal/transactions
├── application                  # casos de uso e portas
├── domain                       # entidades e regras de negócio
├── infrastructure               # persistência em arquivo, decoder TLV, authorizer mock
└── interfaces/http              # handlers e roteamento
```

### Vertical slice

A funcionalidade de transações EMV está isolada dentro de `internal/transactions`, contendo todas as camadas necessárias para o slice de negócio.

### DDD + SOLID

- **Domain** concentra as regras de validação do cartão.
- **Application** orquestra o caso de uso usando interfaces/ports.
- **Infrastructure** implementa adaptadores externos.
- **Interfaces** expõe a API HTTP.
- Dependências são invertidas via interfaces.
- Cada componente tem responsabilidade única.

## Endpoints

### `POST /api/v1/emv/transactions`

Request:

```json
{
  "tlv_payload": "5A0841111111111111115F24033012319F34031E0300",
  "amount": 1500,
  "currency": "BRL"
}
```

Response:

```json
{
  "transaction": {
    "pan": "4111111111111111",
    "expiry_date": "301231",
    "cvm": "1E0300",
    "amount": 1500,
    "currency": "BRL",
    "tlvs": {
      "5A": "4111111111111111",
      "5F24": "301231",
      "9F34": "1E0300"
    },
    "processed_at": "2026-03-19T00:00:00Z"
  },
  "authorization": {
    "approved": true,
    "code": "00",
    "message": "approved",
    "authorized_at": "2026-03-19T00:00:00Z",
    "correlation_id": "..."
  },
  "status": "approved"
}
```

### `GET /healthz`

Healthcheck para Kubernetes.

### `GET /metrics`

Métricas Prometheus.

## Executando localmente

### Requisitos

- Go 1.23+
- Docker (opcional)

### Com Go

```bash
go mod tidy
go run ./cmd/api
```

### Com Docker

```bash
docker build -t emv-api:local .
docker run --rm -p 8080:8080 -e DATABASE_PATH='/tmp/transactions.jsonl' emv-api:local
```

## Variáveis de ambiente

- `HTTP_ADDRESS` (default `:8080`)
- `LOG_LEVEL` (default `INFO`)
- `DATABASE_PATH` (default `./data/transactions.jsonl`)
- `APP_ENV` (default `local`)

## Testes

```bash
go test ./...
```

## Exemplo com curl

```bash
curl -X POST http://localhost:8080/api/v1/emv/transactions \
  -H 'Content-Type: application/json' \
  -d '{
    "tlv_payload": "5A0841111111111111115F24033012319F34031E0300",
    "amount": 1500,
    "currency": "BRL"
  }'
```

## Docker e Kubernetes

- `Dockerfile`: build multi-stage, imagem mínima para execução local.
- `deploy/k8s/deployment.yaml`: Deployment + Service prontos para cluster.

## Observabilidade

- Logs estruturados em JSON com `log/slog`.
- Métricas Prometheus em `/metrics`.
- Endpoint `/healthz` para readiness/liveness.

## Próximos passos sugeridos

- Adicionar tracing distribuído com OpenTelemetry.
- Criar testes de integração end-to-end com arquivo temporário e cenários HTTP.
- Evoluir autorização mock para provider externo configurável.
