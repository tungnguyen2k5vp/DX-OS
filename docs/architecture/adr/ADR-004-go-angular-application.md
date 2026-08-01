# ADR-004: Go + Angular cho ứng dụng nghiệp vụ

## Status

Accepted

## Context

MVP cần UI hiện đại, REST API, workflow mua sắm, tích hợp Keycloak/Nextcloud/Metabase/RAGFlow và
thể hiện năng lực lập trình. NocoBase yêu cầu plugin thương mại cho OIDC/Approval chuyên dụng.

## Options

| Phương án | Ưu điểm | Nhược điểm |
|---|---|---|
| NocoBase | nhanh cho CRUD | phụ thuộc edition/plugin, ít code tự xây |
| Java + Angular | enterprise/BPMN mạnh | runtime/tooling nặng hơn |
| Go + Angular | API/integration gọn, image nhẹ, UI hiện đại | workflow phải tự xây |

## Decision

Chọn Angular SPA và Go modular monolith. PostgreSQL lưu nghiệp vụ; Keycloak là IdP. Không dùng
NocoBase trong MVP.

## Rationale

- Phù hợp lựa chọn công nghệ của nhóm.
- Một quy trình mua sắm đủ nhỏ để state machine tường minh.
- Go phù hợp API, integration và Agent Tool Gateway.
- Angular phù hợp dashboard/form/approval UI.

## Trade-offs

- Nhóm tự xây form, workflow UI, RBAC integration và audit.
- Không có BPMN designer sẵn.
- Cần kỷ luật contract/test để frontend/backend đồng bộ.

## Revisit trigger

- Có nhiều quy trình thay đổi thường xuyên cần người nghiệp vụ tự thiết kế.
- Workflow có parallel gateway/escalation phức tạp.
- Team tăng và module cần scale/vòng đời độc lập.

