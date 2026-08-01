---
id: docs-index
title: Chỉ mục tài liệu DX-OS
description: Bản đồ toàn bộ tài liệu kỹ thuật và nghiệp vụ của DX-OS.
slug: /tai-lieu
sidebar_position: 4
---

# Chỉ mục tài liệu DX-OS

Tài liệu này là điểm bắt đầu cho nhóm phát triển. Kiến trúc hiện tại đã chốt **Go + Angular** cho
ứng dụng do nhóm tự xây; Keycloak, PostgreSQL, Nextcloud và Metabase được tích hợp như các sản
phẩm độc lập. RAGFlow/Agent thuộc lộ trình tiếp theo và chưa được triển khai.

## Đọc theo vai trò

| Vai trò                 | Đọc trước                                                                                                                                                                                       |
| ----------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Người dùng nghiệp vụ    | [Hướng dẫn sử dụng và role](USER_GUIDE.md)                                                                                                                                                      |
| Tất cả thành viên dự án | [Implementation Guide](IMPLEMENTATION_GUIDE.md), [Project Brief](PROJECT_BRIEF.md)                                                                                                              |
| Backend Go              | [Backend Go](implementation/BACKEND_GO.md), [Database](implementation/DATABASE.md), [API](implementation/API.md)                                                                                |
| Frontend Angular        | [Frontend Angular](implementation/FRONTEND_ANGULAR.md), [UI Design System](implementation/UI_DESIGN_SYSTEM.md), [API](implementation/API.md), [Authentication](implementation/AUTHORIZATION.md) |
| DevOps                  | [Local Development](implementation/LOCAL_DEVELOPMENT.md), [Deployment](implementation/DEPLOYMENT.md), [Operations](implementation/OPERATIONS.md)                                                |
| QA                      | [Testing](implementation/TESTING.md), [Backlog](BACKLOG.md)                                                                                                                                     |
| Kiến trúc/giảng viên    | [System Context](architecture/CONTEXT.md), thư mục [ADR](architecture/adr/)                                                                                                                     |

## Tài liệu cốt lõi

1. [Hướng dẫn sử dụng và role](USER_GUIDE.md) — đăng nhập, menu, quy trình và quyền của sáu role.
2. [Implementation Guide](IMPLEMENTATION_GUIDE.md) — phạm vi, stack, giai đoạn và Definition of Done.
3. [System Context](architecture/CONTEXT.md) — thành phần, ranh giới và luồng dữ liệu.
4. [Repository Structure](implementation/REPOSITORY_STRUCTURE.md) — cấu trúc source dự kiến.
5. [Backend Go](implementation/BACKEND_GO.md) — module, state machine và quy tắc coding.
6. [Frontend Angular](implementation/FRONTEND_ANGULAR.md) — route, feature, state và UI.
7. [UI Design System](implementation/UI_DESIGN_SYSTEM.md) — Spartan UI, Tailwind và theme DX-OS.
8. [Database](implementation/DATABASE.md) — ERD, bảng, index và transaction.
9. [API](implementation/API.md) — chuẩn REST, endpoint, lỗi và idempotency.
10. [Authentication](implementation/AUTHORIZATION.md) — Keycloak, OIDC, RBAC và policy.
11. [Local Development](implementation/LOCAL_DEVELOPMENT.md) — cài đặt và chạy local.
12. [Deployment](implementation/DEPLOYMENT.md) — Dev, Demo/UAT và Production pilot.
13. [Testing](implementation/TESTING.md) — test pyramid và acceptance matrix.
14. [Operations](implementation/OPERATIONS.md) — log, metric, backup, restore và incident.
15. [Procurement MVP Runbook](runbooks/PROCUREMENT_MVP.md) — CRUD, workflow hai vòng duyệt, audit và ngân sách.
16. [Attachment Runbook](runbooks/ATTACHMENTS.md) — Nextcloud/WebDAV, policy, phân quyền và vận hành tài liệu.
17. [Reporting Runbook](runbooks/REPORTING.md) — curated views, KPI, Metabase read-only, RBAC và đối soát số liệu.

## Nguồn yêu cầu

- Tài liệu đầu vào nằm ngoài repository: `../DX_OS_Implementation_Guide.pdf` tính từ thư mục
  `dx-os-lab`.
- Các quyết định còn mở: [DECISIONS_NEEDED.md](DECISIONS_NEEDED.md)
- Phiên bản phần mềm: [THIRD_PARTY_BASELINE.md](THIRD_PARTY_BASELINE.md)
