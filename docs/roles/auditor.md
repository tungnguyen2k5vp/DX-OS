---
title: Kiểm toán
description: Đối soát hồ sơ, ngân sách và báo cáo ở chế độ chỉ đọc.
sidebar_position: 4
---

# Hướng dẫn cho Kiểm toán

Role Keycloak: **auditor**.

## Phạm vi quyền

- Xem hồ sơ và timeline theo mandate.
- Tải tài liệu của hồ sơ được phép xem.
- Xem dashboard ngân sách toàn phạm vi.
- Xem báo cáo toàn phạm vi.
- Không tạo, sửa, duyệt hoặc điều chỉnh dữ liệu.

## Quy trình đối soát

1. Mở phiếu cần kiểm tra.
2. Đối chiếu requester, department, cost center, dòng hàng và tổng tiền.
3. Kiểm tra attachment type, checksum/metadata và điều kiện báo giá.
4. Đọc timeline: actor, role, timestamp, status và comment.
5. Mở **Ngân sách** để đối chiếu allocation, reservation, commitment và adjustment.
6. Mở **Báo cáo** với cùng khoảng ngày/currency để so sánh số liệu.

## Dấu hiệu phải báo lỗi

- Có nút Điều chỉnh ngân sách.
- Một lệnh tạo/sửa/duyệt trả về thành công.
- Không truy vết được actor hoặc event của thay đổi nghiệp vụ.
- Số liệu báo cáo khác curated view khi dùng cùng bộ lọc.

Không cấp thêm finance cho tài khoản auditor để “tiện kiểm tra”; điều đó phá vỡ nguyên tắc
read-only.

Xem chi tiết tại [Hướng dẫn sử dụng tổng thể](../USER_GUIDE.md#9-hướng-dẫn-cho-auditor).
