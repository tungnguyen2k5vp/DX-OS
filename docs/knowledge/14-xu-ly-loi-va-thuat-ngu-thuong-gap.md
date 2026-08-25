# Xử lý lỗi và thuật ngữ thường gặp

## Lỗi truy cập

- `401 Unauthorized`: chưa đăng nhập hoặc token hết hạn; đăng xuất rồi đăng nhập lại.
- `403 Forbidden`: tài khoản đã đăng nhập nhưng thiếu vai trò hoặc nằm ngoài phạm vi phòng ban/tổ chức.
- `404 Not Found`: dữ liệu không tồn tại hoặc backend cố ý ẩn dữ liệu ngoài phạm vi.
- `409 Conflict`: dữ liệu đã thay đổi, khóa chống gửi trùng bị xung đột hoặc thao tác không còn hợp lệ; tải lại trang.
- `422 Unprocessable Entity`: dữ liệu nhập chưa đạt quy tắc; đọc thông báo ngay dưới trường.
- `429 Too Many Requests`: gửi quá nhiều yêu cầu; chờ theo `Retry-After` rồi thử lại.

## Không thấy nút Gửi hoặc Hủy

Kiểm tra tài khoản có phải chủ phiếu không. Gửi và Hủy chỉ dùng cho Bản nháp; Gửi lại và Hủy chỉ dùng khi Yêu cầu chỉnh sửa. Nếu phiếu từ 20 triệu VND mà chưa có tệp Báo giá đang hoạt động, hệ thống không cho gửi.

## Thuật ngữ

- **SLA / thời hạn xử lý**: mốc thời gian mục tiêu để một bước công việc được xử lý. Quá SLA nghĩa là đã quá hạn mục tiêu.
- **Correlation ID / mã liên kết yêu cầu**: mã kỹ thuật nối log, API và sự kiện của cùng một yêu cầu; dùng khi điều tra lỗi.
- **Idempotency key / khóa chống gửi trùng**: mã giúp bấm lại cùng thao tác không tạo thêm bản ghi trùng.
- **Expected version / phiên bản dự kiến**: số phiên bản mà giao diện đang sửa; khác dữ liệu mới nhất thì backend chặn ghi đè.
- **Reservation / khoản đang giữ**: ngân sách tạm dành cho phiếu sau duyệt phòng ban.
- **Commitment / khoản đã cam kết**: ngân sách đã được duyệt cuối.
- **Checksum SHA-256 / mã kiểm tra tệp**: dấu vân tay dùng phát hiện tệp bị thay đổi, không phải mã hóa nội dung.

## Kiểm tra dịch vụ local

```powershell
docker compose ps
Invoke-RestMethod http://localhost:8081/health/ready
```

Website ở cổng 4200, Keycloak 8080, API 8081, Nextcloud 8082 và Metabase 3000.

## Nguồn mã xác minh

`backend/internal/platform/httpapi`, `backend/internal/procurement/model.go`, `backend/internal/aiassistant/service.go`.

