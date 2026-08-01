---
title: Tài chính
description: Duyệt cuối, quản lý hạn mức và xem báo cáo theo tổ chức.
sidebar_position: 3
---

# Hướng dẫn cho Tài chính

Role Keycloak: **finance**.

## Phạm vi quyền

- Xem phiếu trong phạm vi tài chính được cấp.
- Xử lý bước cuối đối với phiếu Trưởng bộ phận đã duyệt.
- Xem và điều chỉnh allocation ngân sách.
- Xem báo cáo trong organization của mình.
- Không mặc nhiên tạo phiếu hoặc duyệt cấp trưởng bộ phận.

## Duyệt cuối

1. Mở **Phê duyệt**.
2. Chọn phiếu ở trạng thái Trưởng bộ phận đã duyệt.
3. Đối chiếu số tiền, cost center, tài liệu và reservation hiện có.
4. Chọn:
   - **Phê duyệt**: reservation chuyển thành committed;
   - **Yêu cầu chỉnh sửa**: reservation được giải phóng và phiếu trả về chủ sở hữu;
   - **Từ chối**: reservation được giải phóng và quy trình kết thúc.

Không được tự duyệt phiếu của mình nếu tài khoản được gán thêm role tạo phiếu.

## Điều chỉnh ngân sách

1. Mở **Ngân sách**.
2. Chọn allocation và nhấn **Điều chỉnh**.
3. Nhập hạn mức mới và lý do từ 10 đến 1.000 ký tự.
4. Xác nhận rồi kiểm tra lịch sử điều chỉnh.

Không thể giảm allocation xuống dưới tổng reserved + committed.

## Báo cáo

Mở **Báo cáo** và lọc theo ngày, department, cost center hoặc currency. Không cộng nhiều currency
thành một tổng chung. Metabase dùng tài khoản riêng, không dùng password Keycloak.

Xem chi tiết tại [Hướng dẫn sử dụng tổng thể](../USER_GUIDE.md#8-hướng-dẫn-cho-finance).
