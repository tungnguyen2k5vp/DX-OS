# DX-OS drift registry

Registry này ghi nhận khác biệt giữa tài liệu kiến trúc và code đang chạy. Mục tiêu là giúp nhóm 4 người biết phần nào là hiện trạng, phần nào mới là định hướng, tránh demo nhầm chức năng chưa tồn tại.

## 1. Findings view

| ID | Trạng thái | Mức độ | Bằng chứng | Ảnh hưởng | Hướng xử lý |
|---|---|---|---|---|---|
| DRIFT-001 | Resolved | High | Migration `000009`, package `internal/notifications`, worker claim bằng `FOR UPDATE SKIP LOCKED`, API/UI trung tâm thông báo và smoke test đa vai trò | Mutation nghiệp vụ tạo outbox trong cùng transaction; worker materialize notification có retry | Giữ worker idempotent và theo dõi backlog/attempts khi triển khai production |
| DRIFT-002 | Resolved | Moderate | Router/store đã có `GET/POST /purchase-requests/{id}/comments`; HTTP/domain tests xác minh contract và validation | Người dùng trao đổi được mà không cần đổi trạng thái | Comment được lưu như process event và audit trong cùng transaction |
| DRIFT-003 | Resolved | Moderate | Router/store đã có `/me/tasks-summary`; Angular có trang `/work-center` | Từng vai trò có hàng đợi công việc đúng scope | Backend tính task type và urgency, frontend chỉ trình bày |
| DRIFT-004 | Resolved | Moderate | Transition SUBMIT/RESUBMIT gán `sla_due_at` theo `reporting.sla_policies`, fallback 72 giờ | Deadline tác nghiệp và KPI dùng cùng policy | Dữ liệu cũ tiếp tục dùng fallback từ `submitted_at` |
| DRIFT-005 | Resolved | Low | Tài liệu tính năng mới tách comment độc lập khỏi ghi chú transition | Tài liệu khớp contract đang chạy | Duy trì test contract khi thay đổi API |
| DRIFT-006 | Resolved | High | Migration `000010`, API/UI `/invoices`, đối soát động order–receipt–invoice và smoke test từ ghi nhận tới `PAID` | Tài chính có hàng đợi công nợ, chặn xác minh khi chưa nhận hàng/lệch tiền và giữ bằng chứng thanh toán | Mọi tiền tiếp tục truyền dưới dạng decimal string; một order hiện có tối đa một invoice |
| DRIFT-007 | Resolved | Moderate | Migration `000011`, API/UI `/admin/policies` và `/policies`; `dx_admin` cập nhật có version/audit, auditor chỉ đọc | SLA và ngưỡng chứng từ thay đổi được không cần sửa SQL | Thay đổi SLA chỉ áp dụng cho lần submit/resubmit tiếp theo; smoke test tự khôi phục giá trị |
| DRIFT-008 | Resolved | Moderate | Middleware `principalRateLimit`, test quota tách theo subject và response `429` có `Retry-After` | Token hợp lệ không thể gọi API vô hạn trên một instance | Mức hiện tại 120 request/phút/principal; production nhiều replica phải chuyển state limiter sang Redis/gateway |

## 2. Eras view

| Era | Dấu mốc | Đặc điểm |
|---|---|---|
| E1 — Initial skeleton | commit `22c7395` ngày 2026-08-01 | Tài liệu kiến trúc đặt tầm nhìn rộng (outbox, worker, AI, analytics), code tập trung procurement MVP |
| E2 — Documentation portal | commit `752d48d` ngày 2026-08-01 | Thêm Docusaurus; một số endpoint trong tài liệu vẫn là thiết kế, chưa phải contract đang chạy |
| E3 — Demo hardening | thay đổi chưa commit trong workspace | Dashboard theo vai trò, báo cáo/ngân sách export, hướng dẫn sử dụng và các cải tiến phục vụ báo cáo |
| E4 — Procurement operations | thay đổi chưa commit trong workspace | Supplier directory, purchase order/giao nhận và audit evidence center nối tiếp sau phê duyệt |
| E5 — Enterprise gap audit | 2026-08-15, thay đổi chưa commit | Đối chiếu procure-to-pay, notification/outbox, policy administration và API abuse protection |

