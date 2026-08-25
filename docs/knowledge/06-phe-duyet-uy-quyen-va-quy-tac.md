# Phê duyệt, ủy quyền và quy tắc theo giá trị

## Phê duyệt phòng ban

Trưởng bộ phận xử lý phiếu `SUBMITTED` thuộc đúng phòng ban. Người duyệt không được là người yêu cầu. Có ba quyết định: Phê duyệt, Yêu cầu chỉnh sửa hoặc Từ chối. Khi phê duyệt thành công, hệ thống giữ ngân sách và chuyển phiếu sang bước tiếp theo theo quy tắc áp dụng.

## Phê duyệt tài chính

Tài chính xử lý phiếu đã qua cấp Trưởng bộ phận khi quy tắc yêu cầu bước tài chính. Phê duyệt cuối chuyển ngân sách từ đang giữ sang đã cam kết và đưa phiếu sang `APPROVED`. Yêu cầu chỉnh sửa hoặc từ chối giải phóng khoản đang giữ.

## Quy tắc phê duyệt theo giá trị

Quy tắc có tiền tệ, giá trị tối thiểu, giá trị tối đa tùy chọn, phạm vi phòng ban, các cấp cần duyệt, độ ưu tiên và trạng thái hoạt động. Quy tắc phòng ban được ưu tiên hơn quy tắc chung; trong cùng phạm vi, số ưu tiên nhỏ hơn được xét trước. Nếu không khớp quy tắc nào, hệ thống dùng quy trình mặc định gồm Trưởng bộ phận và Tài chính.

Chỉ `dx_admin` quản lý quy tắc. Tài chính và Kiểm toán xem được. Mỗi lần tạo hoặc cập nhật đều ghi nhật ký kiểm toán và dùng phiên bản để tránh ghi đè.

## Ủy quyền phê duyệt

Trưởng bộ phận có thể chọn một người dùng đang hoạt động, thời gian bắt đầu, thời gian kết thúc và lý do. Ủy quyền chỉ có hiệu lực trong khoảng ngày đã chọn và đúng phòng ban. Không thể tự ủy quyền cho chính mình. Trưởng bộ phận hoặc Quản trị có thể dừng/kích hoạt lại theo quyền tương ứng.

## Nguồn mã xác minh

`backend/internal/procurement/approval_governance.go`, `backend/internal/procurement/store.go`, `frontend/src/app/features/procurement/pages/approval-governance`.

