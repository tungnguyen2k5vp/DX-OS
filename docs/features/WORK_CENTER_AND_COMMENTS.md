# Work Center và trao đổi trên phiếu

## Phạm vi đã triển khai

### Việc của tôi

Mở **Việc của tôi** trên menu hoặc truy cập `http://localhost:4200/work-center`.

- Employee/manager tạo phiếu thấy draft hoặc phiếu bị yêu cầu chỉnh sửa của chính mình.
- Department manager thấy phiếu `SUBMITTED` của người khác trong cùng phòng ban.
- Finance thấy phiếu `MANAGER_APPROVED` trong cùng tổ chức.
- Auditor thấy các phiếu đang xử lý để theo dõi SLA, nhưng không có quyền thay đổi.
- DX admin và AI operator chưa có tác vụ procurement nên hàng đợi có thể trống.

SLA mặc định là 72 giờ từ lần `SUBMIT`/`RESUBMIT` gần nhất. Hệ thống xếp việc quá hạn lên
đầu và đánh dấu việc còn dưới 24 giờ là sắp đến hạn. Endpoint nguồn là:

```http
GET /api/v1/me/tasks-summary
```

Response gồm `items`, `total`, `overdueCount` và `dueSoonCount`. Giá trị tiền tiếp tục được trả bằng
decimal string; frontend không chuyển sang số thực JavaScript.

### Trao đổi độc lập

Trong trang chi tiết phiếu, khu vực **Trao đổi** cho phép hỏi đáp mà không thực hiện transition:

```http
GET /api/v1/purchase-requests/{id}/comments
POST /api/v1/purchase-requests/{id}/comments
Content-Type: application/json

{"body":"Vui lòng xác nhận thời gian giao hàng dự kiến."}
```

Nội dung từ 1 đến 2.000 ký tự. Employee chỉ thao tác trên phiếu của mình, manager theo phòng ban,
finance theo phạm vi tài chính. Auditor chỉ đọc. Mỗi comment được ghi vào `process_events` và
`audit_logs` trong cùng transaction, nên vừa xuất hiện trong luồng trao đổi vừa để lại bằng chứng audit.

## Luồng test nhanh

1. Employee tạo và gửi phiếu, sau đó mở **Việc của tôi** để xác nhận draft đã rời hàng đợi.
2. Manager mở **Việc của tôi**, chọn phiếu vừa gửi, thêm một trao đổi rồi duyệt.
3. Finance mở **Việc của tôi**, đọc trao đổi, thêm phản hồi và duyệt cuối.
4. Auditor mở cùng phiếu để xác nhận đọc được trao đổi nhưng không có nút gửi.
5. Kiểm tra timeline có event `COMMENT_ADDED` và các transition tương ứng.

## Phần chưa triển khai

Worker notification/outbox, email và AI Agent vẫn là roadmap. Work Center hiện đọc dữ liệu trực tiếp
từ PostgreSQL qua Go API; không tuyên bố có thông báo nền hoặc retry queue.
