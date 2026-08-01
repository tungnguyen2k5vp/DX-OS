---
title: Điều phối AI
description: Trạng thái hiện tại và nguyên tắc cho giai đoạn RAG/Agent.
sidebar_position: 6
---

# Hướng dẫn cho Điều phối AI

Role Keycloak: **ai_operator**.

## Trạng thái hiện tại

Role đã tồn tại trong realm dx-os, nhưng RAG/Agent chưa được triển khai. Tài khoản ai_operator hiện:

- đăng nhập được;
- xem được trang Tổng quan;
- chưa có menu AI;
- chưa có recommendation queue;
- chưa có quyền chạy tool;
- không thay thế manager hoặc finance.

## Policy đã chốt cho giai đoạn sau

Khi Agent được triển khai:

1. AI chỉ tạo recommendation, không tự cấp quyền.
2. Tool phải nằm trong allowlist của Go backend.
3. Thao tác nhạy cảm cần human approval từ role nghiệp vụ phù hợp.
4. Backend kiểm authorization tại thời điểm thực thi.
5. Input, output, citation, approver và execution result phải được audit.
6. Nội dung RAG là dữ liệu, không được coi là system instruction.

Không mô phỏng tính năng AI bằng cách gán finance hoặc dx_admin cho ai_operator.

Xem chi tiết tại [Hướng dẫn sử dụng tổng thể](../USER_GUIDE.md#11-hướng-dẫn-cho-ai-operator).
