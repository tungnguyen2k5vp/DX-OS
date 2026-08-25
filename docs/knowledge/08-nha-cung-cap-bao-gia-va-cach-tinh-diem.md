# Nhà cung cấp, báo giá và cách tính điểm

## Hồ sơ nhà cung cấp

Tài chính tạo và cập nhật mã, tên, mã số thuế, liên hệ, địa chỉ, ngân hàng, hợp đồng, trạng thái tuân thủ, mức rủi ro, điểm hiệu suất và ghi chú. Kiểm toán chỉ xem. Nhà cung cấp phải đang hoạt động và không bị chặn tuân thủ mới được dùng cho báo giá mới.

## So sánh báo giá

Trang **So sánh báo giá** hiển thị các phiếu đã phê duyệt nhưng chưa có đơn hàng đang hoạt động. Tài chính nhập nhà cung cấp, số báo giá, tổng giá, tiền tệ, ngày giao, thời gian bảo hành, điều khoản thanh toán và ghi chú. Auditor và Quản trị được xem nhưng chỉ Tài chính quản lý báo giá.

## Cách tính điểm thành phần

- **Điểm giá** = giá thấp nhất trong cùng lần so sánh chia cho giá báo giá, nhân 100. Giá thấp nhất đạt 100 điểm.
- **Điểm tiến độ** = 100 trừ 5 điểm cho mỗi ngày giao chậm hơn ngày sớm nhất; tối thiểu 0.
- **Điểm chất lượng** = điểm hiệu suất trong hồ sơ nhà cung cấp; nếu chưa có thì dùng 60.
- **Điểm tuân thủ** bắt đầu từ 100 nếu đã xác minh, 65 nếu đang chờ, 30 nếu hết hạn, hoặc 0 cho trạng thái khác; sau đó trừ 35 điểm nếu rủi ro cao hoặc 15 điểm nếu rủi ro trung bình, tối thiểu 0.

**Điểm tổng hợp** = Giá 40% + Tiến độ 25% + Chất lượng 20% + Tuân thủ 15%.

Điểm cao nhất chỉ là đề xuất. Tài chính vẫn phải chọn báo giá và ghi lý do từ 10 đến 2.000 ký tự. Khi chọn, báo giá đó thành `SELECTED`, các báo giá còn lại thành `REJECTED` và lần so sánh thành `AWARDED`.

## Tạo đơn từ báo giá

Sau khi chọn, nút **Tạo đơn hàng** chuyển sang trang Đặt hàng và tự điền phiếu, nhà cung cấp, mã tham chiếu, ngày giao và ghi chú. Tài chính chỉ cần đối chiếu rồi phát hành.

## Nguồn mã xác minh

`backend/internal/procurement/sourcing.go`, `frontend/src/app/features/procurement/pages/sourcing-board`, `frontend/src/app/features/procurement/pages/operations-board`.

