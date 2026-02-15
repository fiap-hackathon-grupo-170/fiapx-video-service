# 🎥 FIAP X - Video Service

Microsserviço responsável pelo **Bounded Context de Vídeos**.

## 📦 Bounded Context: VÍDEOS

**Entidade Principal:** `Video`

**Responsabilidades:**
- ✅ Receber upload de vídeos
- ✅ Armazenar metadados no PostgreSQL
- ✅ Salvar vídeo no MinIO/S3
- ✅ Publicar evento `VideoUploaded`
- ✅ Consumir eventos de status (`ProcessingCompleted`, `ProcessingFailed`)
- ✅ Gerar presigned URL para download do ZIP
- ✅ Listar vídeos do usuário
- ✅ Deletar vídeos

---

## 🏗️ Arquitetura (Clean Architecture)

```
src/main/java/com/fiapx/video/
├── domain/                          # CAMADA DE DOMÍNIO
│   ├── entity/
│   │   ├── Video.java              # Entidade com regras de negócio
│   │   └── VideoStatus.java        # Enum de status
│   ├── repository/
│   │   └── VideoRepository.java    # Interface
│   ├── service/
│   │   └── VideoService.java       # Serviço de domínio
│   └── event/
│       ├── VideoUploadedEvent.java
│       ├── ProcessingCompletedEvent.java
│       └── ProcessingFailedEvent.java
│
├── application/                     # CASOS DE USO
│   ├── usecase/
│   │   ├── UploadVideoUseCase.java
│   │   ├── ListVideosUseCase.java
│   │   ├── GetVideoUseCase.java
│   │   ├── DownloadVideoUseCase.java
│   │   └── DeleteVideoUseCase.java
│   └── dto/
│       ├── VideoUploadRequest.java
│       ├── VideoResponse.java
│       └── VideoDetailResponse.java
│
└── infrastructure/                  # FRAMEWORKS & DRIVERS
    ├── persistence/
    │   └── JpaVideoRepository.java
    ├── storage/
    │   └── MinioStorageService.java
    ├── messaging/
    │   ├── RabbitMQEventPublisher.java
    │   └── VideoStatusConsumer.java
    ├── cache/
    │   └── RedisCacheService.java
    ├── security/
    │   └── SecurityConfig.java
    ├── web/
    │   ├── VideoController.java
    │   └── GlobalExceptionHandler.java
    └── config/
        ├── RabbitMQConfig.java
        ├── MinioConfig.java
        └── RedisConfig.java
```

---

## 📡 Endpoints

### Upload de Vídeo
```http
POST /api/videos/upload
Content-Type: multipart/form-data
Authorization: Bearer <JWT>

file: video.mp4

Response 202 Accepted:
{
  "id": "uuid",
  "filename": "video.mp4",
  "status": "UPLOADED",
  "uploadedAt": "2026-02-15T10:30:00Z"
}
```

### Listar Vídeos do Usuário
```http
GET /api/videos
Authorization: Bearer <JWT>

Response 200 OK:
[
  {
    "id": "uuid",
    "filename": "video.mp4",
    "status": "COMPLETED",
    "framesCount": 120,
    "size": "50.5 MB",
    "uploadedAt": "2026-02-15T10:30:00Z",
    "processedAt": "2026-02-15T10:32:00Z"
  }
]
```

### Detalhes do Vídeo
```http
GET /api/videos/{id}
Authorization: Bearer <JWT>

Response 200 OK:
{
  "id": "uuid",
  "filename": "video.mp4",
  "status": "COMPLETED",
  "framesCount": 120,
  "size": "50.5 MB",
  "uploadedAt": "2026-02-15T10:30:00Z",
  "processedAt": "2026-02-15T10:32:00Z",
  "zipAvailable": true
}
```

### Download do ZIP
```http
GET /api/videos/{id}/download
Authorization: Bearer <JWT>

Response 302 Found:
Location: https://minio:9000/fiapx-videos/zips/uuid.zip?X-Amz-...
```

### Deletar Vídeo
```http
DELETE /api/videos/{id}
Authorization: Bearer <JWT>

Response 204 No Content
```

---

## 🔄 Eventos

### Publica: `VideoUploaded`

