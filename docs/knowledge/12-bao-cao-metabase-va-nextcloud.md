# Báo cáo DX-OS, Metabase và Nextcloud

## Báo cáo trong DX-OS

Trang `/reports` dành cho Tài chính, Kiểm toán và Quản trị. Bộ lọc gồm khoảng ngày và các chiều nghiệp vụ được API hỗ trợ như phòng ban, trung tâm chi phí hoặc tiền tệ. Số liệu được lấy từ schema báo cáo đã tuyển chọn, không cho giao diện ghi ngược vào dữ liệu nghiệp vụ.

Các chỉ số có thể gồm số lượng phiếu, tổng giá trị theo tiền tệ, thời gian xử lý, tình trạng đúng hạn, yêu cầu chỉnh sửa, tuân thủ tệp và mức sử dụng ngân sách. Không cộng các loại tiền tệ khác nhau thành một con số duy nhất.

## Metabase

Metabase mở tại `http://localhost:3000`. Đây là công cụ dashboard và phân tích nâng cao, có tài khoản riêng, không dùng tài khoản Keycloak của DX-OS. Metabase kết nối bằng user PostgreSQL chỉ đọc và chỉ được nhìn schema `reporting`.

Nếu Metabase không mở, kiểm tra container `metabase` và `http://localhost:3000/api/health`. Nếu mở được nhưng chưa có dashboard, chạy script khởi tạo Metabase của dự án.

## Nextcloud

Nextcloud mở tại `http://localhost:8082`. Đây là kho lưu nội dung tệp. Người dùng nghiệp vụ không cần mở Nextcloud để tải tài liệu của phiếu; thao tác chuẩn là tải lên/tải xuống ngay trong trang chi tiết phiếu để Go API kiểm tra quyền và mã kiểm tra tệp.

Metabase và Nextcloud là hai hệ độc lập: Metabase đọc báo cáo, Nextcloud lưu tệp; chúng không thay thế PostgreSQL nghiệp vụ.

## Nguồn mã xác minh

`backend/internal/reporting`, `scripts/Initialize-Metabase.ps1`, `compose/application.yml`, `backend/internal/platform/documentstore/nextcloud.go`.

