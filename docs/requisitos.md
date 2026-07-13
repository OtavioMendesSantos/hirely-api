# Hirely - Documentação de Especificação Técnica e Arquitetura

O **Hirely** é um aplicativo web focado na gestão, centralização e rastreabilidade de candidaturas a vagas de emprego (_Job Applications_). O sistema adota uma filosofia de design _Mobile-First_, fornecendo uma experiência otimizada para dispositivos móveis através de uma interface baseada em Kanban, tags flexíveis e uma timeline detalhada de interações.

---

## 1. Visão Geral do Escopo (MVP)

### 1.1 Requisitos Funcionais (RF)

- **[RF-01] Autenticação e Gestão de Usuários:**
  - _Registro Tradicional:_ Cadastro de usuário com **Nome, Email e Senha** (armazenamento seguro via hash forte `bcrypt` ou `argon2id`).
  - _Login Tradicional:_ Autenticação por **Email e Senha** com emissão de token JWT Bearer (Access Token / Refresh Token).
  - _Login Social (OAuth 2.0 / OIDC):_ Autenticação via provedores **Google** e **LinkedIn**, com vinculação automática de conta por e-mail verificado.
  - _Isolamento de Dados:_ Isolamento estrito dos dados e recursos vinculados ao `user_id` autenticado.
- **[RF-02] Quadro Kanban Único (Job Applications):** Visualização e gerenciamento de candidaturas através de estados fixos de fluxo: _To Apply (`TO_APPLY`), Applied (`APPLIED`), Interview (`INTERVIEW`), Offer (`OFFER`), Rejected (`REJECTED`)_. Suporte completo a criação, atualização de status/dados e exclusão/arquivamento.
- **[RF-03] Sistema Flexível de Tags:** Usuários podem criar, listar e customizar suas próprias tags com cores personalizadas (ex: "Remote", "Go", "Angular") e utilizá-las para filtragem dinâmica no quadro.
- **[RF-04] Timeline de Histórico e Eventos (`Events`):**
  - _Automático (`AUTOMATIC`):_ Log gerado pelo sistema ao criar ou transicionar uma candidatura entre estados do Kanban.
  - _Manual (`MANUAL`):_ Inserção de notas e interações livres pelo usuário na timeline (ex: "Mandei mensagem para o recrutador no LinkedIn").
- **[RF-05] Visualização de Métricas:** Agregação de dados com totais de candidaturas, taxas de conversão de funil e desempenho por tag e status.

### 1.2 Requisitos Não-Funcionais (RNF)

- **[RNF-01] Interface Mobile-First:** Design projetado prioritariamente para telas móveis e expandido progressivamente para resoluções desktop.
- **[RNF-02] Arquitetura Hexagonal (Ports & Adapters):** Isolamento total do núcleo de negócios (_Domain_ e _Ports_) contra acoplamentos a frameworks, ORMs ou drivers de transporte HTTP.
- **[RNF-03] Portabilidade de Persistência:** Design agnóstico projetado para suportar fácil migração ou coexistência entre bancos relacionais (PostgreSQL/GORM) e orientados a documentos (MongoDB).
- **[RNF-04] API Baseada em Recursos (Google API Design):** Endpoints padronizados em inglês mapeados hierarquicamente a partir do recurso pai (ex: `/v1/users/{user_id}/applications`).
- **[RNF-05] Comportamento Fail-Fast de Configurações:** Parseamento e validação estrita de variáveis de ambiente no startup da aplicação por meio de Struct Tags.
- **[RNF-06] Consistência de Tempo e Segurança:** Servidor configurado globalmente em **UTC** e com suporte nativo a CORS e validação de tokens JWT.

---

## 2. Arquitetura do Sistema

### 2.1 Stack Tecnológica

- **Frontend Client:** Angular (Hospedagem estática em plataformas CDN como Vercel/Netlify com regras de _fallback_ para SPA).
- **Backend Runtime:** Go (Golang 1.23+).
- **ORM / Drivers:** GORM para mapeamento relacional (PostgreSQL) com migrações gerenciadas (`gorm.AutoMigrate` no MVP ou `golang-migrate`).

### 2.2 Estrutura de Diretórios do Backend

