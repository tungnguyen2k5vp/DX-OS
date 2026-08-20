# Hướng dẫn các chức năng vận hành mở rộng

Các chức năng dưới đây dùng nguyên sáu vai trò hiện có. Hệ thống không tạo thêm role; quyền vẫn được kiểm tra ở Go API và giới hạn theo tổ chức/phòng ban.

## 1. Nhân viên — mua sắm có hướng dẫn

1. Mở **Phiếu mua sắm → Tạo phiếu mới**.
2. Chọn hàng thường dùng trong **Chọn nhanh từ danh mục** hoặc tự nhập dòng hàng.
3. Nhập tiêu đề, lý do, trung tâm chi phí và kiểm tra tổng tiền.
4. Bấm **Kiểm tra phiếu trùng**. Nếu có gợi ý, mở phiếu tương tự ở tab mới và chỉ xác nhận tiếp tục khi đây thực sự là nhu cầu khác.
5. Lưu bản nháp, đính kèm báo giá khi hệ thống yêu cầu rồi gửi phê duyệt.

Kiểm tra trùng chỉ cảnh báo; quyết định cuối cùng vẫn thuộc người dùng. API chỉ so sánh các phiếu gần đây trong cùng phòng ban.

## 2. Trưởng bộ phận — duyệt hàng loạt và ủy quyền

- Tại **Phê duyệt**, chọn nhiều phiếu đã kiểm tra rồi bấm **Phê duyệt đã chọn**. Mỗi phiếu vẫn dùng version và idempotency key riêng; phiếu bị người khác sửa sẽ không làm các phiếu còn lại bị duyệt nhầm.
- Tại **Ủy quyền và quy tắc**, chọn một trưởng bộ phận khác, thời gian và lý do. Người nhận chỉ xử lý phiếu của phòng ban được ủy quyền trong đúng khoảng ngày.
- Có thể dừng ủy quyền ngay; mọi thay đổi đều ghi audit log.

## 3. Tài chính — so sánh báo giá

1. Sau khi phiếu được duyệt, mở **So sánh báo giá**.
2. Mở phiếu và nhập ít nhất hai báo giá: nhà cung cấp, số báo giá, tổng giá, ngày giao, bảo hành và điều khoản thanh toán.
3. Hệ thống tính điểm giá, tiến độ, chất lượng và tuân thủ, đồng thời đánh dấu báo giá được đề xuất.
4. Kiểm tra hồ sơ rồi bấm **Chọn báo giá**. Thao tác này chọn một báo giá và loại các phương án còn lại trong cùng giao dịch.
5. Sang **Đặt hàng và giao nhận** để phát hành đơn hàng cho đúng nhà cung cấp đã chọn.

Phiếu từ 50.000.000 VND phải hoàn tất lựa chọn báo giá trước khi phát hành đơn hàng. Phiếu đã đặt hàng không còn xuất hiện trong hàng chờ so sánh.

## 4. Kiểm toán — giám sát liên tục

Kiểm toán viên xem được quy tắc phê duyệt, lịch sử ủy quyền và bảng so sánh báo giá ở chế độ chỉ đọc. Trung tâm khuyến nghị hiện phát hiện thêm:

- phiếu có nguy cơ trùng;
- dấu hiệu chia nhỏ nhu cầu mua sắm;
- đơn giá khác thường so với lịch sử;
- hóa đơn quá hạn;
- thay đổi hồ sơ nhà cung cấp;
- xung đột phân quyền.

Mục **Bằng chứng có thể giải thích** hiển thị từng dữ kiện bằng nhãn tiếng Việt, không còn bắt người dùng đọc JSON thô.

## 5. Quản trị DX-OS — quy tắc và phân tách nhiệm vụ

- Tại **Ủy quyền và quy tắc**, tạo khoảng giá trị và chọn vòng Trưởng bộ phận/Tài chính. Quy tắc phòng ban được ưu tiên hơn quy tắc chung; số ưu tiên nhỏ được xét trước.
- Tại **Quản trị**, xem vai trò gần nhất ghi nhận từ access token và cảnh báo các tổ hợp xung đột như Tài chính + Kiểm toán.
- Cảnh báo không tự sửa role trong Keycloak. Quản trị viên phải xác minh rồi điều chỉnh ở Keycloak để tránh khóa nhầm người dùng.

## 6. Luồng kiểm thử khuyến nghị

`Nhân viên tạo/kiểm tra trùng → Trưởng bộ phận duyệt hoặc ủy quyền → Tài chính duyệt → nhập hai báo giá → chọn nhà cung cấp → phát hành đơn hàng → Nhân viên nhận hàng → Tài chính đối soát hóa đơn → Kiểm toán kiểm tra bằng chứng`.

Sau mỗi bước, kiểm tra **Thông báo**, **Lịch sử xử lý**, **Ngân sách** và **Kiểm toán** để xác nhận dữ liệu liên kết đúng.
