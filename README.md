# FIAP X - Video Service

Microsservico de entrada HTTP do ecossistema **FIAP X**. E o ponto de contato entre o cliente e o pipeline de processamento: recebe uploads autenticados, armazena os videos, dispara o processamento e entrega os resultados.

---

## Como Funciona

1. O cliente envia um video autenticado via JWT (Keycloak)
2. O servico valida o token, o arquivo e o formato
3. O video e armazenado no MinIO (bucket `uploads`)
4. Um registro e criado no PostgreSQL com status `PENDING`
5. Um evento e publicado no RabbitMQ para o processing-service processar
6. O processing-service processa o video e publica o resultado de volta
7. O video-service consome o resultado e atualiza o status no banco (`COMPLETED` ou `FAILED`)
8. O cliente pode listar, detalhar e baixar o ZIP dos frames via URL presignada

---

## Regras de Funcionamento

### Autenticacao

- Todos os endpoints (exceto `/health`) exigem um token JWT valido no header `Authorization: Bearer <token>`
- Os tokens sao emitidos pelo Keycloak e validados com RS256 via JWKS
- As chaves publicas do Keycloak sao cacheadas localmente por 1 hora para evitar chamadas desnecessarias
- Cada usuario so enxerga e manipula seus proprios videos - nao e possivel acessar videos de outros usuarios

### Upload

- Formatos aceitos: `mp4`, `avi`, `mov`, `mkv`, `wmv`, `flv`, `webm`
- Tamanho maximo configuravel (padrao: 500 MB)
- O campo do formulario multipart deve se chamar `video`
- O arquivo e enviado diretamente para o MinIO - nao e salvo em disco no servico

### Status do Video

Os videos passam pelo seguinte ciclo de vida:

PENDING -> PROCESSING -> COMPLETED
                     \-> FAILED

- `PENDING` - video recebido, aguardando processamento
- `PROCESSING` - processing-service esta trabalhando no video
- `COMPLETED` - frames extraidos e ZIP disponivel para download
- `FAILED` - processamento falhou; mensagem de erro disponivel no detalhe do video

### Download

- O endpoint de download so funciona para videos com status `COMPLETED`
- Videos `PENDING` ou `PROCESSING` retornam `409 Conflict`
- Videos `FAILED` retornam `422 Unprocessable Entity`
- A URL retornada e presignada e expira em **15 minutos**
- O ZIP contem os frames extraidos em formato PNG (1 frame por segundo)

### Delete

- Ao deletar um video, o arquivo original no MinIO e removido
- O registro no banco e excluido
- Se o video ja tiver sido processado, o ZIP permanece no MinIO ate ser limpo manualmente

### Cache

- A listagem de videos por usuario e cacheada no Redis com TTL de **60 segundos**
- O cache e invalidado automaticamente apos upload ou delete
- Se o Redis estiver indisponivel, o servico continua funcionando lendo diretamente do banco

### Mensageria

- O servico declara a exchange e as filas de forma idempotente na inicializacao
- Nao ha dependencia de ordem de inicializacao com o processing-service
- A fila de status (`video.status`) e consumida em goroutine dedicada
- Mensagens invalidas ou sem correspondencia no banco sao descartadas com log de erro

### Desligamento

- Ao receber SIGINT ou SIGTERM, o servico para de aceitar novas conexoes
- Aguarda a conclusao das requisicoes em andamento antes de encerrar

---

## API REST

Porta padrao: **8082**. Todos os endpoints exigem `Authorization: Bearer <token>`, exceto `/health`.

| Metodo   | Rota                       | Descricao                                         |
| -------- | -------------------------- | ------------------------------------------------- |
| `GET`    | `/health`                  | Health check (sem autenticacao)                   |
| `POST`   | `/api/videos/upload`       | Upload de video (multipart, campo `video`)        |
| `GET`    | `/api/videos`              | Listar todos os videos do usuario autenticado     |
| `GET`    | `/api/videos/:id`          | Detalhar um video por ID                          |
| `GET`    | `/api/videos/:id/download` | Obter URL presignada do ZIP (so para COMPLETED)   |
| `DELETE` | `/api/videos/:id`          | Deletar um video                                  |

