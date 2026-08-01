# ADR-001: Repository tích hợp và Compose profile

## Status

Accepted

## Context

DX-OS lắp ghép nhiều sản phẩm có repository và vòng đời riêng. Nhóm nhỏ, thời gian MVP ngắn và máy
dev có khoảng 16 GB RAM.

## Decision

Giữ một repository `dx-os-lab` chứa source Go, Angular, cấu hình, adapter, migration, contract,
prompt, test và tài liệu. Không vendoring source của Keycloak, Nextcloud, Metabase hoặc RAGFlow.
Các cụm dịch vụ được bật bằng Compose profile theo giai đoạn.

## Rationale

- Giảm tải build và tránh fork không cần thiết.
- Nâng cấp từng sản phẩm độc lập.
- Phù hợp giới hạn tài nguyên local.
- Repository tập trung vào giá trị thật của đồ án: tích hợp và quản trị.

## Trade-offs

- Cần quản lý compatibility matrix giữa nhiều phiên bản.
- Debug xuyên sản phẩm khó hơn một monolith.
- Cần runbook và contract rõ ràng.

## Consequences

- Mỗi image phải được pin version.
- Mỗi profile có health check và tài liệu dependency.
- Chỉ clone source upstream khi phát triển/điều tra plugin cụ thể.
