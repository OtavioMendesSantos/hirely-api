https://docs.cloud.google.com/apis/design

docker build -t hirely-api:latest .
docker run -p 8080:8080 hirely-api:latest

```txt
hirely-api/
├── cmd/
│   └── api/
│       └── main.go                 # Ponto de entrada, configuração do UTC e injeção de dependências
├── internal/
│   ├── core/                       # Regras de negócio (Hexágono Interno)
│   │   ├── domain/                 # Entidades puras (ex: candidatura.go)
│   │   ├── ports/                  # Interfaces de Repositórios (saída) e Serviços (entrada)
│   │   └── services/               # Implementação dos casos de uso (ex: candidatura_service.go)
│   └── adapters/                   # Infraestrutura e Transporte (Hexágono Externo)
│       ├── http/                   # Camada de API
│       │   ├── router.go           # Roteador principal (Padrão de rotas /v1/)
│       │   ├── health_handler.go   # Endpoints de infraestrutura
│       │   ├── candidatura_handler.go # Operações padrão (Create, List, Get, Update, Delete)
│       │   └── messages.go         # DTOs rigorosos de Request/Response do Google API
│       └── storage/                # Implementações de banco de dados
│           └── postgres/           # Adaptador GORM
│               ├── repository.go   # Implementação concreta das portas de persistência
│               └── model.go        # Structs com tags do GORM (CandidaturaDB)
├── go.mod
└── go.sum
```