```text
hirely-api/
├── cmd/
│   └── api/
│       └── main.go                 # Startup, inicialização global de tempo (UTC) e injeção de dependências
├── internal/
│   ├── config/
│   │   └── config.go               # Fail-Fast parsing de variáveis de ambiente (caarlos0/env + godotenv)
│   ├── core/                       # Núcleo da Aplicação (Hexágono Interno - Puro)
│   │   ├── domain/                 # Entidades de negócio agnósticas (Application, Tag, Event, User)
│   │   ├── ports/                  # Interfaces de entrada (Services) e saída (Repositories)
│   │   └── services/               # Implementação das regras de negócio e casos de uso
│   └── adapters/                   # Infraestrutura e Transporte (Hexágono Externo)
│       ├── http/                   # Camada REST (Google API Design / AIPs)
│       │   ├── router.go           # Roteador centralizado (Mux nativo Go 1.22+)
│       │   ├── middleware/         # Interceptadores HTTP
│       │   │   ├── auth.go         # Validação de JWT Bearer Token e verificação de UserID
│       │   │   └── cors.go         # Configuração de CORS para SPA frontend
│       │   ├── handlers/
│       │   │   ├── health_handler.go      # Monitoramento de infraestrutura
│       │   │   ├── auth_handler.go        # Autenticação (Register / Login Email e Senha)
│       │   │   ├── oauth_handler.go       # Autenticação Social (OAuth 2.0 Google / LinkedIn)
│       │   │   ├── application_handler.go # CRUD de Job Applications
│       │   │   ├── tag_handler.go         # Gestão de Tags
│       │   │   ├── event_handler.go       # Gestão da Timeline de Eventos
│       │   │   └── metrics_handler.go     # Agregações e funil de conversão
│       │   └── dto/
│       │       ├── requests.go     # DTOs de entrada validados
│       │       ├── responses.go    # DTOs de saída
│       │       └── errors.go       # Estrutura padronizada de erros HTTP (RFC 7807 / Google Standard)
│       └── storage/                # Camada de Persistência
│           └── postgres/           # Adaptador concreto utilizando GORM
│               ├── models.go       # Schemas relacionais (ApplicationModel, TagModel, EventModel)
│               └── repositories.go # Implementação concreta de ports/repositories
├── .env                            # Variáveis de ambiente locais (ignorado em prod)
├── Dockerfile                      # Multi-stage build (Builder Alpine -> Final scratch/alpine)
├── go.mod
└── go.sum
```

---

## 3. Especificação do Modelo de Domínio e Banco de Dados

Para garantir a independência do banco de dados, a camada de domínio trata `Application` como um **Aggregate Root**. Os identificadores primários utilizam strings (UUID v4 / ULID).

### 3.1 Entidades do Domínio (`internal/core/domain`)

```text
+-------------------------------+       +------------------------------------+
|         User (Entity)         | 1   N |        OAuthAccount (Entity)       |
+-------------------------------+-------+------------------------------------+
| - ID: string (UUID/ULID)      |       | - ID: string                       |
| - Name: string                |       | - UserID: string                   |
| - Email: string (Unique)      |       | - Provider: OAuthProvider          |
| - PasswordHash: string (opt)  |       |   (GOOGLE, LINKEDIN)               |
| - OAuthAccounts: []OAuthAccount|      | - ProviderUserID: string           |
| - CreatedAt: time.Time        |       | - CreatedAt: time.Time             |
+-------------------------------+       +------------------------------------+
               |
               | 1:N
               v
+-------------------------------------------------------------+
|                     Application (Core)                      |
+-------------------------------------------------------------+
| - ID: string (UUID/ULID)                                    |
| - UserID: string (UUID/ULID)                                |
| - JobTitle: string                                          |
| - CompanyName: string                                       |
| - JobURL: string (opcional)                                 |
| - SalaryRange: string (opcional)                            |
| - Status: ApplicationStatus (TO_APPLY, APPLIED, INTERVIEW,  |
|                              OFFER, REJECTED)               |
| - Tags: []Tag                                               |
| - Timeline: []Event                                         |
| - CreatedAt: time.Time                                      |
| - UpdatedAt: time.Time                                      |
+-------------------------------------------------------------+
               |                               |
               | 1:N                           | N:N
               v                               v
+-----------------------------+   +-----------------------------+
|        Event (Entity)       |   |         Tag (Entity)        |
+-----------------------------+   +-----------------------------+
| - ID: string                |   | - ID: string                |
| - ApplicationID: string     |   | - UserID: string            |
| - Type: EventType           |   | - Name: string              |
|   (AUTOMATIC, MANUAL)       |   | - ColorHex: string          |
| - Description: string       |   | - CreatedAt: time.Time      |
| - PreviousStatus: string    |   +-----------------------------+
| - NewStatus: string         |
| - CreatedAt: time.Time      |
+-----------------------------+
```

### 3.2 Mapeamento Relacional no PostgreSQL (GORM)

No banco relacional PostgreSQL, as tabelas são normalizadas com chaves estrangeiras (`FK`) e índices de performance por `user_id`:

