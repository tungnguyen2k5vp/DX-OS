# Tổng quan website DX-OS

DX-OS là website vận hành quy trình mua sắm doanh nghiệp từ lúc tạo nhu cầu đến phê duyệt, lựa chọn nhà cung cấp, đặt hàng, giao nhận, hóa đơn, thanh toán và kiểm toán.

## Địa chỉ chạy local

| Thành phần | Địa chỉ | Công dụng |
|---|---|---|
| Website DX-OS | `http://localhost:4200` | Giao diện nghiệp vụ Angular |
| Go API | `http://localhost:8081` | API và kiểm tra phân quyền |
| API sẵn sàng | `http://localhost:8081/health/ready` | Trả trạng thái `ready` khi API hoạt động |
| Keycloak | `http://localhost:8080` | Đăng nhập và cấp vai trò |
| Metabase | `http://localhost:3000` | Phân tích và dashboard nâng cao |
| Nextcloud | `http://localhost:8082` | Kho lưu nội dung tệp đính kèm |

## Luồng dữ liệu chính

Người dùng mở Angular, đăng nhập qua Keycloak và nhận token. Angular gửi token tới Go API. Go API kiểm tra vai trò và phạm vi dữ liệu trước khi đọc hoặc ghi PostgreSQL. Nội dung tệp đính kèm được Go API lưu vào Nextcloud; PostgreSQL chỉ lưu thông tin tệp, trạng thái, đường dẫn nội bộ và mã kiểm tra SHA-256. Metabase đọc schema báo cáo bằng tài khoản chỉ đọc.

## Các trang chính

- `/dashboard`: Tổng quan.
- `/work-center`: Việc cần xử lý.
- `/purchase-requests`: Phiếu mua sắm.
- `/approvals`: Hàng đợi phê duyệt.
- `/approval-governance`: Ủy quyền và quy tắc phê duyệt.
- `/suppliers`, `/sourcing`, `/operations`: Nhà cung cấp, báo giá, đặt hàng và giao nhận.
- `/budgets`, `/invoices`, `/reports`: Ngân sách, hóa đơn và báo cáo.
- `/audit`, `/policies`, `/admin`: Kiểm toán, chính sách và quản trị.
- `/ai-assistant`, `/ai-center`: Trợ lý AI và khuyến nghị kiểm soát.

## Nguồn mã xác minh

`frontend/src/app/app.routes.ts`, `backend/cmd/api/main.go`, `compose/application.yml`.

