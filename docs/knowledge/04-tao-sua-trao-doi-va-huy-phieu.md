# Tạo, sửa, trao đổi và hủy phiếu

## Tạo phiếu

Nhân viên hoặc Trưởng bộ phận mở **Phiếu mua sắm → Tạo phiếu mới**. Cần nhập tiêu đề, lý do mua sắm, trung tâm chi phí, tiền tệ và ít nhất một dòng hàng. Mỗi dòng có mô tả, số lượng, đơn vị và đơn giá. Tổng tiền được backend tính lại từ các dòng, không tin trực tiếp số tổng do trình duyệt gửi lên.

Phiếu mới được lưu ở trạng thái Bản nháp. Sau khi có mã phiếu, người tạo mở trang chi tiết để kiểm tra ngân sách, thêm tài liệu và gửi duyệt.

## Sửa phiếu

Chủ phiếu chỉ sửa nội dung khi trạng thái là Bản nháp hoặc Yêu cầu chỉnh sửa. Sau khi lưu, phiên bản phiếu tăng lên. Nếu dữ liệu đã bị người khác thay đổi, hệ thống yêu cầu tải lại thay vì ghi đè phiên bản mới.

## Trao đổi

Khối **Trao đổi** dùng để bổ sung thông tin mà không đổi trạng thái. Nội dung trao đổi được gắn với phiếu, người gửi và thời gian. Yêu cầu chỉnh sửa là hành động quy trình khác với trao đổi: nó thật sự chuyển phiếu về chủ sở hữu.

## Hủy phiếu

Chủ phiếu mở trang chi tiết, nhập lý do theo yêu cầu giao diện và chọn **Hủy phiếu**. Backend kiểm tra lại chủ sở hữu và trạng thái. Thành công thì trạng thái chuyển sang Đã hủy; phiếu không biến mất khỏi lịch sử.

## Dữ liệu trùng

Khi hệ thống cảnh báo phiếu có khả năng trùng, người dùng cần đối chiếu phiếu được gợi ý. Nếu nhu cầu vẫn độc lập, xác nhận đã kiểm tra rồi mới tiếp tục. Cảnh báo không tự hủy hoặc tự gộp phiếu.

## Nguồn mã xác minh

`frontend/src/app/features/procurement/pages/purchase-request-create`, `backend/internal/procurement/store.go`, `backend/internal/procurement/model.go`.