```text
+----------------------+       +--------------------------+
|        users         | 1   N |      oauth_accounts      |
+----------------------+-------+--------------------------+
| id (PK, UUID)        |       | id (PK, UUID)            |
| name (VARCHAR)       |       | user_id (FK, UUID)       |
| email (Unique)       |       | provider (VARCHAR)       |
| password_hash(VARCHAR|       | provider_user_id (Unique)|
| created_at(TIMESTAMP)|       | created_at (TIMESTAMP)   |
+----------------------+       +--------------------------+
          ^
          | 1
          |
          | N
+----------------------+       +--------------------------+       +----------------------+
|     applications     | 1   N |          events          |       |         tags         |
+----------------------+-------+--------------------------+       +----------------------+
| id (PK, UUID)        |       | id (PK, UUID)            |       | id (PK, UUID)        |
| user_id (FK, Index)  |       | application_id (FK, UUID)|       | user_id (FK, Index)  |
| job_title (VARCHAR)  |       | type (VARCHAR)           |       | name (VARCHAR)       |
| company_name(VARCHAR)|       | description (TEXT)       |       | color_hex (VARCHAR)  |
| job_url (TEXT)       |       | previous_status(VARCHAR) |       | created_at (TIMESTAMP|
| salary_range(VARCHAR)|       | new_status (VARCHAR)     |       +----------------------+
| status (VARCHAR)     |       | created_at (TIMESTAMP)   |                  ^
| created_at(TIMESTAMP)|       +--------------------------+                  |
| updated_at(TIMESTAMP)|                                                     |
+----------------------+                                                     |
          ^                                                                  |
          | 1                                                                | 1
          +-------------------+                          +-------------------+
                              | N                      N |
                     +------------------------------------+
                     |          application_tags          |  (Tabela Pivot N:N)
                     +------------------------------------+
                     | application_id (FK, UUID)          |
                     | tag_id (FK, UUID)                  |
                     +------------------------------------+
```

---

## 4. Contratos da API (Google API Design Standard)

A API do Hirely implementa endpoints orientados a recursos em inglês, com padronização de paginação, filtros e tratamento de erros.

### 4.1 Tabela Geral de Endpoints

| Método HTTP | URL Padrão                                                 | Descrição                                                              | Autenticação               |
| ----------- | ---------------------------------------------------------- | ---------------------------------------------------------------------- | -------------------------- |
| `GET`       | `/v1/health`                                               | Verifica integridade básica e timestamp em UTC do servidor.            | Pública                    |
| `POST`      | `/v1/users`                                                | Registro de usuário (padrão `Create` resource do Google API Design).   | Pública                    |
| `POST`      | `/v1/users:login`                                          | Login com Email e Senha, retornando token JWT (padrão Custom Method).  | Pública                    |
| `GET`       | `/v1/users:oauthUrl?provider={provider}`                   | Retorna URL de autorização OAuth 2.0 (`google` ou `linkedin`).         | Pública                    |
| `POST`      | `/v1/users:oauthLogin`                                     | Recebe código OAuth (`provider`, `code`), autentica e retorna token.   | Pública                    |
| `POST`      | `/v1/users/{user_id}/applications`                         | Cria uma nova candidatura para o usuário.                              | JWT (`user_id` compatível) |
| `GET`       | `/v1/users/{user_id}/applications`                         | Lista candidaturas com suporte a filtros e paginação.                  | JWT                        |
| `GET`       | `/v1/users/{user_id}/applications/{application_id}`        | Retorna detalhes de uma candidatura (incluindo tags e timeline).       | JWT                        |
| `PATCH`     | `/v1/users/{user_id}/applications/{application_id}`        | Atualiza status ou dados da candidatura (`job_title`, `status`, etc.). | JWT                        |
| `DELETE`    | `/v1/users/{user_id}/applications/{application_id}`        | Remove ou arquiva uma candidatura.                                     | JWT                        |
| `GET`       | `/v1/users/{user_id}/applications:stats`                   | Retorna métricas agregadas (funil por status, contagem por tag).       | JWT                        |
| `POST`      | `/v1/users/{user_id}/applications/{application_id}/events` | Adiciona nota/evento manual na timeline da candidatura.                | JWT                        |
| `POST`      | `/v1/users/{user_id}/tags`                                 | Cria uma tag customizada.                                              | JWT                        |
| `GET`       | `/v1/users/{user_id}/tags`                                 | Lista todas as tags customizadas do usuário.                           | JWT                        |
| `DELETE`    | `/v1/users/{user_id}/tags/{tag_id}`                        | Remove uma tag customizada.                                            | JWT                        |

---

### 4.2 Parâmetros de Busca e Filtragem (`GET /v1/users/{user_id}/applications`)

O endpoint de listagem aceita os seguintes parâmetros de consulta (_query parameters_):

