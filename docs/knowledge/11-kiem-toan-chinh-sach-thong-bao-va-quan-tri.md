# Kiểm toán, chính sách, thông báo và quản trị

## Trung tâm kiểm toán

Auditor và Quản trị xem sự kiện kiểm toán theo loại tài nguyên, hành động và khoảng thời gian. Bảng thể hiện người thực hiện, vai trò, trạng thái trước/sau, thời điểm và mã liên kết yêu cầu. Auditor có thể tạo hồ sơ kiểm toán, giao người phụ trách, cập nhật mức độ/trạng thái và xuất gói bằng chứng JSON theo mã định danh phiếu.

Gói bằng chứng tổng hợp phiếu, dòng hàng, timeline, tệp, ngân sách, đơn hàng, hóa đơn và sự kiện liên quan. Đây là bản xuất phục vụ đối chiếu; dữ liệu nguồn vẫn nằm trong các bảng nghiệp vụ.

## Trung tâm chính sách

Auditor xem chính sách. `dx_admin` cập nhật thời hạn xử lý và quy tắc chứng từ. Thời hạn mới áp dụng khi phiếu được gửi hoặc gửi lại, không viết lại hạn lịch sử. Thay đổi dùng phiên bản dự kiến và được ghi audit.

## Thông báo

Các thay đổi nghiệp vụ tạo sự kiện hàng đợi trong cùng transaction. Worker chuyển sự kiện thành thông báo cho người dùng, vai trò hoặc phòng ban phù hợp. Người dùng có thể xem chưa đọc, đánh dấu từng thông báo hoặc đánh dấu tất cả. Số trên biểu tượng chuông là số chưa đọc.

## Quản trị

`dx_admin` xem và cập nhật hồ sơ người dùng nghiệp vụ, phòng ban và trạng thái truy cập DX-OS. Vai trò nghiệp vụ vẫn được quản lý tập trung qua Keycloak; chỉnh hồ sơ trong DX-OS không tự cấp quyền Keycloak mới.

## Nguồn mã xác minh

`backend/internal/procurement/audit_cases.go`, `backend/internal/procurement/policies.go`, `backend/internal/procurement/admin.go`, `backend/internal/notifications`.
