# Desafio Técnico EMV - Go

Implementação de uma API em Go para processamento de transações EMV baseada em **vertical slice architecture**, com separação inspirada em **DDD**, princípios **SOLID**, **CQRS leve** para leitura/escrita, persistência em **PostgreSQL**, execução local via **Docker Compose**, observabilidade e documentação OpenAPI/Swagger.

## Visão geral

O fluxo atualizado atende ao desafio original e separa explicitamente os caminhos de comando e consulta:

1. Recebe uma transação EMV simulada via HTTP.
2. Decodifica TLVs (`5A`, `5F24`, `9F34`).
3. Valida PAN, data de validade e CVM no domínio.
4. Simula autorização em um gateway mock aleatório.
5. Persiste o resultado em PostgreSQL por meio da camada de escrita.
6. Expõe consultas de leitura desacopladas do fluxo de comando.
7. Mantém healthcheck, métricas Prometheus e documentação Swagger/OpenAPI.

## Estrutura do projeto

```text
cmd/api                          # bootstrap da aplicação
internal/platform                # config, observabilidade e utilitários HTTP
internal/transactions
├── application                  # casos de uso CQRS e portas
├── domain                       # entidades e regras de negócio
├── infrastructure               # postgres, decoder TLV, authorizer mock
└── interfaces/http              # handlers e roteamento
```

## Arquitetura da aplicação

A arquitetura detalhada agora está documentada em [`docs/architecture.md`](docs/architecture.md), incluindo:

- visão em camadas e por fluxo;
- diagramas Mermaid para request flow, componentes e implantação;
- pontos fortes e trade-offs da implementação atual;
- análise de escalabilidade com gargalos, riscos e roadmap recomendado.

### Resumo arquitetural

- **Vertical slice**: o slice `transactions` encapsula domínio, aplicação, infraestrutura e interface HTTP.
- **DDD + SOLID**: o domínio concentra invariantes; a aplicação orquestra casos de uso; a infraestrutura implementa adaptadores; a interface HTTP expõe contratos externos.
- **CQRS pragmático**: comandos e consultas já possuem serviços separados; hoje compartilham a mesma tabela PostgreSQL, mas a arquitetura permite evoluir para tabelas distintas de write model e read model.
- **Observabilidade básica**: há logs estruturados, métricas em memória expostas em `/metrics` e endpoints operacionais como `/healthz`.

## Endpoints

### `POST /api/v1/emv/transactions`

Processa uma transação EMV de forma idempotente. O header `Idempotency-Key` é obrigatório e garante replay seguro da mesma intenção de pagamento.

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
- `psql` disponível no ambiente quando a aplicação for executada fora de containers

### Com Go

1. Suba o PostgreSQL localmente.
2. Configure `POSTGRES_URL`.
3. Execute:

```bash
go mod tidy
go run ./cmd/api
```

### Com Docker Compose

Passo a passo para subir todo o ambiente localmente:

1. Na raiz do projeto, garanta que as portas `5432` e `8080` estejam livres.
2. Construa a imagem da API e suba os containers:

```bash
docker compose up --build
```

3. Aguarde o `postgres` ficar saudável e a API iniciar. O `docker-compose.yml` já define o `depends_on` com `condition: service_healthy`, então a aplicação só sobe depois do banco responder ao healthcheck.
4. Em outro terminal, valide se os containers estão em execução:

```bash
docker compose ps
```

5. Teste o healthcheck da API:

```bash
curl http://localhost:8080/healthz
```

6. Se quiser acompanhar os logs da aplicação:

```bash
docker compose logs -f api
```

7. Para derrubar o ambiente:

```bash
docker compose down
```

8. Para derrubar e remover também o volume do PostgreSQL, reiniciando o banco do zero:

```bash
docker compose down -v
```

Depois de subir a aplicação, acesse:

- `http://localhost:8080/swagger`
- `http://localhost:8080/openapi.json`
- `http://localhost:8080/healthz`
- `http://localhost:8080/metrics`

### Separação entre tabela de escrita e tabela de leitura

Sim, **é totalmente possível** separar as tabelas de leitura e escrita neste projeto. A base atual já ajuda nessa evolução porque a aplicação possui `CommandService` e `QueryService` distintos, ainda que hoje ambos usem o mesmo repositório e a mesma tabela.

Uma evolução natural seria:

1. o command side continuar persistindo a transação canônica em uma tabela de escrita, por exemplo `transactions_write`;
2. após a persistência, a aplicação atualizar uma tabela de leitura otimizada, por exemplo `transactions_read`;
3. o `QueryService` passar a ler somente da tabela de leitura;
4. a tabela de leitura poder ter índices e colunas moldados para consulta, sem impactar o modelo transacional de escrita.

Benefícios esperados:

- menor contenção entre escrita e leitura;
- liberdade para criar projeções específicas para listagem e busca;
- caminho mais claro para read replicas, cache ou atualização assíncrona no futuro.

Trade-offs a considerar:

- maior complexidade de sincronização entre write model e read model;
- necessidade de definir consistência síncrona ou eventual;
- necessidade de migrations e observabilidade melhores para operar duas projeções.

## Idempotência no POST de pagamento

O endpoint `POST /api/v1/emv/transactions` agora exige o header `Idempotency-Key`.

Comportamento esperado:

- primeira chamada com uma nova `Idempotency-Key`: a autorização mock pode aprovar ou rejeitar aleatoriamente;
- mesma `Idempotency-Key` + mesmo payload/valor: a API retorna a mesma resposta já processada, sem reautorizar a transação;
- mesma `Idempotency-Key` + payload ou valor diferente: a API retorna conflito (`409`);
- ausência do header: a API retorna erro de requisição inválida (`400`).

Isso evita reprocessamento acidental em cenários de retry e vincula a unicidade à intenção de pagamento, em vez de depender apenas do valor da transação.

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

Primeira chamada: a autorização mock pode aprovar ou rejeitar aleatoriamente.

```bash
curl -X POST http://localhost:8080/api/v1/emv/transactions \
  -H 'Content-Type: application/json' \
  -H 'Idempotency-Key: pay-123' \
  -d '{
    "tlv_payload": "5A0841111111111111115F24033012319F34031E0300",
    "amount": 1500,
    "currency": "BRL"
  }'
```

Retry seguro com a mesma `Idempotency-Key`: a API devolve a mesma resposta da primeira tentativa, mesmo que o mock de autorização seja aleatório.

```bash
curl -X POST http://localhost:8080/api/v1/emv/transactions \
  -H 'Content-Type: application/json' \
  -H 'Idempotency-Key: pay-123' \
  -d '{
    "tlv_payload": "5A0841111111111111115F24033012319F34031E0300",
    "amount": 1500,
    "currency": "BRL"
  }'
```

## Escalabilidade: leitura executiva

Hoje o projeto está **bem estruturado para evoluir**, mas **ainda não está pronto para alta escala sem ajustes importantes**.

### O que favorece a escalabilidade

- separação entre command side e query side no nível de aplicação;
- handlers HTTP stateless, facilitando scale-out horizontal;
- `http.Server` com timeout de leitura de cabeçalho e shutdown gracioso;
- índice em `processed_at` para listagem recente;
- deployment Kubernetes e compose para ambientes de execução.

### O que limita a escala neste momento

- o repositório usa o binário `psql` via `os/exec` em cada operação, o que aumenta latência, consumo de CPU e overhead por request;
- não existe pool de conexões nativo do banco;
- escrita e leitura compartilham a mesma tabela, o que reduz isolamento entre workloads;
- métricas ficam apenas em memória do processo, sem integração com tracing ou exporter dedicado;
- não há migrations versionadas nem estratégia clara de evolução de schema;
- o manifesto Kubernetes usa apenas 1 réplica e não define requests/limits nem autoscaling.

### Próximos passos prioritários

1. substituir `psql` por driver Go (`pgx` ou `database/sql`) com pool de conexões;
2. separar read model e write model se a volumetria de consulta crescer;
3. adicionar migrations versionadas e automação de rollout;
4. instrumentar métricas/tracing com stack padrão de observabilidade;
5. configurar readiness/liveness com recursos, réplicas e HPA no Kubernetes.

## Infra local

- `Dockerfile`: build multi-stage da aplicação.
- `docker-compose.yml`: sobe API + PostgreSQL para desenvolvimento local.
- `deploy/k8s/deployment.yaml`: manifesto base para Kubernetes.

## Testes

```bash
go test ./...
```

## Documentação complementar

- [Arquitetura e escalabilidade](docs/architecture.md)

