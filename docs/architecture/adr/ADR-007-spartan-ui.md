# ADR-007: Spartan UI làm design system Angular

## Status

Accepted

## Context

Nhóm muốn giao diện theo phong cách và triết lý open-code của shadcn/ui nhưng frontend đã chốt
Angular. shadcn/ui chính thức không cung cấp Angular component implementation trực tiếp.

## Options

| Phương án | Ưu điểm | Nhược điểm |
|---|---|---|
| Angular Material | ổn định, hệ sinh thái lớn | phong cách Material, tùy biến khác shadcn |
| Tự port shadcn | kiểm soát tuyệt đối | effort và accessibility risk cao |
| Spartan UI | Angular-native, Tailwind, copy-owned styles | dependency cộng đồng cần quản lý |
| Chuyển React | dùng shadcn chính thức | phá quyết định Angular và tài liệu hiện có |

## Decision

Dùng Spartan UI + Tailwind CSS v4 + Angular CDK. Spartan là component system duy nhất; không trộn
Angular Material.

## Rationale

- Giữ Angular và đạt trải nghiệm gần shadcn.
- Behavior primitives có accessibility; helm styles nằm trong source để tùy chỉnh.
- Phù hợp Standalone Components và cấu trúc feature-based.
- Tập component đủ cho admin/workflow UI.

## Trade-offs

- Cần review code do CLI copy vào repository.
- Update upstream không tự động merge customization.
- Nhóm chịu trách nhiệm visual regression và accessibility sau khi sửa helm.

## Revisit trigger

- Spartan không hỗ trợ Angular major đang dùng.
- Thiếu primitive quan trọng không thể bổ sung hợp lý.
- Maintenance/bug/security vượt khả năng nhóm.

