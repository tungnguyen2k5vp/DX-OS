# Hóa đơn, đối soát và thanh toán

## Điều kiện ghi nhận hóa đơn

Tài chính ghi hóa đơn cho đơn hàng đã phát hành. Hệ thống dùng đơn hàng, biên nhận và hóa đơn để đối soát. Một đơn hàng có thể có nhiều hóa đơn và một hóa đơn đã xác minh có thể có nhiều lần thanh toán.

## Trạng thái hóa đơn

| Trạng thái | Ý nghĩa |
|---|---|
| `RECORDED` | Đã nhập, đang chờ kiểm tra |
| `DISPUTED` | Có sai lệch cần xử lý với nhà cung cấp |
| `VERIFIED` | Đã đối soát và đủ điều kiện thanh toán |
| `PAID` | Đã thanh toán đủ |

## Kết quả đối soát

- `WAITING_RECEIPT`: Chưa đủ dữ liệu nhận hàng.
- `CURRENCY_MISMATCH`: Tiền tệ hóa đơn khác đơn hàng.
- `AMOUNT_MISMATCH`: Tổng tiền không phù hợp.
- `PARTIAL_MATCH`: Mới khớp một phần.
- `MATCHED`: Đủ điều kiện xác minh theo dữ liệu hiện có.

Chỉ hóa đơn khớp mới được xác minh. Hóa đơn tranh chấp có thể được cập nhật và mở lại về trạng thái đã ghi nhận để đối soát lại.

## Thanh toán

Tài chính nhập số tiền, ngày trả, mã tham chiếu và ghi chú. Backend chặn số tiền không hợp lệ hoặc vượt số còn phải trả. Mỗi lần thanh toán được lưu riêng. Khi tổng đã trả bằng số tiền hóa đơn, trạng thái tự chuyển sang `PAID`. Hóa đơn đã xác minh nhưng quá ngày đến hạn và còn dư sẽ xuất hiện trong cảnh báo thanh toán quá hạn.

## Nguồn mã xác minh

`backend/internal/procurement/invoices.go`, `backend/internal/procurement/payments.go`, `frontend/src/app/features/procurement/pages/invoice-board`.

