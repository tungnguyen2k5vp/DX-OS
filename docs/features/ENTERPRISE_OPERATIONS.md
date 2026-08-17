# Vận hành doanh nghiệp: procure-to-pay, kiểm soát và quản trị

## 1. Luồng procure-to-pay đang chạy

1. Employee tạo phiếu, đính kèm chứng từ và gửi duyệt.
2. Department Manager duyệt trong phòng ban; hệ thống giữ ngân sách.
3. Finance duyệt cấp tổ chức; ngân sách chuyển sang committed.
4. Finance chọn nhà cung cấp và phát hành purchase order.
5. Finance có thể sửa hoặc hủy đơn trước khi phát sinh biên nhận/hóa đơn.
6. Requester hoặc Manager cùng phòng ghi từng lần nhận hàng và số lượng từng dòng. Kết quả hỗ trợ nhận một phần, nhận đủ, hư hỏng, sai hàng hoặc từ chối.
7. Finance có thể ghi nhiều hóa đơn cho một đơn hàng. Hệ thống đối chiếu tổng hóa đơn với giá trị order và phân biệt `PARTIAL_MATCH`, `MATCHED`, vượt tiền hoặc sai tiền tệ.
8. Finance xác minh hóa đơn rồi ghi nhiều lần thanh toán; hệ thống chặn vượt số tiền còn lại và tự chuyển `PAID` khi đủ.
9. Auditor xem toàn bộ bằng chứng, mở hồ sơ phát hiện, theo dõi khắc phục và xuất evidence package JSON.
10. DX Admin quản lý hồ sơ người dùng, trạng thái truy cập, phòng ban và theo dõi notification backlog.
11. AI Operator quét các luật rủi ro có thể giải thích và phải ghi quyết định của con người; khuyến nghị không tự thay đổi dữ liệu nghiệp vụ.

## 2. Trạng thái hóa đơn

| Trạng thái | Ý nghĩa | Thao tác tiếp theo |
|---|---|---|
| `RECORDED` | Đã nhập hóa đơn | Sửa, tranh chấp hoặc xác minh nếu match |
| `DISPUTED` | Có sai lệch/cần xử lý với nhà cung cấp | Sửa dữ liệu, sau đó mở lại |
| `VERIFIED` | Đã qua đối soát ba bên | Ghi nhận thanh toán |
| `PAID` | Đã thanh toán và giữ payment reference | Chỉ đọc/audit |

Match status gồm `WAITING_RECEIPT`, `CURRENCY_MISMATCH`, `AMOUNT_MISMATCH`, `PARTIAL_MATCH`, `MATCHED`. Tiền dùng PostgreSQL `numeric(19,4)` và decimal string xuyên suốt Go/JSON/Angular.

## 3. Kiểm soát giao nhận

- Trạng thái order: `ORDERED`, `PARTIALLY_RECEIVED`, `RECEIPT_EXCEPTION`, `RECEIVED`, `CANCELLED`.
- Mỗi biên nhận có idempotency key và chi tiết số lượng/condition theo dòng phiếu.
- Tổng đã nhận không được vượt số lượng đặt; ngoại lệ luôn giữ lịch sử và audit.
- Chỉ requester hoặc manager cùng phòng xác nhận hàng; Finance không tự xác nhận order do mình điều phối.

## 4. Hồ sơ nhà cung cấp

Finance quản lý địa chỉ, ngân hàng, tài khoản, hợp đồng/hạn hợp đồng, trạng thái tuân thủ, điểm hiệu suất và ghi chú. Auditor chỉ đọc. Nhà cung cấp `BLOCKED`, hợp đồng hết hạn hoặc rủi ro cao được đưa vào khuyến nghị kiểm soát.

## 5. Notification/outbox

- Mutation nghiệp vụ enqueue `outbox_events` trong cùng PostgreSQL transaction.
- Worker claim batch bằng `FOR UPDATE SKIP LOCKED`, materialize notification và đánh dấu processed trong một transaction.
- Audience có thể là user, role và department; actor bị loại khỏi audience.
- UI có unread badge, danh sách, đánh dấu một/tất cả đã đọc.
- Worker retry event lỗi; production cần alert theo backlog và `attempts`.

## 6. Trung tâm chính sách

- URL: `http://localhost:4200/policies`.
- `dx_admin`: sửa SLA và quy tắc chứng từ bằng `expectedVersion`; thay đổi được audit.
- `auditor`: chỉ đọc.
- Role khác: backend trả `403`.
- SLA mới áp dụng khi submit/resubmit; không viết lại deadline lịch sử.

## 7. Kiểm toán, quản trị và khuyến nghị

- `/audit`: audit log, hồ sơ khắc phục và xuất evidence package.
- `/admin`: thống kê vận hành, người dùng và cây phòng ban. Vai trò vẫn do Keycloak quản lý tập trung.
- `/ai-center`: khuyến nghị SLA, giá trị lớn và rủi ro nhà cung cấp; mọi quyết định đều có version và audit log.
- Các API đặc quyền kiểm tra role tại backend; employee gọi trực tiếp nhận `403`.

## 8. Rate limit

- Mọi route `/api/v1` sau xác thực có quota 120 request/phút/Keycloak subject.
- Vượt quota trả `429`, `Retry-After`, `X-RateLimit-Limit`, `X-RateLimit-Remaining` và `X-RateLimit-Reset`.
- `/health/live` và `/health/ready` không dùng quota người dùng.
- State limiter hiện ở RAM của API; khi chạy nhiều replica phải dùng Redis hoặc API gateway để có quota toàn cụm.

## 9. Bằng chứng kiểm thử

Chạy `scripts/Test-ProcurementWorkflow.ps1`. Script kiểm tra positive/negative RBAC, stale version, policy admin/read-only, notification isolation, order/receipt, chặn invoice trước receipt, verify/pay và audit. Credential/token sinh ra nằm trong `data/runtime` (Git ignored); token ngắn hạn được xóa cuối script.
