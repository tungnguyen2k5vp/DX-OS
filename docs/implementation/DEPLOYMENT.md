# Triển khai hệ thống

## 1. Mục tiêu môi trường

| Môi trường | Mục tiêu | Dữ liệu |
|---|---|---|
| Local | phát triển từng feature | giả lập cá nhân |
| Dev/Integration | tích hợp chung | test dùng chung |
| Demo/UAT | nghiệm thu và trình diễn | fixture cố định, reset được |
| Production pilot | người dùng giới hạn | dữ liệu đã phê duyệt |

Không đưa dữ liệu production về local.

## 2. Compose profiles mục tiêu

| Profile | Thành phần |
|---|---|
| `foundation` | PostgreSQL, Keycloak |
| `app` | Go API, worker, Angular/reverse proxy |
| `human` | Nextcloud |
| `data` | Metabase và job curated |
| `ai` | RAGFlow stack upstream |
| `monitoring` | log/metric collector và dashboard |

Trên máy 16 GB chỉ bật foundation + app; RAGFlow chạy riêng hoặc chỉ bật khi demo AI.

## 3. Build artifacts

### Backend

Multi-stage Docker build:

1. builder dùng Go version pin;
2. `go mod download`;
3. `CGO_ENABLED=0 go build` với version/commit metadata;
4. runtime distroless/alpine phù hợp;
5. chạy non-root;
6. expose API port;
7. health check ngoài container hoặc endpoint.

### Frontend

1. builder dùng Node version tương thích Angular;
2. `npm ci`;
3. `npm run build`;
4. copy static output sang Nginx;
5. Nginx fallback SPA route về `index.html`;
6. cache immutable hashed assets, không cache dài `index.html`.

Mỗi image gắn version/commit, không chỉ `latest`.

## 4. Routing production

Khuyến nghị:

| Host | Đích |
|---|---|
| `app.example.com` | Angular và `/api` đến Go |
| `sso.example.com` | Keycloak |
| `files.example.com` | Nextcloud |
| `bi.example.com` | Metabase |
| RAGFlow | internal/VPN; không public nếu không cần |

Reverse proxy:

- TLS 1.2+;
- HSTS sau khi domain ổn định;
- security headers/CSP;
- request/body limit;
- trusted proxy/IP đúng;
- WebSocket chỉ bật route cần;
- access log có request ID nhưng không có token/query nhạy cảm.

## 5. Deployment order

```text
1. backup và preflight
2. pull image/version mới
3. chạy database migration job
4. deploy Keycloak config thay đổi tương thích
5. deploy Go API + worker
6. smoke test API
7. deploy Angular
8. chạy E2E smoke
9. theo dõi error/latency
10. ghi release evidence
```

Migration phải backward-compatible với phiên bản app đang chạy trong thời gian rollout.

## 6. Configuration và secret

- `.env.example` chỉ mô tả tên biến.
- Dev có thể dùng Docker secret/env protected.
- Pilot dùng secret manager của hạ tầng.
- Không bake secret vào Angular; mọi giá trị frontend đều public.
- Angular runtime config chỉ chứa public API URL, issuer và client ID.
- Database/Keycloak/Nextcloud/RAGFlow secret chỉ ở server.

## 7. Database

- Không expose PostgreSQL ra internet.
- Mỗi service có database role riêng.
- Pool size dựa trên connection limit.
- Migration account có quyền DDL; runtime account chỉ quyền cần thiết.
- Metabase chỉ đọc curated schema.

## 8. Keycloak production

Không dùng `start-dev`. Cần:

- hostname/public URL đúng;
- proxy headers đúng;
- HTTPS;
- external PostgreSQL;
- health/metrics;
- realm/client export được quản lý;
- admin console giới hạn mạng;
- backup database trước nâng cấp.

## 9. RAGFlow

RAGFlow dùng Compose upstream đúng version vì phụ thuộc document engine, MySQL, object storage và
Redis. Không copy riêng container RAGFlow vào base compose mà bỏ dependencies.

- Chạy trên máy đủ RAM/disk.
- API key từ secret.
- Chỉ ingest tài liệu được owner cho phép.
- Backup knowledge configuration và nguồn gốc tài liệu.
- Tool execution đi qua Go API, không cấp database/shell.

## 10. Rollback

### App-only

- giữ image version trước;
- rollback Angular/Go nếu migration tương thích ngược.

### Migration không tương thích

- dừng write;
- restore backup hoặc chạy forward fix đã kiểm thử;
- không chạy down migration tùy tiện.

### Keycloak/platform upgrade

- theo migration guide upstream;
- backup trước;
- test realm/client flow trên Dev/UAT.

## 11. Production readiness gate

- TLS/domain và secret manager.
- RBAC/ownership negative test pass.
- Backup và restore drill pass.
- Dependency/license/SBOM review.
- Observability dashboard và alert owner.
- RTO/RPO được chốt.
- Data classification/retention được duyệt.
- Runbook incident và contact list.

