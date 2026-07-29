# Hirely API

**Visão Geral**  
O Hirely é um aplicativo web voltado para a gestão, centralização e rastreabilidade de candidaturas a vagas de emprego. Desenvolvido com uma abordagem *Mobile-First*, o sistema otimiza o acompanhamento de processos seletivos através de uma interface baseada em Kanban, um sistema flexível de tags e uma timeline detalhada de interações.

## Principais Funcionalidades

* **Quadro Kanban (Job Applications):** Gerenciamento visual do fluxo de candidaturas por meio de estágios fixos (*To Apply*, *Applied*, *Interview*, *Offer*, *Rejected*).
* **Gestão de Identidade e Segurança:** Autenticação tradicional (hash seguro) e integração OAuth 2.0 (Google OIDC) garantindo o isolamento estrito de dados por usuário.
* **Timeline de Eventos (Auditoria):** Rastreabilidade completa da candidatura através da geração automática de logs do sistema e registro de notas manuais.
* **Sistema Dinâmico de Tags:** Customização de tags (nome e formato hexadecimal de cor) para categorização avançada de candidaturas.
* **Dashboard e Métricas:** Agregação de estatísticas para análise de desempenho, incluindo taxa de conversão do funil de vagas.

## Arquitetura e Engenharia

* **Stack Tecnológica:** Backend de alta performance em **Go (1.26+)**, framework **Gin**, banco de dados **PostgreSQL** e ORM **GORM**. (Interface de usuário construída em Angular).
* **Padrões de Projeto:** Estruturado em **Arquitetura Hexagonal (Ports & Adapters)**, garantindo o isolamento total das regras de negócio (*Domain*) contra acoplamentos de infraestrutura.
* **Design de API:** Padrão estrutural *Google API Design*, com endpoints RESTful hierárquicos, orientados a recursos e inicialização *Fail-Fast*.
* **Observabilidade:** *Structured Logging* nativo (`log/slog`) com saída estrita em JSON contendo identificadores únicos de requisição (`traceId`).

## Como Executar (Ambiente Local)

1. Crie o banco de dados PostgreSQL:
   ```bash
   psql -U postgres -c "CREATE DATABASE hirely;"
   ```
2. Configure as variáveis de ambiente:
   ```bash
   cp .env.example .env
   ```
3. Construa e execute via Docker:
   ```bash
   docker build -t hirely-api:latest .
   docker run -p 8080:8080 hirely-api:latest
   ```

*(Consulte a pasta `docs/` para ler a especificação técnica detalhada da API e payloads de integração)*