- `status`: Filtra por status (ex: `?status=APPLIED` ou múltiplos: `?status=INTERVIEW,OFFER`).
- `tag_ids`: Filtra por tags (ex: `?tag_ids=uuid-tag-1,uuid-tag-2`).
- `page_size`: Quantidade de itens por página (padrão: `20`, máximo: `100`).
- `page_token`: Token para paginação (`next_page_token` retornado pela chamada anterior).

---

### 4.3 Exemplos de Payloads JSON

#### Requisição de Registro (`RegisterRequest` - `POST /v1/users`)

```json
{
  "name": "Otavio Mendes",
  "email": "otavio@hirely.app",
  "password": "StrongPassword123!"
}
```

#### Resposta de Registro (`POST /v1/users`)

```json
{
  "id": "c3a7e4b2-891d-4f1a-b6e9-2f4d1e8c9a0b",
  "name": "Otavio Mendes",
  "email": "otavio@hirely.app",
  "createTime": "2026-07-13T19:00:00Z"
}
```

#### Requisição de Login Tradicional (`LoginRequest` - `POST /v1/users:login`)

```json
{
  "email": "otavio@hirely.app",
  "password": "StrongPassword123!"
}
```

#### Resposta de Login Tradicional (`POST /v1/users:login`)

```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
}
```

#### Requisição de Callback Social (`OAuthCallbackRequest` - `POST /v1/users:oauthLogin`)

```json
{
  "provider": "google",
  "code": "4/0AeaYSHC...",
  "redirect_uri": "https://hirely.app/auth/callback"
}
```

#### Resposta de Autenticação Social / OAuth (`AuthResponse`)

```json
{
  "user": {
    "id": "c3a7e4b2-891d-4f1a-b6e9-2f4d1e8c9a0b",
    "name": "Otavio Mendes",
    "email": "otavio@hirely.app"
  },
  "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "token_type": "Bearer",
  "expires_in": 86400
}
```

#### Resposta da Listagem de Candidaturas (`ListApplicationsResponse`)

```json
{
  "applications": [
    {
      "name": "users/user-123/applications/9b1deb4d-3b7d-4bad-9bdd-2b0d7b3dcb6d",
      "id": "9b1deb4d-3b7d-4bad-9bdd-2b0d7b3dcb6d",
      "job_title": "Senior Backend Engineer",
      "company_name": "Hirely Corp",
      "job_url": "https://linkedin.com/jobs/view/12345",
      "salary_range": "$120k - $140k",
      "status": "APPLIED",
      "tags": [
        {
          "id": "11111111-2222-3333-4444-555555555555",
          "name": "Go",
          "color_hex": "#00ADD8"
        },
        {
          "id": "66666666-7777-8888-9999-000000000000",
          "name": "Remote",
          "color_hex": "#10B981"
        }
      ],
      "created_at": "2026-07-09T14:00:00Z",
      "updated_at": "2026-07-09T16:30:00Z"
    }
  ],
  "next_page_token": ""
}
```

#### Resposta de Métricas / Funil (`ApplicationStatsResponse`)

```json
{
  "total_applications": 24,
  "funil_by_status": {
    "TO_APPLY": 5,
    "APPLIED": 12,
    "INTERVIEW": 4,
    "OFFER": 1,
    "REJECTED": 2
  },
  "conversion_rate_interview": 0.33,
  "top_tags": [
    { "tag_name": "Remote", "count": 18 },
    { "tag_name": "Go", "count": 14 }
  ]
}
```

#### Padrão de Resposta para Erros (`ErrorResponse`)

Seguindo o formato padrão Google Cloud / RFC 7807:

```json
{
  "error": {
    "code": 400,
    "message": "Invalid status transition from REJECTED to INTERVIEW.",
    "status": "INVALID_ARGUMENT",
    "details": []
  }
}
```

---

## 5. Próximos Passos de Desenvolvimento

Para prosseguir sequencialmente com a codificação do ecossistema, o backlog técnico recomendado segue a ordem abaixo:

1. **Modelagem de Domínio (`internal/core/domain`):** Criação das entidades `Application`, `Tag`, `Event` e `User` com seus status enumerados e encurtadores/construtores.
2. **Definição dos Contratos de Persistência (`internal/core/ports`):** Interfaces de serviços e repositórios (`ApplicationRepository`, `TagRepository`, `EventRepository`).
3. **Implementação do Adaptador GORM (`internal/adapters/storage/postgres`):** Modelagem das tabelas e chaves estrangeiras (`applications`, `tags`, `events`, `application_tags`) com suporte a migração automática.
4. **Codificação dos Casos de Uso (`internal/core/services`):** Lógica de transição de status no Kanban (gerando `Event` automático) e validação de permissão de `UserID`.
5. **Implementação HTTP e Middlewares (`internal/adapters/http`):** Middlewares de JWT e CORS, roteamento (`router.go`) e handlers RESTful padronizados.
