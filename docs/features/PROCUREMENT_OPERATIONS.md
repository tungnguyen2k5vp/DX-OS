# Vận hành mua sắm sau phê duyệt

## 1. Ba nhóm chức năng đã triển khai

### Danh mục nhà cung cấp

Trang `/suppliers` dành cho Finance và Auditor:

- Finance tạo/cập nhật mã, tên, mã số thuế, liên hệ, trạng thái hợp tác và mức rủi ro.
- Mã nhà cung cấp và mã số thuế không được trùng trong cùng tổ chức.
- Nhà cung cấp `INACTIVE` không thể được chọn cho đơn hàng mới.
- Auditor chỉ đọc; mọi lần tạo/cập nhật đều ghi `audit_logs`.

### Đặt hàng và giao nhận

Trang `/operations` dùng cho Employee, Manager, Finance và Auditor:

1. Phiếu phải hoàn tất hai vòng duyệt và ở trạng thái `APPROVED`.
2. Finance chọn nhà cung cấp đang hoạt động, ngày giao dự kiến, mã tham chiếu và phát hành đơn hàng.
3. Hệ thống sinh mã `PO-YYYY-NNNNNN`, chống tạo trùng bằng `Idempotency-Key` và quan hệ một phiếu–một đơn hàng.
4. Requester hoặc Manager cùng phòng ban xác nhận ngày nhận thực tế.
5. Bảng vận hành ưu tiên đơn giao trễ, hiển thị số chờ đặt, đang giao, giao trễ và đã nhận.

Finance không được tự xác nhận nhận hàng. Đây là phân tách nhiệm vụ giữa người đặt và người nghiệm thu.

### Trung tâm kiểm toán

Trang `/audit` dành cho Auditor và DX Admin:

- lọc theo loại tài nguyên, hành động, khoảng ngày;
- xem actor, role, trạng thái trước/sau, thời gian và correlation ID;
- theo dõi riêng thay đổi nhà cung cấp và sự kiện đơn hàng;
- không cung cấp mutation từ màn hình audit.

## 2. API

```text
GET    /api/v1/suppliers
POST   /api/v1/suppliers
PATCH  /api/v1/suppliers/{supplierId}

GET    /api/v1/procurement-operations
POST   /api/v1/procurement-operations/orders
POST   /api/v1/procurement-operations/orders/{requestId}/receipt

GET    /api/v1/audit/events
```

## 3. State và audit

```text
APPROVED request
    -> AWAITING_ORDER
    -> ORDERED
    -> RECEIVED
```

Trạng thái giao nhận nằm ở `purchase_orders`, không làm biến dạng state machine phê duyệt của
`purchase_requests`. `ORDER_PLACED` và `DELIVERY_RECEIVED` được thêm vào `process_events` để timeline
phiếu có bức tranh end-to-end. Thay đổi request/order/supplier được ghi audit trong cùng transaction.

## 4. Tiền và ngày

- Giá trị đơn hàng luôn lấy từ `purchase_requests.total_amount numeric(19,4)` và truyền sang API/UI bằng string.
- Không cho client nhập lại giá trị đơn hàng, tránh lệch với ngân sách đã committed.
- Ngày giao dự kiến dùng `YYYY-MM-DD`; không cho chọn ngày quá khứ.
- Ngày nhận thực tế không được ở tương lai.

## 5. Kịch bản demo

1. Finance tạo một nhà cung cấp rủi ro thấp.
2. Hoàn tất luồng duyệt một phiếu bằng Employee → Manager → Finance.
3. Finance mở **Giao nhận**, chọn phiếu vừa duyệt và phát hành đơn hàng.
4. Employee hoặc Manager mở **Giao nhận**, xác nhận đã nhận.
5. Auditor mở **Kiểm toán**, lọc `purchase_order` để xem hai bằng chứng tạo đơn và nhận hàng.
6. Thử bằng Finance xác nhận nhận hàng và bằng Auditor sửa nhà cung cấp; API phải trả `403`.
