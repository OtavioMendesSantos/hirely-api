# Servidor MCP (Model Context Protocol)

O backend expõe um servidor MCP (Model Context Protocol) para que assistentes/agentes gerenciem candidaturas e tags em nome do usuário autenticado.

## Transporte e endpoint

| Item         | Valor                                                                                                              |
| ------------ | ------------------------------------------------------------------------------------------------------------------ |
| Transporte   | **Streamable HTTP** (spec MCP 2025-03-26+)                                                                         |
| Endpoint     | `POST /v1/mcp`                                                                                                     |
| Autenticação | `Authorization: Bearer hirely_sk_...` (obrigatória em todas as rotas)                                              |
| Protocolo    | Detectado por requisição: **2026-07-28** (`server/discover`, stateless) ou legado (`initialize`, até `2025-11-25`) |
| Content-Type | `application/json`                                                                                                 |

O mesmo endpoint atende clientes modernos e legados: o `mcp-go` negocia a "era" a partir do header `Mcp-Protocol-Version` e do `_meta` do corpo JSON-RPC. Respostas são JSON simples (`application/json`); SSE só é usado quando há streaming de verdade.

> **Compatibilidade:** `GET /v1/mcp/sse` e `POST /v1/mcp/messages` (transporte **HTTP+SSE**) continuam funcionando, porém estão **deprecados** no spec MCP e serão removidos. Clientes novos devem usar `POST /v1/mcp`.

## Ferramentas disponíveis

| Tool                 | Descrição                                                  |
| -------------------- | ---------------------------------------------------------- |
| `read_applications`  | Lista candidaturas (filtros opcionais por texto e status). |
| `insert_application` | Cria uma candidatura.                                      |
| `update_application` | Atualiza uma candidatura existente.                        |
| `list_tags`          | Lista as tags do usuário.                                  |
| `create_tag`         | Cria uma tag.                                              |
| `delete_tag`         | Remove uma tag.                                            |

## Exemplo rápido / curl

```bash
KEY="hirely_sk_..."   # sua API Key

# Handshake moderno (2026-07-28)
curl -s https://hirely-api.up.railway.app/v1/mcp \
  -H "Authorization: Bearer $KEY" \
  -H "Content-Type: application/json" \
  -H "MCP-Protocol-Version: 2026-07-28" \
  -H "Mcp-Method: server/discover" \
  -d '{"jsonrpc":"2.0","id":1,"method":"server/discover","params":{"_meta":{"io.modelcontextprotocol/protocolVersion":"2026-07-28","io.modelcontextprotocol/clientInfo":{"name":"curl","version":"1"},"io.modelcontextprotocol/clientCapabilities":{}}}}'

# Handshake legado (initialize)
curl -s https://hirely-api.up.railway.app/v1/mcp \
  -H "Authorization: Bearer $KEY" \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"curl","version":"1"}}}'
```

## Configuração de cliente (OpenCode v2)

Formato V2 (`mcp.servers.<nome>`). O campo `protocol: "auto"` testa `server/discover` (2026-07-28) e cai para o fluxo legado se necessário:

```jsonc
{
  "mcp": {
    "servers": {
      "hirely-backend": {
        "type": "remote",
        "url": "https://hirely-api.up.railway.app/v1/mcp",
        "oauth": false,
        "headers": { "Authorization": "Bearer {env:HIRELY_API_KEY}" },
        "protocol": "auto",
      },
    },
  },
}
```

Defina `HIRELY_API_KEY` com a sua API Key antes de iniciar o OpenCode. Depois valide com `opencode mcp list` (esperado: `✓ hirely-backend connected`).

## Segurança

- Bearer token validado em todas as rotas MCP (API Key ou sessão).
- Corpo JSON-RPC limitado a 1 MiB.
- Rate limiting por IP no endpoint MCP (120 req/min).
- CORS expõe apenas os headers necessários, incluindo `Mcp-Session-Id` e `Mcp-*`.
- Nenhum secret é registrado em log.
