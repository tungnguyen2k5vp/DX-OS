# Đơn hàng, giao nhận và nghiệm thu

## Phát hành đơn hàng

Phiếu phải ở trạng thái `APPROVED`. Tài chính chọn nhà cung cấp hợp lệ, mã tham chiếu ngoài, ngày giao dự kiến và ghi chú. Hệ thống lấy số tiền, tiền tệ và dòng hàng từ phiếu đã duyệt, không yêu cầu nhập lại. Mỗi phiếu chỉ có một đơn hàng đang hoạt động; khóa chống gửi trùng ngăn việc bấm hai lần tạo hai đơn.

## Trạng thái đơn hàng

- `ORDERED`: Đã phát hành, chờ giao.
- `PARTIALLY_RECEIVED`: Đã nhận một phần.
- `RECEIPT_EXCEPTION`: Có ngoại lệ như hư hỏng, sai hàng hoặc từ chối.
- `RECEIVED`: Đã nhận đủ.
- `CANCELLED`: Đơn đã hủy theo điều kiện cho phép.

## Ai được xác nhận nhận hàng?

Người yêu cầu hoặc Trưởng bộ phận cùng phòng ban xác nhận giao nhận. Tài chính không tự nghiệm thu đơn do mình điều phối. Đây là phân tách trách nhiệm giữa người mua và người nhận.

## Nghiệm thu theo dòng

Mỗi lần nhận ghi ngày nhận, ghi chú và số lượng theo từng dòng. Tổng số lượng đã nhận không được vượt số lượng đặt. Hệ thống hỗ trợ nhiều lần nhận cho một đơn và giữ lịch sử từng biên nhận. Ngoại lệ không bị xóa mà được ghi thành sự kiện và nhật ký kiểm toán.

## Sửa hoặc hủy đơn

Tài chính chỉ sửa/hủy khi trạng thái và dữ liệu liên quan cho phép. Khi đã có biên nhận hoặc hóa đơn, hệ thống hạn chế thay đổi để bảo toàn bằng chứng.

## Nguồn mã xác minh

`backend/internal/procurement/store.go`, `backend/internal/procurement/operations_extensions.go`, `frontend/src/app/features/procurement/pages/operations-board`.
