# Vòng đời phiếu mua sắm

## Các trạng thái

| Trạng thái kỹ thuật | Hiển thị dễ hiểu | Ý nghĩa |
|---|---|---|
| `DRAFT` | Bản nháp | Chủ phiếu đang hoàn thiện nội dung |
| `SUBMITTED` | Đã gửi | Chờ Trưởng bộ phận xử lý |
| `MANAGER_APPROVED` | Trưởng bộ phận đã duyệt | Chờ Tài chính duyệt cuối |
| `CHANGES_REQUESTED` | Yêu cầu chỉnh sửa | Trả về chủ phiếu để bổ sung |
| `APPROVED` | Đã phê duyệt | Hoàn tất phê duyệt, có thể báo giá và đặt hàng |
| `REJECTED` | Đã từ chối | Quy trình kết thúc do người duyệt từ chối |
| `CANCELLED` | Đã hủy | Chủ phiếu dừng nhu cầu ở trạng thái cho phép |

## Luồng mặc định hai cấp

`Bản nháp → Đã gửi → Trưởng bộ phận đã duyệt → Đã phê duyệt`.

Ở bước Trưởng bộ phận hoặc Tài chính, người duyệt có thể yêu cầu chỉnh sửa. Phiếu chuyển sang `CHANGES_REQUESTED`; chủ phiếu sửa rồi dùng Gửi lại để trở về `SUBMITTED`. Người duyệt cũng có thể từ chối và kết thúc quy trình.

## Khi nào được hủy phiếu?

Chỉ chủ phiếu được hủy khi phiếu đang là `DRAFT` hoặc `CHANGES_REQUESTED`. Hủy là chuyển trạng thái sang `CANCELLED`, không xóa bản ghi. Timeline, sự kiện quy trình và nhật ký kiểm toán vẫn còn để truy vết.

## Vì sao không thấy nút thao tác?

Nút Gửi, Gửi lại, Hủy, Phê duyệt, Yêu cầu chỉnh sửa hoặc Từ chối chỉ xuất hiện khi đồng thời đúng vai trò, đúng chủ sở hữu/phòng ban và đúng trạng thái. Ví dụ: phiếu đã gửi thì Nhân viên không còn thấy nút Hủy; Tài chính không thấy nút duyệt khi phiếu chưa qua Trưởng bộ phận.

## Nguồn mã xác minh

`backend/internal/procurement/model.go` — `DecideTransition`; `frontend/src/app/features/procurement/pages/purchase-request-detail/purchase-request-detail.ts`.

