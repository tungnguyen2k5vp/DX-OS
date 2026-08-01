---
title: Quản trị DX-OS
description: Trách nhiệm quản trị và giới hạn của dx_admin.
sidebar_position: 5
---

# Hướng dẫn cho Quản trị DX-OS

Role Keycloak: **dx_admin**.

## Quyền hiện tại

- Đăng nhập DX-OS.
- Xem trang Tổng quan.
- Xem báo cáo toàn phạm vi.
- Thực hiện công việc cấu hình/support qua công cụ quản trị được cấp.

## Không phải superuser nghiệp vụ

dx_admin không mặc nhiên được:

- tạo hoặc sửa phiếu;
- duyệt thay manager/finance;
- điều chỉnh ngân sách;
- đọc mọi hồ sơ như auditor;
- giả danh người dùng qua API thông thường.

Việc quản lý realm/user local thực hiện qua Keycloak Admin Console hoặc script vận hành. Phiên bản
hiện tại chưa có trang quản trị DX-OS riêng.

## Nguyên tắc support

- Dùng endpoint/quy trình support riêng nếu được triển khai.
- Mọi thao tác thay đổi phải ghi audit.
- Không sửa trạng thái trực tiếp trong PostgreSQL.
- Không gán role nghiệp vụ rộng chỉ để vượt lỗi 403.
- Không dùng bootstrap admin Keycloak làm service account runtime.

Xem chi tiết tại [Hướng dẫn sử dụng tổng thể](../USER_GUIDE.md#10-hướng-dẫn-cho-dx-admin).
