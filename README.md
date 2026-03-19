# Desafio Técnico EMV - Go

Implementação de uma API em Go para processamento de transações EMV baseada em **vertical slice architecture**, com separação inspirada em **DDD**, princípios **SOLID**, **CQRS leve** para leitura/escrita, persistência em **PostgreSQL**, execução local via **Docker Compose**, observabilidade e documentação OpenAPI/Swagger.

## Visão geral

O fluxo atualizado atende ao desafio original e agora separa explicitamente os caminhos de comando e consulta:

1. Recebe uma transação EMV simulada via HTTP.
2. Decodifica TLVs (`5A`, `5F24`, `9F34`).
3. Valida PAN, data de validade e CVM no domínio.
4. Simula autorização em um gateway mock determinístico.
5. Persiste o resultado em PostgreSQL por meio da camada de escrita.
6. Expõe consultas de leitura desacopladas do fluxo de comando.
7. Mantém healthcheck, métricas Prometheus e documentação Swagger/OpenAPI.

## Arquitetura

```text
cmd/api                          # bootstrap da aplicação
internal/platform                # config, observabilidade e utilitários HTTP
internal/transactions
├── application                  # casos de uso CQRS e portas
├── domain                       # entidades e regras de negócio
├── infrastructure               # postgres, decoder TLV, authorizer mock
└── interfaces/http              # handlers e roteamento
```

### Vertical slice

O slice `transactions` continua encapsulando domínio, aplicação, infraestrutura e interface HTTP, preservando o padrão vertical slice.

### DDD + SOLID

- **Domain** concentra invariantes e validações.
- **Application** separa `CommandService` e `QueryService`.
- **Infrastructure** implementa adaptadores PostgreSQL e integrações externas.
- **Interfaces** expõe endpoints HTTP sem acoplamento ao banco.
- Dependências seguem inversion via interfaces (`TransactionWriter` e `TransactionReader`).

### CQRS aplicado

A adoção de CQRS foi considerada viável porque o fluxo já possuía um caso de uso de escrita bem definido. A implementação foi feita de maneira pragmática:

- **Comandos**: `POST /api/v1/emv/transactions` processa e grava a transação.
- **Consultas**: `GET /api/v1/emv/transactions` e `GET /api/v1/emv/transactions/{correlationId}` usam a projeção de leitura persistida.

Ainda usamos a mesma tabela PostgreSQL para evitar complexidade prematura, mas o contrato de aplicação já separa leitura e escrita, facilitando futura evolução para read models dedicados.

## Endpoints

### `POST /api/v1/emv/transactions`

Processa uma transação EMV.

### `GET /api/v1/emv/transactions?limit=50`

Lista as transações mais recentes da projeção de leitura.

### `GET /api/v1/emv/transactions/{correlationId}`

Consulta uma transação específica pelo `correlation_id` retornado na autorização.

### `GET /healthz`

Healthcheck para Docker/Kubernetes.

### `GET /metrics`

Métricas Prometheus.

### `GET /openapi.json`

Especificação OpenAPI 3.0 da API.

### `GET /swagger`

Interface Swagger UI para explorar e testar os endpoints da aplicação.

## Executando localmente

### Requisitos

- Go 1.23+
- Docker + Docker Compose

### Com Go

1. Suba o PostgreSQL localmente.
2. Configure `POSTGRES_URL`.
3. Execute:

```bash
go mod tidy
go run ./cmd/api
```

### Com Docker Compose

```bash
docker compose up --build
```

Depois de subir a aplicação, acesse:

- `http://localhost:8080/swagger`
- `http://localhost:8080/openapi.json`
- `http://localhost:8080/healthz`

## Variáveis de ambiente

- `HTTP_ADDRESS` (default `:8080`)
- `LOG_LEVEL` (default `INFO`)
- `APP_ENV` (default `local`)
- `POSTGRES_URL` (default `postgres://postgres:postgres@localhost:5432/tupi_fintech?sslmode=disable`)
- `POSTGRES_HOST` (default `localhost`)
- `POSTGRES_PORT` (default `5432`)
- `POSTGRES_DB` (default `tupi_fintech`)
- `POSTGRES_USER` (default `postgres`)
- `POSTGRES_PASSWORD` (default `postgres`)

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

## Infra local

- `Dockerfile`: build multi-stage da aplicação.
- `docker-compose.yml`: sobe API + PostgreSQL para desenvolvimento local.
- `deploy/k8s/deployment.yaml`: manifesto base para Kubernetes.

## Testes

```bash
go test ./...
```

## Próximos passos sugeridos

- Evoluir o read side para tabela/materialized view dedicada.
- Adicionar migrations versionadas.
- Incluir testes de integração com PostgreSQL via container efêmero.