---

## Estrutura do Projeto

O projeto segue **Clean Architecture** com separacao em camadas:

```
fiapx-video-service/
├── cmd/server/main.go              # Ponto de entrada - conecta todas as dependencias
├── internal/
│   ├── domain/                     # Regras de negocio (sem dependencias externas)
│   │   ├── entity/
│   │   │   ├── video.go            # Entidade Video (ciclo de vida PENDING > COMPLETED/FAILED)
│   │   │   └── message.go          # Estrutura das mensagens de entrada e saida no RabbitMQ
│   │   └── port/                   # Contratos (interfaces) que a infra deve implementar
│   ├── usecase/                    # Casos de uso - orquestram dominio e infra
│   │   ├── upload_video.go         # Valida, armazena, persiste e publica evento
│   │   ├── list_videos.go          # Lista com cache-aside (Redis -> PostgreSQL)
│   │   ├── get_video.go            # Busca por ID com verificacao de ownership
│   │   ├── download_video.go       # Gera URL presignada com validacao de status
│   │   ├── delete_video.go         # Remove do MinIO e do banco, invalida cache
│   │   └── status_handler.go       # Processa resultados publicados pelo processing-service
│   └── infra/                      # Implementacoes concretas
│       ├── config/                 # Carregamento de variaveis de ambiente
│       ├── postgres/               # Persistencia de videos com pgxpool (sem ORM)
│       ├── rabbitmq/               # Publisher (video.processing) e Consumer (video.status)
│       ├── minio/                  # Upload, delete e geracao de presigned URLs
│       ├── redis/                  # Cache-aside de listagem com TTL 60s
│       ├── keycloak/               # Validacao JWT RS256 com cache de JWKS (1h)
│       ├── metrics/                # Metricas Prometheus
│       ├── tracing/                # Rastreamento distribuido (OpenTelemetry/Jaeger)
│       └── http/                   # Servidor Gin, handlers e middleware JWT
├── migrations/                     # Scripts SQL de criacao do banco
├── pkg/logger/                     # Logger JSON estruturado (Zap)
├── tests/integration/              # Testes de ponta a ponta com testcontainers
├── Dockerfile                      # Imagem Docker multi-stage
├── Makefile                        # Comandos uteis
└── .env.example                    # Variaveis de ambiente com valores padrao
```

---

## Como Executar

### Pre-requisitos

- Docker
- Infraestrutura FIAP X rodando (`fiapx-infra-docker`)

### Com Docker

```bash
make docker-build
make docker-run
```

### Local (desenvolvimento)

Requer Go 1.22+.

```bash
make build
make run
```

---

## Configuracao

Todas as configuracoes sao feitas via variaveis de ambiente. Veja [.env.example](.env.example) para a lista completa com valores padrao.

---

## Observabilidade

- **Health check:** `GET http://localhost:8082/health`
- **Metricas Prometheus:** `http://localhost:8083/metrics`
- **Tracing:** OpenTelemetry exportando para Jaeger (opcional, nao bloqueia a inicializacao)

Metricas expostas:

| Metrica                                     | Tipo      | Descricao                               |
| ------------------------------------------- | --------- | --------------------------------------- |
| `fiapx_videos_uploaded_total`               | Counter   | Total de uploads realizados com sucesso |
| `fiapx_videos_status_updated_total{status}` | Counter   | Atualizacoes de status por tipo         |
| `fiapx_video_upload_duration_seconds`       | Histogram | Duracao do fluxo de upload              |
| `fiapx_video_active_uploads`                | Gauge     | Uploads em andamento no momento         |

---

## Testes

```bash
make test               # Unitarios
make test-integration   # Ponta a ponta (requer Docker)
```

Os testes de integracao sobem PostgreSQL, MinIO e RabbitMQ automaticamente via containers temporarios, realizam um upload completo e validam que o registro foi criado corretamente e o evento publicado na fila.
