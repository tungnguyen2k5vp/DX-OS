# Tệp đính kèm và Nextcloud

## Loại tài liệu được hỗ trợ

Tệp có thể được phân loại là Báo giá, Đặc tả, Hợp đồng hoặc Tài liệu khác. Định dạng được chấp nhận gồm PDF, DOCX, XLSX, JPG/JPEG và PNG. Mỗi tệp phải lớn hơn 0 byte và không vượt quá 10 MB.

Backend kiểm tra đồng thời tên tệp, phần mở rộng, loại nội dung và chữ ký thực tế của tệp. Đổi đuôi một tệp không đúng định dạng sẽ không vượt qua kiểm tra.

## Quy tắc báo giá từ 20 triệu VND

Cấu hình khởi tạo yêu cầu phiếu VND có tổng tiền từ 20.000.000 trở lên phải có ít nhất một tệp đang hoạt động thuộc loại **Báo giá** trước khi Gửi hoặc Gửi lại. Tệp loại khác không thay thế được Báo giá. Quản trị có thể thay đổi ngưỡng và loại chứng từ tại Trung tâm chính sách.

## Tệp được lưu ở đâu?

Nội dung tệp nằm trong Nextcloud. PostgreSQL lưu thông tin mô tả, tên gốc, loại tệp, kích thước, SHA-256, đường dẫn lưu trữ, người tải lên và trạng thái. Trình duyệt không truy cập đường dẫn Nextcloud trực tiếp; tải lên và tải xuống đều đi qua Go API để kiểm tra quyền.

## Xóa tệp

Chỉ chủ phiếu được xóa khi phiếu còn ở trạng thái có thể chỉnh sửa. Backend đánh dấu quá trình xóa, xóa nội dung khỏi Nextcloud rồi giữ metadata ở trạng thái `DELETED` để còn dấu vết. Tệp đã xóa không được tính là chứng từ hợp lệ.

## Nếu Nextcloud lỗi giữa chừng

Tệp chỉ hiển thị khi đạt trạng thái `ACTIVE`. Nếu lưu Nextcloud hoặc hoàn tất metadata thất bại, backend thực hiện dọn dẹp bù trừ để tránh hiển thị tệp chưa hoàn chỉnh.

## Nguồn mã xác minh

`backend/internal/procurement/attachments.go`, `backend/internal/procurement/model.go`, `backend/internal/platform/documentstore/nextcloud.go`, `backend/migrations/000006_purchase_request_attachments.sql`.

