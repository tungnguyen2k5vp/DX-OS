# Vai trò, menu và phân quyền trong DX-OS

DX-OS có sáu vai trò nghiệp vụ. Menu chỉ hỗ trợ người dùng tìm chức năng; backend vẫn kiểm tra quyền ở mọi API quan trọng.

## Nhân viên — `employee`

Nhân viên tạo và quản lý phiếu của mình, gửi hoặc gửi lại phiếu, hủy phiếu khi còn được phép, trao đổi, quản lý tệp đính kèm và xác nhận hàng được giao cho nhu cầu của mình. Nhân viên không phê duyệt và không điều chỉnh ngân sách.

## Trưởng bộ phận — `department_manager`

Trưởng bộ phận có thể tạo phiếu như Nhân viên, duyệt phiếu của người khác trong cùng phòng ban, yêu cầu chỉnh sửa, từ chối, lập ủy quyền có thời hạn và xác nhận giao nhận trong phạm vi phòng ban. Người duyệt không được tự duyệt phiếu do chính mình tạo.

## Tài chính — `finance`

Tài chính duyệt cấp cuối, quản lý hạn mức ngân sách, hồ sơ nhà cung cấp, báo giá, lựa chọn nhà cung cấp, phát hành đơn hàng, nhập hóa đơn, đối soát và ghi nhận thanh toán. Tài chính không tự xác nhận hàng cho đơn do mình điều phối.

## Kiểm toán — `auditor`

Kiểm toán xem rộng dữ liệu mua sắm, ngân sách, nhà cung cấp, báo giá, hóa đơn, báo cáo, chính sách và lịch sử. Kiểm toán quản lý hồ sơ kiểm toán và xuất gói bằng chứng, nhưng không sửa dữ liệu nghiệp vụ mua sắm hoặc tài chính.

## Quản trị DX-OS — `dx_admin`

Quản trị quản lý hồ sơ người dùng, phòng ban, trạng thái truy cập, quy tắc phê duyệt và chính sách vận hành. Vai trò này không tự động thay thế Nhân viên, Trưởng bộ phận hoặc Tài chính.

## Điều phối AI — `ai_operator`

Điều phối AI tạo đợt quét khuyến nghị và ghi quyết định Chấp nhận, Bác bỏ hoặc Bỏ qua. Khuyến nghị không tự phê duyệt phiếu, không tự điều chỉnh tiền và không thay thế người có thẩm quyền.

## Chức năng chung

Mọi tài khoản đã xác thực đều xem được Tổng quan, Thông báo và Trợ lý AI nội bộ. Quyền thực tế còn phụ thuộc tổ chức, phòng ban, chủ phiếu và trạng thái hiện tại.

## Nguồn mã xác minh

`frontend/src/app/core/navigation/navigation.model.ts`, `frontend/src/app/features/procurement/procurement.guard.ts`, `backend/internal/procurement/model.go`.

