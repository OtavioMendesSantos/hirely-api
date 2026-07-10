
## 1. Padronização de Logs (Structured Logging)

Para garantir alta observabilidade e facilitar a indexação em plataformas de monitoramento (como Datadog, ELK ou Railway Logs), a API utiliza o padrão de **Structured Logging** baseado na biblioteca nativa do Go (`log/slog`).

Todas as saídas do sistema (`stdout` e `stderr`) são emitidas exclusivamente em formato JSON, seguindo o schema abaixo:

### 1.1 Schema do Log

```json
{
  "timestamp": "2026-08-01T12:00:00Z",
  "traceId": "req-abc123-xyz890",
  "service": "hirely-api",
  "env": "production",
  "level": "error",
  "operation": "CreateUser",
  "message": "Falha ao persistir usuário no banco de dados",
  "error": "duplicate key value violates unique constraint",
  "stackTrace": "goroutine 1 [running]:\nmain.main()...",
  "context": {
    "email": "candidato@email.com",
    "provider": "google"
  }
}
```

```json
{
  "timestamp": "2025-08-01T12:00:00Z", // HORA DO ACONTECIDO (UTC)
  "traceId": "abc123", // ID PARA TRACKING (ÚNICO POR TODO O FLUXO)
  "service": "task-processor", // NOME DO SERVIÇO GERADOR DO LOG
  "env": "production", // AMBIENTE QUE OCORREU
  "level": "error", // TIPO DO ERRO: info | warn | error | debug
  "operation": "CreateUser", // EM QUAL OPERACAO OCORREU
  "message": "Erro ao criar usuário", // MENSAGEM CONCISA
  "error": "duplicate key" // O QUE ORIGINOU O ERRO
  "stackTrace": // TRACING COMPLETO DO ERRO
  "context": // INFOS EXTRAS COMO IDS E ETC, APENAS SE NECESSÁRIO E RELEVANTE
}
```

### 1.2 Dicionário de Dados

| Campo | Tipo | Obrigatório | Descrição |
| --- | --- | --- | --- |
| `timestamp` | `string` | Sim | Data e hora exata do evento no formato ISO 8601, sempre em UTC (`Z`). |
| `traceId` | `string` | Sim | Identificador único da requisição (Correlation ID). Utilizado para rastrear todo o ciclo de vida de uma requisição que cruza múltiplos serviços ou middlewares. |
| `service` | `string` | Sim | Nome da aplicação ou microsserviço que gerou o log (ex: `hirely-api`). |
| `env` | `string` | Sim | Ambiente de execução atual (`development`, `staging`, `production`). |
| `level` | `string` | Sim | Severidade do evento. Valores permitidos: `info`, `warn`, `error`, `debug`. |
| `operation` | `string` | Sim | Nome da função, handler ou caso de uso onde o log foi originado (ex: `RegisterUser`, `DatabaseConn`). |
| `message` | `string` | Sim | Mensagem descritiva e concisa sobre o evento ocorrido. |
| `error` | `string` | Não | Obrigatório apenas quando `level` for `error`. Contém a string original de erro do sistema (`err.Error()`). |
| `stackTrace` | `string` | Não | Rastreamento da pilha de execução. Utilizado estritamente para exceções não tratadas (panics) ou erros críticos de infraestrutura. |
| `context` | `object` | Não | Estrutura chave-valor livre contendo metadados relevantes para a depuração (ex: IDs de recursos, payloads parciais sanitizados). **Atenção:** Dados sensíveis (senhas, tokens) nunca devem ser injetados aqui. |

### 1.3 Exemplo de Implementação (Go `log/slog`)

Os campos globais (`service`, `env`) e de transformação (`timestamp`, `level`) são injetados automaticamente no setup do logger. Os demais atributos devem ser passados explicitamente nos handlers e services:

```go
slog.Error("Falha ao persistir usuário no banco de dados",
    slog.String("traceId", ctx.Value("traceId").(string)),
    slog.String("operation", "RegisterUser"),
    slog.String("error", err.Error()),
    slog.Any("context", map[string]string{
        "email": req.Email,
    }),
)
```