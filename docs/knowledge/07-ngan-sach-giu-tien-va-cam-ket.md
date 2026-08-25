# Ngân sách, giữ tiền và cam kết chi

## Các khái niệm

- **Tổng hạn mức**: số tiền được phân bổ cho trung tâm chi phí trong kỳ.
- **Đang giữ**: tiền tạm dành cho phiếu đã qua Trưởng bộ phận nhưng chưa duyệt cuối.
- **Đã cam kết**: tiền của phiếu đã được Tài chính phê duyệt.
- **Còn khả dụng**: hạn mức trừ số đang giữ và đã cam kết.

Các số tiền luôn đi kèm tiền tệ. Không cộng VND với USD thành một tổng chung.

## Luồng ngân sách theo phiếu

Khi Trưởng bộ phận phê duyệt, backend khóa dữ liệu liên quan, kiểm tra số còn khả dụng và tạo khoản giữ trong cùng transaction. Nếu thiếu tiền, toàn bộ thao tác thất bại và không để lại khoản giữ dở dang.

Khi Tài chính phê duyệt, khoản giữ chuyển thành cam kết. Nếu Tài chính yêu cầu chỉnh sửa hoặc từ chối, khoản giữ được giải phóng. Phiếu đã hủy khi còn Bản nháp/Yêu cầu chỉnh sửa chưa tạo cam kết mới.

## Điều chỉnh hạn mức

Tài chính có thể điều chỉnh phân bổ ngân sách và phải nhập lý do. Không thể hạ hạn mức xuống thấp hơn tổng đang giữ cộng đã cam kết. Kiểm toán có quyền xem nhưng không điều chỉnh.

## Kiểm tra khi số liệu không khớp

Đối chiếu cùng kỳ tài chính, trung tâm chi phí và tiền tệ. Sau đó mở timeline phiếu để xác định thời điểm giữ, cam kết hoặc giải phóng. Báo cáo và Metabase là lớp đọc; nguồn nghiệp vụ vẫn là transaction trong PostgreSQL.

## Nguồn mã xác minh

`backend/internal/procurement/store.go`, `backend/internal/procurement/model.go`, `frontend/src/app/features/procurement/pages/budget-dashboard`.
