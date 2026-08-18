# Quy ước thuật ngữ tiếng Việt

Tài liệu này giúp giao diện, thông báo API và tài liệu DX-OS dùng từ nhất quán.

## Thuật ngữ được Việt hóa

| Tiếng Anh | Cách hiển thị trong DX-OS |
|---|---|
| Employee | Nhân viên |
| Manager / Department manager | Trưởng bộ phận |
| Finance | Tài chính / Bộ phận Tài chính |
| Auditor | Kiểm toán |
| Procurement | Mua sắm |
| Cost center | Trung tâm chi phí |
| Allocation | Hạn mức ngân sách hoặc khoản phân bổ, tùy ngữ cảnh |
| Workflow | Luồng xử lý hoặc quy trình |
| Timeline | Lịch sử xử lý |
| Dashboard | Bảng điều khiển hoặc Tổng quan |
| Attachment | Tệp đính kèm |
| Supplier | Nhà cung cấp |
| Purchase order | Đơn hàng |
| Invoice | Hóa đơn |
| Receipt | Biên bản giao nhận |
| Return / Request changes | Yêu cầu chỉnh sửa |
| Reject | Từ chối |
| Lead time | Thời gian xử lý |
| Compliance | Tuân thủ |
| Healthy | Hoạt động tốt |

## Thuật ngữ kỹ thuật được giữ nguyên

Các tên sau được giữ nguyên vì là chuẩn kỹ thuật, tên sản phẩm hoặc giá trị giao tiếp giữa các hệ thống:

- API, HTTP, JSON, CSV, UUID, URL và SQL;
- OIDC, OAuth 2.0, PKCE, RBAC, SLA và SSO;
- Keycloak, PostgreSQL, Angular, Go, Docker Compose, Nextcloud và Metabase;
- endpoint, access token, refresh token, Idempotency-Key và Content-Type khi nói về hợp đồng API;
- mã role như `employee`, `department_manager`, `finance`, `auditor`, `dx_admin` và `ai_operator`;
- mã trạng thái gửi qua API như `DRAFT`, `SUBMITTED`, `APPROVED`, `REJECTED` và `CANCELLED`.

Khi mã kỹ thuật xuất hiện trên giao diện, DX-OS hiển thị thêm nghĩa tiếng Việt, ví dụ `LOW (thấp)` hoặc thay bằng nhãn tiếng Việt nhưng vẫn gửi nguyên mã cho API.
