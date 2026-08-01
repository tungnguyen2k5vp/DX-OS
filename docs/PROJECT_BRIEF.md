# Project Brief

## Bài toán

DX-OS là nền tảng vận hành doanh nghiệp hợp nhất, không phải hệ điều hành máy tính. Giá trị nằm ở
việc các ứng dụng dùng chung định danh, quy trình, dữ liệu và ngữ cảnh AI.

Kịch bản dọc của MVP là **đề nghị mua sắm nội bộ** vì đủ để chứng minh toàn bộ H-P-D-I:
đăng nhập, tạo hồ sơ, validation, phê duyệt nhiều bước, tài liệu, KPI, RAG, Agent có kiểm soát và
audit.

## Kết quả cần chứng minh

```text
SSO -> tạo phiếu -> kiểm tra -> phê duyệt -> dữ liệu curated
    -> dashboard -> AI đề xuất -> người duyệt -> tool API -> audit
```

## Giả định làm việc

- Dự án là lab/đồ án với nhóm nhỏ và khung thời gian khoảng 10 tuần.
- Dưới 100 người dùng demo, dữ liệu ở quy mô GB thấp, tải giao dịch thấp.
- Một máy phát triển hiện có khoảng 16 GB RAM; các dịch vụ phải chạy theo profile.
- Dùng LLM/embedding API bên ngoài trong MVP; không chạy LLM local.
- Môi trường đầu tiên là local/dev với dữ liệu giả lập, không chứa dữ liệu cá nhân thật.
- PostgreSQL dùng một cluster với database và role riêng cho từng dịch vụ ở giai đoạn lab.

Các giả định này phải được xem lại trước pilot/production.

## Phạm vi tự xây

- Angular SPA: dashboard, form, danh sách, approval inbox, AI và audit UI.
- Go modular monolith: REST API, state machine, rule, RBAC/data scope và integration.
- Realm, client, role mapping và chính sách Keycloak.
- PostgreSQL schema, migration, outbox, curated model và data quality.
- Adapter Nextcloud, Metabase và RAGFlow.
- OpenAPI, event contract, retry, idempotency và audit.
- Tool gateway có allowlist, backend authorization và human approval.
- Compose, seed/reset, test, backup/restore và tài liệu.

## Không tự xây

- Identity provider, file server, BI engine, workflow engine hoặc RAG engine mới.
- NocoBase/Flowable trong MVP; quy trình được code tường minh trong Go.
- Microservices tùy ý khi chưa có nhu cầu scale độc lập.
- Agent có database account, shell access hoặc quyền quản trị Keycloak.
- Event sourcing/CQRS/Kafka trong MVP.

## Definition of Done chung

- Cấu hình/code có hướng dẫn chạy và không chứa secret.
- Chạy lại được trên môi trường sạch.
- Có test case và bằng chứng kết quả.
- Có health/log/correlation ID đủ để chẩn đoán.
- Tài liệu và ADR được cập nhật.
- Có rollback hoặc quy trình restore.
