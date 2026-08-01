# Vận hành và quan sát hệ thống

## 1. Logging

Go log JSON với:

- timestamp;
- level;
- service/version/environment;
- request/correlation ID;
- actor ID dạng nội bộ hoặc subject đã chuẩn hóa;
- route/status/latency;
- resource type/id khi phù hợp;
- error code, không phải raw database error.

Không log:

- Authorization header/access token;
- password/client secret/API key;
- full document/prompt nhạy cảm;
- raw file;
- PII không cần thiết.

Angular error report không gửi token hoặc form content nhạy cảm.

## 2. Metrics

### Go API

- request rate, error rate, latency;
- in-flight request;
- database pool usage/wait;
- transition count theo action/result;
- authorization denied;
- outbox pending/dead;
- dependency latency/error;
- Agent tool success/failure.

### Infrastructure

- CPU/RAM/disk;
- container restart;
- PostgreSQL connection/size/slow query;
- Keycloak login success/failure;
- Nextcloud storage;
- RAGFlow ingestion queue và dependency health.

Metric label không dùng user ID/request ID để tránh cardinality cao.

## 3. Tracing

Correlation ID đi qua Angular -> reverse proxy -> Go -> integration. OpenTelemetry trace được dùng khi
có collector; nếu chưa, structured log vẫn phải truy vết được.

## 4. Health

- Liveness chỉ kiểm process.
- Readiness kiểm database và dependency bắt buộc.
- Dependency tùy chọn không làm API hoàn toàn unready; feature trả degraded/503 cụ thể.
- Health endpoint không yêu cầu auth trong mạng nội bộ nhưng không lộ chi tiết.

## 5. Backup

| Dữ liệu | Cách backup |
|---|---|
| DX-OS PostgreSQL | logical dump + hạ tầng snapshot nếu có |
| Keycloak PostgreSQL | dump riêng |
| Nextcloud | database + data directory + config đồng nhất |
| Metabase app DB | dump |
| RAGFlow | theo upstream cho DB/object/document engine |
| Config | Git + secret manager backup theo policy |

Backup phải mã hóa, có retention và checksum. “Có file backup” chưa đủ; phải restore drill.

## 6. Restore drill

1. tạo môi trường cô lập;
2. xác nhận backup version/checksum;
3. restore database/file;
4. khởi động đúng image version;
5. chạy migration nếu quy trình yêu cầu;
6. smoke test login, request, attachment, dashboard và RAG;
7. ghi thời gian thực tế và lỗi;
8. xóa an toàn môi trường drill.

## 7. RTO/RPO định hướng

Cho Demo/UAT:

- RPO: có thể reset về fixture;
- RTO: trong một buổi demo preparation.

Production pilot phải chốt số cụ thể với đơn vị sử dụng trước triển khai; không suy từ giá trị demo.

## 8. Alert tối thiểu

- API 5xx/error rate tăng.
- p95 latency vượt ngưỡng.
- database disk/pool gần giới hạn.
- outbox dead > 0.
- backup thất bại.
- Keycloak login failure bất thường.
- Agent tool failure hoặc denied spike.
- certificate sắp hết hạn.

Mỗi alert có owner, severity và runbook; không tạo alert không ai chịu trách nhiệm.

## 9. Incident flow

```text
detect -> triage -> contain -> recover -> verify -> communicate -> postmortem
```

Ưu tiên:

1. ngăn mất/rò dữ liệu;
2. tắt Agent execution nếu nghi sai hành động;
3. chuyển feature về read-only/degraded khi phù hợp;
4. giữ log/evidence;
5. restore/rollback;
6. viết postmortem và action item.

## 10. Bảo trì

- Review dependency/image định kỳ.
- Upgrade patch bảo mật qua Dev -> UAT -> Pilot.
- Rehearse restore theo lịch.
- Xoay secret/service account.
- Review role và user lifecycle.
- Archive audit theo retention.
- Đánh giá lại RAG quality/prompt injection set khi knowledge thay đổi.

