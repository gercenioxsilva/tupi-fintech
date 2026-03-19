package http

import (
	"fmt"
	"net/http"
)

func openAPISpec(serverURL, environment string) map[string]any {
	return map[string]any{
		"openapi": "3.0.3",
		"info": map[string]any{
			"title":       "Tupi Fintech EMV API",
			"description": "API para processamento de transações EMV com mock de autorização aleatória, idempotência, PostgreSQL, observabilidade e documentação Swagger.",
			"version":     "1.2.0",
		},
		"servers": []map[string]string{{
			"url":         serverURL,
			"description": fmt.Sprintf("Ambiente %s", environment),
		}},
		"tags": []map[string]string{
			{"name": "Observability", "description": "Status e métricas da aplicação"},
			{"name": "Transactions", "description": "Processamento e consulta de transações EMV"},
		},
		"paths": map[string]any{
			"/healthz": map[string]any{"get": map[string]any{"tags": []string{"Observability"}, "summary": "Valida a saúde da aplicação", "operationId": "getHealthz", "responses": map[string]any{"200": map[string]any{"description": "Aplicação disponível", "content": map[string]any{"application/json": map[string]any{"schema": map[string]any{"$ref": "#/components/schemas/HealthResponse"}}}}}}},
			"/metrics": map[string]any{"get": map[string]any{"tags": []string{"Observability"}, "summary": "Expõe métricas Prometheus", "operationId": "getMetrics", "responses": map[string]any{"200": map[string]any{"description": "Métricas no formato Prometheus", "content": map[string]any{"text/plain": map[string]any{"schema": map[string]any{"type": "string"}}}}}}},
			"/api/v1/emv/transactions": map[string]any{
				"post": map[string]any{
					"tags":        []string{"Transactions"},
					"summary":     "Processa uma transação EMV com mock aleatório e idempotência",
					"operationId": "processTransaction",
					"parameters": []map[string]any{{
						"name":        "Idempotency-Key",
						"in":          "header",
						"required":    true,
						"description": "Chave única da intenção de pagamento. A primeira tentativa passa pelo mock de autorização aleatória; repetições com a mesma chave e o mesmo payload retornam a mesma resposta; reuso com dados diferentes retorna conflito.",
						"schema":      map[string]any{"type": "string", "minLength": 1},
					}},
					"requestBody": map[string]any{"required": true, "content": map[string]any{"application/json": map[string]any{"schema": map[string]any{"$ref": "#/components/schemas/ProcessTransactionCommand"}}}},
					"responses": map[string]any{
						"200": map[string]any{"description": "Transação processada com sucesso ou replay idempotente", "content": map[string]any{"application/json": map[string]any{"schema": map[string]any{"$ref": "#/components/schemas/ProcessedTransaction"}}}},
						"400": map[string]any{"description": "Payload inválido ou header de idempotência ausente"},
						"409": map[string]any{"description": "Mesma Idempotency-Key reutilizada com dados diferentes"},
						"422": map[string]any{"description": "Transação rejeitada por regra de negócio"},
					},
				},
				"get": map[string]any{
					"tags":        []string{"Transactions"},
					"summary":     "Lista a projeção de leitura das transações",
					"operationId": "listTransactions",
					"parameters":  []map[string]any{{"name": "limit", "in": "query", "schema": map[string]any{"type": "integer", "default": 50, "minimum": 1, "maximum": 100}}},
					"responses":   map[string]any{"200": map[string]any{"description": "Lista de transações", "content": map[string]any{"application/json": map[string]any{"schema": map[string]any{"$ref": "#/components/schemas/TransactionListResponse"}}}}},
				},
			},
			"/api/v1/emv/transactions/{correlationId}": map[string]any{
				"get": map[string]any{
					"tags":        []string{"Transactions"},
					"summary":     "Consulta uma transação pela correlação da autorização",
					"operationId": "getTransactionByCorrelationId",
					"parameters":  []map[string]any{{"name": "correlationId", "in": "path", "required": true, "schema": map[string]any{"type": "string"}}},
					"responses":   map[string]any{"200": map[string]any{"description": "Transação encontrada", "content": map[string]any{"application/json": map[string]any{"schema": map[string]any{"$ref": "#/components/schemas/ProcessedTransaction"}}}}, "404": map[string]any{"description": "Transação não encontrada"}},
				},
			},
		},
		"components": map[string]any{"schemas": map[string]any{
			"HealthResponse":            map[string]any{"type": "object", "properties": map[string]any{"status": map[string]any{"type": "string", "example": "ok"}, "env": map[string]any{"type": "string", "example": environment}}},
			"ProcessTransactionCommand": map[string]any{"type": "object", "required": []string{"tlv_payload", "amount", "currency"}, "properties": map[string]any{"tlv_payload": map[string]any{"type": "string"}, "amount": map[string]any{"type": "integer", "format": "int64", "minimum": 1}, "currency": map[string]any{"type": "string", "minLength": 3, "maxLength": 3}}},
			"ProcessedTransaction":      map[string]any{"type": "object", "properties": map[string]any{"idempotency_key": map[string]any{"type": "string", "example": "pay-123"}, "transaction": map[string]any{"$ref": "#/components/schemas/Transaction"}, "authorization": map[string]any{"$ref": "#/components/schemas/AuthorizationResult"}, "status": map[string]any{"type": "string", "example": "approved"}}},
			"TransactionListResponse":   map[string]any{"type": "object", "properties": map[string]any{"items": map[string]any{"type": "array", "items": map[string]any{"$ref": "#/components/schemas/ProcessedTransaction"}}}},
			"Transaction":               map[string]any{"type": "object", "properties": map[string]any{"pan": map[string]any{"type": "string", "example": "4111111111111111"}, "expiry_date": map[string]any{"type": "string", "example": "301231"}, "cvm": map[string]any{"type": "string", "example": "1E0300"}, "amount": map[string]any{"type": "integer", "format": "int64", "example": 1500}, "currency": map[string]any{"type": "string", "example": "BRL"}, "tlvs": map[string]any{"type": "object", "additionalProperties": map[string]any{"type": "string"}}, "processed_at": map[string]any{"type": "string", "format": "date-time"}}},
			"AuthorizationResult":       map[string]any{"type": "object", "properties": map[string]any{"approved": map[string]any{"type": "boolean", "example": true}, "code": map[string]any{"type": "string", "example": "00"}, "message": map[string]any{"type": "string", "example": "approved"}, "authorized_at": map[string]any{"type": "string", "format": "date-time"}, "correlation_id": map[string]any{"type": "string", "example": "abc-123"}}},
		}},
	}
}

func swaggerUIHandler() http.Handler {
	const page = `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <title>Tupi Fintech Swagger UI</title>
  <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css" />
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
  <script>
    window.ui = SwaggerUIBundle({
      url: '/openapi.json',
      dom_id: '#swagger-ui',
      presets: [SwaggerUIBundle.presets.apis],
      deepLinking: true,
    });
  </script>
</body>
</html>`

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(page))
	})
}
