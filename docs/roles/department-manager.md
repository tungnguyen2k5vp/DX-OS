---
title: Trưởng bộ phận
description: Xử lý bước duyệt phòng ban và giữ ngân sách.
sidebar_position: 2
---

# Hướng dẫn cho Trưởng bộ phận

Role Keycloak: **department_manager**.

## Phạm vi quyền

- Tạo và quản lý phiếu của mình như employee.
- Xem phiếu thuộc department đang quản lý.
- Xử lý phiếu ở trạng thái Đã gửi.
- Không được tự duyệt phiếu do chính mình tạo.
- Không thực hiện bước duyệt tài chính.

## Xử lý hộp thư phê duyệt

1. Mở menu **Phê duyệt**.
2. Chọn một phiếu Đã gửi thuộc đúng phòng ban.
3. Đối chiếu lý do, dòng hàng, tổng tiền, báo giá, cost center và budget check.
4. Chọn một trong ba hành động:
   - **Phê duyệt**: chuyển phiếu sang bước tài chính và giữ ngân sách;
   - **Yêu cầu chỉnh sửa**: trả lại chủ phiếu;
   - **Từ chối**: kết thúc quy trình.
5. Nhập comment rõ ràng khi yêu cầu chỉnh sửa hoặc từ chối.

## Kiểm soát trước khi duyệt

- Người yêu cầu không phải chính tài khoản đang duyệt.
- Phiếu thuộc department được cấp.
- Tài liệu bắt buộc đã đủ.
- Cost center đúng và ngân sách khả dụng đủ.
- Version trên màn hình vẫn là version mới nhất.

Nếu ngân sách không đủ, transaction duyệt sẽ không hoàn tất và không tạo reservation dở dang.

Xem chi tiết tại
[Hướng dẫn sử dụng tổng thể](../USER_GUIDE.md#7-hướng-dẫn-cho-department-manager).
