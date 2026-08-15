# Vận hành doanh nghiệp: thông báo, hóa đơn và chính sách

## 1. Luồng procure-to-pay đang chạy

1. Employee tạo phiếu, đính kèm chứng từ và gửi duyệt.
2. Department Manager duyệt trong phòng ban; hệ thống giữ ngân sách.
3. Finance duyệt cấp tổ chức; ngân sách chuyển sang committed.
4. Finance chọn nhà cung cấp và phát hành purchase order.
5. Finance có thể ghi hóa đơn, nhưng chưa được xác minh trước khi người yêu cầu xác nhận nhận hàng.
6. Requester hoặc Manager cùng phòng xác nhận biên nhận; Finance không được tự nhận hàng cho order mình quản lý.
7. Hệ thống tính đối soát từ dữ liệu hiện tại: trạng thái nhận hàng, currency và amount. Không lưu một bản match state độc lập.
8. Finance xác minh hóa đơn khớp rồi ghi mã/ngày thanh toán. Requester nhận thông báo `INVOICE_PAID`.
9. Auditor xem order, hóa đơn, ngân sách, policy và audit ở chế độ chỉ đọc.

## 2. Trạng thái hóa đơn

| Trạng thái | Ý nghĩa | Thao tác tiếp theo |
|---|---|---|
| `RECORDED` | Đã nhập hóa đơn | Sửa, tranh chấp hoặc xác minh nếu match |
| `DISPUTED` | Có sai lệch/cần xử lý với nhà cung cấp | Sửa dữ liệu, sau đó mở lại |
| `VERIFIED` | Đã qua đối soát ba bên | Ghi nhận thanh toán |
| `PAID` | Đã thanh toán và giữ payment reference | Chỉ đọc/audit |

Match status gồm `WAITING_RECEIPT`, `CURRENCY_MISMATCH`, `AMOUNT_MISMATCH`, `MATCHED`. Tiền dùng PostgreSQL `numeric(19,4)` và decimal string xuyên suốt Go/JSON/Angular.

## 3. Notification/outbox

- Mutation nghiệp vụ enqueue `outbox_events` trong cùng PostgreSQL transaction.
- Worker claim batch bằng `FOR UPDATE SKIP LOCKED`, materialize notification và đánh dấu processed trong một transaction.
- Audience có thể là user, role và department; actor bị loại khỏi audience.
- UI có unread badge, danh sách, đánh dấu một/tất cả đã đọc.
- Worker retry event lỗi; production cần alert theo backlog và `attempts`.

## 4. Trung tâm chính sách

- URL: `http://localhost:4200/policies`.
- `dx_admin`: sửa SLA và quy tắc chứng từ bằng `expectedVersion`; thay đổi được audit.
- `auditor`: chỉ đọc.
- Role khác: backend trả `403`.
- SLA mới áp dụng khi submit/resubmit; không viết lại deadline lịch sử.

## 5. Rate limit

- Mọi route `/api/v1` sau xác thực có quota 120 request/phút/Keycloak subject.
- Vượt quota trả `429`, `Retry-After`, `X-RateLimit-Limit`, `X-RateLimit-Remaining` và `X-RateLimit-Reset`.
- `/health/live` và `/health/ready` không dùng quota người dùng.
- State limiter hiện ở RAM của API; khi chạy nhiều replica phải dùng Redis hoặc API gateway để có quota toàn cụm.

## 6. Bằng chứng kiểm thử

Chạy `scripts/Test-ProcurementWorkflow.ps1`. Script kiểm tra positive/negative RBAC, stale version, policy admin/read-only, notification isolation, order/receipt, chặn invoice trước receipt, verify/pay và audit. Credential/token sinh ra nằm trong `data/runtime` (Git ignored); token ngắn hạn được xóa cuối script.