Không có đủ lịch sử commit để quy trách nhiệm hoặc suy ra thời điểm chính xác hơn; registry chỉ ghi bằng chứng có thể kiểm tra.

## 3. Responsibility view

| Trách nhiệm | Nguồn đúng hiện tại | Điểm lệch/nhân đôi |
|---|---|---|
| Authorization và scope phiếu | Go domain/store (`ScopeFor`, `DecideTransition`) | Frontend chỉ ẩn/hiện UI; backend vẫn là nơi quyết định cuối cùng |
| Workflow event và audit | `process_events` + `audit_logs` trong cùng transaction | Không có shared-purpose duplicate; hai bảng phục vụ timeline nghiệp vụ và audit compliance khác nhau |
| Bình luận | `GET/POST /purchase-requests/{id}/comments`, lưu `process_events` + `audit_logs` cùng transaction | Đã thống nhất; không còn implementation song song |
| SLA tác nghiệp | Cột `sla_due_at` + `reporting.sla_policies` + curated reporting view + Policy Center | Submit/resubmit sở hữu lifecycle; admin chỉ đổi policy nguồn, không sửa deadline lịch sử |
| Notification/outbox | `outbox_events` + worker + `user_notifications`/`notification_reads` | Mutation sở hữu enqueue; worker sở hữu materialization; UI chỉ đọc/đánh dấu đã đọc |
| Nhà cung cấp và giao nhận | `suppliers` + `purchase_orders`, được sở hữu bởi procurement store | Tách state giao nhận khỏi status phê duyệt để không tạo hai nghĩa cho `APPROVED` |
| Hóa đơn và thanh toán | `purchase_invoices` + `invoice_events`, procurement store và trang `/invoices` | Match status được tính từ state order/receipt/invoice hiện tại, không lưu bản sao dễ lệch |
| Bằng chứng kiểm toán | Mutation ghi `audit_logs`; reporting chỉ cung cấp query read-only | Hai phía có mục đích khác nhau và thống nhất schema, không phải duplicate mutation path |

## 4. Risk view

| Rủi ro | Kiểm tra | Kết luận |
|---|---|---|
| Event state-existence | Các mutation tạo/cập nhật/transition ghi event và audit trong cùng PostgreSQL transaction; attachment có trạng thái bù trừ riêng | Core workflow không để event trỏ tới state chưa commit. Comment mới phải giữ nguyên nguyên tắc này |
| Money representation | PostgreSQL dùng `numeric(19,4)`; Go nhận/trả decimal bằng `string` và validate bằng `big.Rat`; Angular model cũng dùng `string` | Không chuyển tiền sang `float64`/JavaScript `number`. Work center chỉ đọc và chuyển tiếp chuỗi tiền |
| Scope leakage | Finance chỉ thấy request từ `MANAGER_APPROVED` trở đi; manager theo department; employee theo ownership; auditor toàn cục read-only | Endpoint task/comment mới phải tái sử dụng scope từ store, không tự tin tưởng role ở frontend |
| Documentation drift | Nhiều mục kiến trúc là target state nhưng viết ở thì hiện tại | Tài liệu bàn giao phải phân biệt “đã chạy” và “roadmap” |
| Post-approval state existence | Tạo order khóa request `APPROVED`, kiểm supplier cùng tổ chức/đang active rồi ghi order + process event + audit trong một transaction; receipt khóa order trước khi ghi event | Không có event đặt/nhận hàng nếu state nguồn không tồn tại hoặc transaction rollback |
| Separation of duties | Finance tạo order; requester/manager cùng phòng nhận hàng; Auditor chỉ đọc | Backend kiểm quyền độc lập với việc ẩn nút trên Angular |
| API abuse | JSON/file giới hạn kích thước; OIDC xác thực mọi `/api/v1`; limiter 120 request/phút theo Keycloak subject | `429` có `Retry-After`; health không bị giới hạn. Limiter in-memory chỉ phù hợp một API instance |

## 5. Update rule

- Khi finding đã được sửa, giữ nguyên dòng và chuyển trạng thái thành `Resolved`, kèm file/test xác minh.
- Chỉ thêm finding khi có bằng chứng code, migration, test hoặc commit; không suy đoán ý định tác giả.
- Mọi chức năng liên quan tiền phải tiếp tục dùng decimal string từ database đến UI.