```json
{
  "eventType": "VideoUploaded",
  "eventId": "uuid",
  "occurredAt": "2026-02-15T10:30:00Z",
  "videoId": "uuid",
  "userId": "cognito-sub",
  "videoS3Path": "uploads/uuid.mp4",
  "filename": "video.mp4",
  "sizeBytes": 52428800
}
```

**Exchange:** `fiapx.video`  
**Routing Key:** `video.uploaded`  
**Queue destino:** `video.processing`

### Consome: `ProcessingCompleted`

```json
{
  "eventType": "ProcessingCompleted",
  "eventId": "uuid",
  "occurredAt": "2026-02-15T10:32:00Z",
  "videoId": "uuid",
  "zipS3Path": "zips/uuid.zip",
  "framesCount": 120
}
```

### Consome: `ProcessingFailed`

```json
{
  "eventType": "ProcessingFailed",
  "eventId": "uuid",
  "occurredAt": "2026-02-15T10:32:00Z",
  "videoId": "uuid",
  "errorMessage": "FFmpeg error: ...",
  "retryable": true
}
```

---

## 💾 Banco de Dados

### PostgreSQL (:5432)

```sql
CREATE TABLE videos (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL,
    filename VARCHAR(500) NOT NULL,
    s3_path_original VARCHAR(1000) NOT NULL,
    zip_s3_path VARCHAR(1000),
    size_bytes BIGINT NOT NULL,
    mime_type VARCHAR(100) NOT NULL,
    status VARCHAR(50) NOT NULL,
    frames_count INTEGER,
    error_message TEXT,
    uploaded_at TIMESTAMP NOT NULL,
    processed_at TIMESTAMP,
    updated_at TIMESTAMP,
    
    INDEX idx_user_id (user_id),
    INDEX idx_status (status),
    INDEX idx_uploaded_at (uploaded_at DESC)
);
```

---

## 🚀 Como Executar

### Desenvolvimento Local

```bash
# 1. Subir dependências
cd ../fiapx-infra-docker
docker-compose up -d postgres-videos minio rabbitmq redis keycloak

# 2. Configurar variáveis
export DB_HOST=localhost
export DB_PORT=5432
export MINIO_ENDPOINT=http://localhost:9000
export RABBITMQ_HOST=localhost
export KEYCLOAK_ISSUER_URI=http://localhost:8081/realms/fiapx

# 3. Rodar aplicação
mvn spring-boot:run
```

### Docker

```bash
# Build
docker build -t fiapx/video-service:latest .

# Run
docker run -p 8082:8082 \
  -e DB_HOST=postgres-videos \
  -e MINIO_ENDPOINT=http://minio:9000 \
  -e RABBITMQ_HOST=rabbitmq \
  fiapx/video-service:latest
```

---

## 🧪 Testes

```bash
# Unit tests
mvn test

# Integration tests
mvn verify

# Coverage
mvn jacoco:report
# Report: target/site/jacoco/index.html
```

---

## 📊 Métricas

**Prometheus:** http://localhost:8082/actuator/prometheus

**Métricas principais:**
- `video_uploads_total` - Total de uploads
- `video_upload_duration_seconds` - Tempo de upload
- `video_processing_completed_total` - Vídeos processados
- `video_processing_failed_total` - Falhas

**Grafana Dashboard:** Ver `../fiapx-infra-docker/grafana/dashboards/`

---

## 🔐 Autenticação

Este serviço valida JWT emitido pelo **Keycloak**.

**Claims esperados:**
- `sub` - User ID
- `email` - Email do usuário
- `realm_access.roles` - Roles

**Configuração:**
```yaml
spring:
  security:
    oauth2:
      resourceserver:
        jwt:
          issuer-uri: http://localhost:8081/realms/fiapx
```

---

## 📝 TODO / Melhorias

- [ ] Implementar cache Redis para lista de vídeos
- [ ] Adicionar paginação nos endpoints de listagem
- [ ] Implementar filtros (por status, data)
- [ ] Adicionar webhook para notificar cliente
- [ ] Implementar retry com backoff exponencial
- [ ] Adicionar testes de contrato (Pact)

---

**Desenvolvido para o Hackathon FIAP X** 🎬
