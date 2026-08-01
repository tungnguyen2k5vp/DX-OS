# Thiết kế REST API

## 1. Consumer và style

Consumer chính là Angular SPA; consumer phụ là RAGFlow tool và script demo/test. MVP dùng REST,
OpenAPI 3.1 và JSON.

- Base path: `/api/v1`.
- Resource dùng danh từ số nhiều và kebab-case.
- Timestamp ISO-8601 UTC.
- ID UUID.
- Money truyền dạng chuỗi thập phân để tránh mất chính xác.
- Breaking change tạo version mới; additive change giữ v1.

## 2. Header chuẩn

| Header | Yêu cầu |
|---|---|
| `Authorization: Bearer <token>` | mọi endpoint ngoài health |
| `X-Correlation-ID` | client có thể gửi; server sinh nếu thiếu |
| `Idempotency-Key` | bắt buộc với transition và tool execution |
| `If-Match` hoặc `expectedVersion` | optimistic concurrency |
| `Content-Type: application/json` | request JSON |

Response luôn trả `X-Correlation-ID`. Không echo token hoặc secret.

## 3. HTTP semantics

| Trường hợp | Status |
|---|---|
| lấy/cập nhật thành công | 200 |
| tạo resource/transition | 201 |
| xóa/command không body | 204 |
| request sai cú pháp | 400 |
| chưa đăng nhập/token sai | 401 |
| không đủ quyền/scope | 403 |
| không tìm thấy | 404 |
| version/idempotency conflict | 409 |
| business precondition không đạt | 422 |
| rate limit | 429 |
| dependency tạm lỗi | 503 |

Lỗi theo `application/problem+json`:

```json
{
  "type": "https://docs.dx-os.local/problems/validation",
  "title": "Request validation failed",
  "status": 422,
  "detail": "Quotation is required for this amount.",
  "instance": "/api/v1/purchase-requests/7a.../transitions",
  "code": "QUOTATION_REQUIRED",
  "correlationId": "01J...",
  "errors": [
    { "field": "attachments", "code": "required", "message": "Cần ít nhất một báo giá." }
  ]
}
```

## 4. Pagination, filter và sort

MVP dùng page-based pagination:

```text
GET /purchase-requests?page=1&pageSize=20&status=SUBMITTED
```

- `page >= 1`.
- `1 <= pageSize <= 100`.
- Phiên bản hiện tại sắp xếp cố định theo `createdAt DESC`, sau đó `id DESC`.
- Search input được parameterize; không ghép SQL.

Response:

```json
{
  "items": [],
  "page": 1,
  "pageSize": 20,
  "total": 0,
  "pages": 0
}
```

## 5. Purchase request endpoints

| Method | Endpoint | Ý nghĩa |
|---|---|---|
| `GET` | `/purchase-requests` | danh sách theo scope |
| `POST` | `/purchase-requests` | tạo draft |
| `GET` | `/purchase-requests/{id}` | chi tiết + allowed actions |
| `PATCH` | `/purchase-requests/{id}` | sửa draft/changes requested |
| `POST` | `/purchase-requests/{id}/transitions` | submit/approve/reject/... |
| `GET` | `/purchase-requests/{id}/events` | timeline |
| `GET/POST` | `/purchase-requests/{id}/comments` | comment |
| `GET/POST` | `/purchase-requests/{id}/attachments` | metadata/file flow |

### Tạo draft

```json
POST /api/v1/purchase-requests
{
  "title": "Mua máy tính cho phòng kỹ thuật",
  "reason": "Thay thế thiết bị đã hết khấu hao",
  "currency": "VND",
  "costCenter": "IT-2026",
  "items": [
    {
      "description": "Laptop",
      "quantity": "2",
      "unit": "chiếc",
      "unitPrice": "25000000"
    }
  ]
}
```

Response `201` trả `id`, `requestCode`, `status`, `version`, `totalAmount`, timestamps và items.
`requesterId`/`departmentId` không nhận từ body; backend suy ra từ token và profile nghiệp vụ.

### Phạm vi đọc hiện tại

- `employee`: phiếu do chính user tạo.
- `department_manager`: toàn bộ phiếu trong phòng ban.
- `finance`: phiếu trong organization đã tới bước tài chính.
- `auditor`: đọc toàn bộ dữ liệu nghiệp vụ.
- `dx_admin` và `ai_operator`: không mặc nhiên có quyền đọc/tạo phiếu.

Phiếu ngoài scope trả `404` để không làm lộ sự tồn tại của resource.

### Transition

```json
POST /api/v1/purchase-requests/{id}/transitions
Idempotency-Key: 4db...

{
  "action": "APPROVE",
  "expectedVersion": 3,
  "comment": "Đồng ý theo báo giá đính kèm."
}
```

Action:

- `SUBMIT`
- `RESUBMIT`
- `APPROVE`
- `REJECT`
- `REQUEST_CHANGES`
- `CANCEL`

Server suy ra bước hiện tại và role cần thiết; client không gửi target status.

## 6. Attachment flow

Ưu tiên backend kiểm soát metadata/quyền:

1. Angular xin upload session/presigned flow từ Go.
2. Go kiểm quyền, loại file, kích thước và tạo path.
3. File đi tới Nextcloud qua adapter hoặc endpoint upload được giới hạn.
4. Go xác nhận checksum/etag và tạo `attachments`.

Không cho client tự chọn path tùy ý hoặc ghi trực tiếp vào folder khác.

## 7. Dashboard API

| Method | Endpoint |
|---|---|
| `GET` | `/analytics/process-performance` |
| `GET` | `/analytics/budget-usage` |
| `GET` | `/analytics/sla` |
| `GET` | `/me/tasks-summary` |

Query hỗ trợ `from`, `to`, `departmentId` theo scope. Metabase đọc curated schema bằng account
read-only; Angular có thể embed/link dashboard sau khi auth model được chốt.

## 8. AI/RAG API

| Method | Endpoint | Ý nghĩa |
|---|---|---|
| `POST` | `/ai/queries` | tạo câu hỏi RAG |
| `GET` | `/ai/queries/{id}` | kết quả/citation |
| `POST` | `/purchase-requests/{id}/agent-runs` | phân tích hồ sơ |
| `GET` | `/agent-recommendations` | danh sách recommendation |
| `GET` | `/agent-recommendations/{id}` | chi tiết |
| `POST` | `/agent-recommendations/{id}/decisions` | approve/reject |
| `POST` | `/agent-recommendations/{id}/executions` | execute đã được duyệt |

Decision và execution là hai resource riêng. Execute phải kiểm approval chưa hết hạn, input hash và
business precondition.

## 9. Health endpoints

- `/health/live`: process đang chạy, không phụ thuộc downstream.
- `/health/ready`: database và dependency bắt buộc sẵn sàng.
- `/metrics`: chỉ mạng nội bộ/monitoring; không public.

Health response không lộ DSN, secret hoặc stack trace.

## 10. OpenAPI workflow

1. Thiết kế/thay contract trong `contracts/openapi/dx-os-v1.yaml`.
2. Lint OpenAPI.
3. Generate/verify Angular types và Go server types nếu dùng.
4. Implement.
5. Chạy contract test.
6. Publish Swagger UI chỉ ở dev hoặc có auth.

## 11. Security API

- Body limit theo endpoint; upload có limit riêng.
- Timeout server/client rõ ràng.
- CORS allowlist, không `*` với credential.
- Rate limit AI/tool và endpoint nhạy cảm.
- Validate URL/path để tránh SSRF/path traversal.
- Không nhận role, requester hoặc department từ body nếu có thể suy từ authenticated principal.
