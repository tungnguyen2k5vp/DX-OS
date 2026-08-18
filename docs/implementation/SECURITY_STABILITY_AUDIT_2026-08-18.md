---
title: Rà soát bảo mật và ổn định 2026-08-18
sidebar_position: 98
---

# Rà soát bảo mật và ổn định DX-OS

Ngày thực hiện: 18/08/2026. Phạm vi gồm Go API/worker/migration, Angular, PostgreSQL,
Keycloak, Nextcloud, Docusaurus và cấu hình Docker Compose.

## Kết quả chính

| Mức độ | Phát hiện | Trạng thái |
|---|---|---|
| Nghiêm trọng | Go 1.26.5, pgx 5.7.6 và x/text 0.24.0 có đường gọi tới CVE đã công bố | Đã nâng Go 1.26.6, pgx 5.9.2, x/text 0.39.0; govulncheck còn 0 lỗ hổng có đường gọi |
| Cao | Auditor có thể đọc dữ liệu ngoài organization ở một số board/report | Đã bắt buộc scope theo organization ở phiếu, task, vận hành, hóa đơn, báo cáo và audit center |
| Cao | Tài khoản gán chồng auditor với role nghiệp vụ có thể ghi dữ liệu | Đã ưu tiên quyền auditor chỉ đọc đối với nghiệp vụ; auditor chỉ được ghi hồ sơ kiểm toán |
| Cao | Hóa đơn nhiều tiền tệ có thể cộng tổng sai ngữ nghĩa | Đã chặn match nếu bất kỳ hóa đơn nào của PO khác tiền tệ phiếu |
| Cao | Idempotency key phát lại với payload khác vẫn có thể nhận thành công | Đã so sánh toàn bộ ý định của PO, hóa đơn và thanh toán; payload khác trả conflict |
| Cao | Upload tin Content-Type do client khai báo | Đã kiểm chữ ký PDF/JPEG/PNG, cấu trúc DOCX/XLSX, phần mở rộng, kích thước, số multipart part và tổng body |
| Cao | Migration chỉ ghi tên file, không phát hiện file cũ bị sửa | Đã lưu SHA-256 cho 13 migration và dừng khi checksum không khớp |
| Cao | Hồ sơ kiểm toán có thể tham chiếu owner thuộc organization khác | Đã xác minh owner đang active và cùng organization khi tạo/cập nhật |
| Trung bình | display_name quản trị bị username Keycloak ghi đè ở mỗi request | Đã giữ display_name do quản trị viên cấu hình |
| Trung bình | Bảng prototype approval_delegations lệch với source migration | Đã xác minh bảng rỗng rồi loại bỏ bằng migration 000013 |
| Trung bình | Parser multipart ghi file tạm và bị scanner cảnh báo giới hạn | Đã thay bằng parser streaming, giới hạn 10 part và body tối đa |

## Biên quyền sau hardening

| Vai trò | Phạm vi dữ liệu | Quyền ghi chính |
|---|---|---|
| employee | Phiếu của mình | Tạo/sửa/gửi phiếu, comment, upload, xác nhận nhận hàng phù hợp |
| department_manager | Phòng ban của mình | Duyệt cấp phòng ban; không tự duyệt phiếu của mình |
| finance | Organization của mình | Ngân sách, nhà cung cấp, PO, hóa đơn và thanh toán |
| auditor | Organization của mình | Chỉ đọc nghiệp vụ; tạo/cập nhật audit case và xuất evidence |
| dx_admin | Organization của mình | Hồ sơ user, phòng ban và policy; không là superuser nghiệp vụ |
| ai_operator | Organization của mình | Tạo/ra quyết định recommendation theo allowlist |

Nếu token có đồng thời role `auditor` và role nghiệp vụ, backend áp quyền hạn chế của auditor cho
mọi thao tác kinh doanh. Quyền được kiểm ở API/store; việc ẩn nút trên Angular không phải lớp bảo vệ.

## Bản đồ trách nhiệm

| Thành phần | Trách nhiệm |
|---|---|
| Keycloak | Danh tính, token RS256, realm role |
| `platform/auth` | Kiểm chữ ký, issuer, audience, expiry và thuật toán token |
| `platform/identity` | Ánh xạ subject sang user/department/organization, trạng thái active |
| `procurement` | State machine, RBAC, tenant scope, tiền tệ, idempotency và audit nghiệp vụ |
| `reporting` | Báo cáo và audit log chỉ trong organization |
| PostgreSQL migration | Constraint, schema history và checksum |
| Nextcloud adapter | Đường dẫn an toàn, timeout, giới hạn download và quản lý response body |
| Angular/Nginx | OIDC client, route guard, CSP và security header; backend vẫn quyết quyền cuối |

## Drift registry

| Khu vực | Dấu hiệu drift | Xử lý hiện tại | Theo dõi |
|---|---|---|---|
| Migration 000012 | Database từng có bảng approval_delegations nhưng source hiện tại không có | Bảng có 0 dòng, đã xóa ở 000013 | Mọi migration mới bắt buộc checksum |
| Nginx image | Dockerfile dùng 1.28.0 trong khi `.env.example` ghim 1.30.4 | Dockerfile nhận `NGINX_IMAGE` từ Compose | Nâng version qua review rõ ràng |
| Reporting identity | Có bản sao logic provision user và ghi đè display_name | Dùng chung `identity.Ensure` | Không nhân bản logic tenant identity |
| Auditor scope | Tài liệu nói theo mandate nhưng schema chưa có mandate | Tạm giới hạn theo organization | Nếu cần kiểm toán chéo tổ chức phải thêm audit mandate riêng |

## Bằng chứng kiểm thử

- `go test ./...`: đạt.
- `go test -race ./...`: đạt.
- `go vet ./...`: đạt.
- `gosec`: 36 file, 11.304 dòng, 0 issue.
- `govulncheck`: 0 lỗ hổng có đường gọi.
- Angular test target và production build: đạt.
- Docusaurus production build và TypeScript typecheck: đạt.
- Compose config: hợp lệ; API, PostgreSQL và Nextcloud healthy.
- Runtime: API/web trả 200; API không token trả 401.
- `Test-ProcurementWorkflow.ps1`: đạt toàn bộ luồng đa vai trò, RBAC âm, ngân sách, PO,
  nhận hàng, invoice, payment, audit, notification và upload.
- Database: 13 migration có checksum; 0 audit log thiếu organization.

Backup trước migration được lưu cục bộ, Git ignore tại
`data/runtime/backups/dxos-pre-security-20260818.dump`.

## Rủi ro còn lại và điều kiện production

1. Docusaurus còn 18 cảnh báo gián tiếp qua `image-size`; npm hiện báo không có bản sửa. Phạm vi là
   build tài liệu với nguồn Markdown tin cậy, không nằm trong Go API runtime. Tiếp tục theo dõi bản
   Docusaurus mới và không build tài liệu không tin cậy.
2. Stack hiện là môi trường local development, bind vào `127.0.0.1` và dùng HTTP. Khi triển khai thật
   phải đặt sau TLS reverse proxy, dùng secret manager, đổi toàn bộ mật khẩu demo và không mở trực
   tiếp PostgreSQL/Keycloak/Nextcloud ra Internet.
3. Rate limit đang lưu trong RAM của một API instance. Nếu chạy nhiều replica phải chuyển sang bộ
   đếm dùng chung như Redis hoặc gateway rate limiting.
4. Checksum của migration cũ được ghi nhận lần đầu ở đợt hardening này nên không chứng minh được
   lịch sử trước ngày 18/08/2026; từ thời điểm này mọi thay đổi file đã áp dụng sẽ bị chặn.
