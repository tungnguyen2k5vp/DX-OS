---
title: Nhân viên
description: Tạo, sửa, đính kèm tài liệu và gửi phiếu mua sắm.
sidebar_position: 1
---

# Hướng dẫn cho Nhân viên

Role Keycloak: **employee**.

## Phạm vi quyền

- Tạo phiếu mua sắm.
- Xem phiếu do chính mình tạo.
- Sửa phiếu ở trạng thái Bản nháp hoặc Yêu cầu chỉnh sửa.
- Tải lên/xóa tài liệu khi phiếu còn cho phép sửa.
- Gửi, gửi lại hoặc hủy phiếu của mình.
- Không được duyệt phiếu.

## Quy trình chính

1. Mở **Phiếu mua sắm** và chọn **Tạo phiếu mới**.
2. Nhập tiêu đề, lý do, cost center, tiền tệ và ít nhất một dòng hàng.
3. Lưu để tạo Bản nháp.
4. Mở chi tiết và tải tài liệu cần thiết.
5. Nếu tổng từ 20.000.000 VND, tải ít nhất một tài liệu loại **Báo giá**.
6. Kiểm tra budget check rồi chọn **Gửi duyệt**.
7. Theo dõi timeline và trạng thái trong danh sách.

## Khi bị yêu cầu chỉnh sửa

1. Đọc comment của người duyệt trong timeline.
2. Chọn **Chỉnh sửa**.
3. Cập nhật nội dung hoặc tài liệu.
4. Lưu rồi chọn **Gửi lại**.

Nếu nhu cầu không còn, có thể hủy phiếu khi đang là Bản nháp hoặc Yêu cầu chỉnh sửa.

## Giới hạn cần nhớ

- Lý do phải có từ 10 đến 5.000 ký tự.
- Mỗi phiếu có 1–100 dòng hàng.
- Tệp hỗ trợ: PDF, DOCX, XLSX, JPEG và PNG; tối đa 10 MB.
- Tổng tiền do backend tính, không phải giá trị do trình duyệt tự quyết định.
- HTTP 403 nghĩa là tài khoản không có quyền/data scope, không phải sai mật khẩu.

Xem chi tiết tại [Hướng dẫn sử dụng tổng thể](../USER_GUIDE.md#6-hướng-dẫn-cho-employee).
