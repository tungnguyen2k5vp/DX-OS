# Bước 8 — Quản lý tài liệu đính kèm

## Mục tiêu

Phiếu mua sắm có thể gắn báo giá và tài liệu nghiệp vụ. Go API chịu trách nhiệm xác thực, phân quyền,
kiểm tra quy tắc và checksum; Nextcloud chỉ là kho WebDAV nội bộ. Trình duyệt không nhận tài khoản
hoặc URL WebDAV.

## Kiến trúc và vòng đời

1. Angular gửi `multipart/form-data` tới Go API bằng access token Keycloak.
2. API kiểm tra chủ phiếu, trạng thái có thể sửa, loại tệp và giới hạn 10 MB.
3. API tạo metadata `UPLOADING` trong PostgreSQL.
4. API `PUT` nội dung vào Nextcloud bằng service account.
5. Khi WebDAV thành công, metadata chuyển sang `ACTIVE`, lưu ETag và audit log.
6. Khi tải xuống, API kiểm tra data scope, đọc tệp và xác minh SHA-256.
7. Khi xóa, metadata chuyển `DELETING`, API xóa WebDAV rồi đánh dấu `DELETED`.

Không dùng direct link tới Nextcloud vì direct link sẽ bỏ qua data scope của Procurement.

## Quy tắc nghiệp vụ

- Loại tài liệu: `QUOTATION`, `SPECIFICATION`, `CONTRACT`, `OTHER`.
- Định dạng: PDF, DOCX, XLSX, JPEG, PNG; dung lượng từ 1 byte đến 10 MB.
- Chỉ chủ phiếu được upload/xóa khi trạng thái là `DRAFT` hoặc `CHANGES_REQUESTED`.
- Người có quyền xem phiếu cũng có quyền liệt kê và tải tài liệu của phiếu.
- Phiếu VND của tổ chức `DX-OS` từ `20.000.000` trở lên phải có ít nhất một `QUOTATION` ở trạng
  thái `ACTIVE` trước `SUBMIT` hoặc `RESUBMIT`.
- Tên tệp chỉ là metadata; đường dẫn storage do backend sinh và không nhận từ client.

## Khởi chạy

```powershell
docker compose --profile foundation --profile application up -d --build
docker compose --profile foundation --profile application ps
```

- Angular: `http://localhost:4200`
- Go API: `http://localhost:8081`
- Nextcloud kiểm tra quản trị local: `http://localhost:8082`

Người dùng DX-OS không đăng nhập Nextcloud. `NEXTCLOUD_ADMIN_USER` và
`NEXTCLOUD_ADMIN_PASSWORD` trong `.env` là service account của Go API và không được đưa xuống
Angular.

## API

- `GET /api/v1/purchase-requests/{requestId}/attachments`
- `POST /api/v1/purchase-requests/{requestId}/attachments`
- `GET /api/v1/purchase-requests/{requestId}/attachments/{attachmentId}/content`
- `DELETE /api/v1/purchase-requests/{requestId}/attachments/{attachmentId}`

Multipart upload gồm `documentType=QUOTATION` và `file=@bao-gia.pdf`.

Problem codes chính:

- `quotation-required` — thiếu báo giá khi gửi duyệt;
- `attachment-not-found` — tài liệu không tồn tại hoặc không thuộc phiếu;
- `document-store-unavailable` — Nextcloud/WebDAV tạm thời không sẵn sàng;
- `invalid-attachment` — sai loại, tên, MIME hoặc dung lượng.

## Dữ liệu và vận hành

PostgreSQL là nguồn sự thật cho quan hệ phiếu–tài liệu, policy, trạng thái và audit. Nextcloud là nguồn
nhị phân. Backup hợp lệ phải gồm volume `nextcloud_data`, database `nextcloud` và database `dxos`.

Nếu upload WebDAV thất bại, backend xóa metadata `UPLOADING`. Nếu xóa WebDAV thất bại, backend đưa
metadata từ `DELETING` về `ACTIVE`. Kiểm tra bản ghi kẹt bằng:

```sql
SELECT id, purchase_request_id, status, updated_at
FROM purchase_request_attachments
WHERE status IN ('UPLOADING', 'DELETING')
  AND updated_at < now() - interval '15 minutes';
```

Không xóa thủ công tệp trong Nextcloud trước khi đối chiếu metadata và audit log.

## Kiểm thử chấp nhận

1. Tạo phiếu trên 20 triệu VND; gửi khi chưa có báo giá phải nhận `422 quotation-required`.
2. Upload PDF báo giá; danh sách trả `requirementMet=true`.
3. Tải xuống; SHA-256 nội dung khớp metadata.
4. User ngoài data scope không được liệt kê hoặc tải tài liệu.
5. Xóa khi DRAFT thành công; xóa sau SUBMIT bị từ chối.
6. Tắt Nextcloud; upload/download trả `503` và không để metadata `ACTIVE` sai lệch.

## Giới hạn hiện tại

- Chưa có antivirus/Content Disarm and Reconstruction; không mở tệp trực tiếp trong trình duyệt.
- Chưa có job tự động dọn bản ghi kẹt hoặc orphan.
- Chưa cấu hình Nextcloud OIDC vì người dùng không truy cập Nextcloud trực tiếp trong luồng MVP.
