# Cấu trúc repository

## 1. Cấu trúc mục tiêu

```text
dx-os-lab/
├── backend/
│   ├── cmd/
│   │   ├── api/
│   │   ├── worker/
│   │   └── migrate/
│   ├── internal/
│   │   ├── platform/
│   │   ├── identity/
│   │   ├── organization/
│   │   ├── purchase/
│   │   ├── workflow/
│   │   ├── document/
│   │   ├── analytics/
│   │   ├── ai/
│   │   ├── agent/
│   │   ├── audit/
│   │   └── integration/
│   ├── migrations/
│   ├── queries/
│   ├── generated/
│   ├── tests/
│   ├── go.mod
│   └── Dockerfile
├── frontend/
│   ├── src/app/
│   │   ├── core/
│   │   ├── layout/
│   │   ├── shared/
│   │   └── features/
│   ├── public/
│   ├── components.json
│   ├── package.json
│   ├── package-lock.json
│   └── Dockerfile
├── contracts/
│   ├── openapi/
│   ├── events/
│   └── schemas/
├── compose/
├── data/
│   ├── postgres/
│   ├── metabase/
│   └── seed/
├── iam/keycloak/
├── human/nextcloud/
├── ai/
│   ├── knowledge/
│   ├── prompts/
│   ├── guardrails/
│   └── evaluations/
├── observability/
├── scripts/
├── tests/
│   ├── e2e/
│   ├── contract/
│   └── performance/
├── docs/
├── .env.example
└── docker-compose.yml
```

## 2. Quy tắc phụ thuộc

```text
Angular feature -> Angular core API client
Go handler -> Go application service -> repository/integration
Domain/business rule không import HTTP hoặc database driver
Platform code không chứa rule mua sắm
```

Không áp dụng Clean Architecture đầy đủ với nhiều lớp interface không có nhu cầu. Mỗi module Go dùng
cấu trúc thực dụng:

```text
purchase/
├── model.go
├── policy.go
├── service.go
├── repository.go
├── handler.go
└── *_test.go
```

Chỉ tạo interface tại boundary cần mock hoặc có khả năng thay implementation: clock, repository,
Nextcloud client, RAG client và tool executor.

## 3. Quyền sở hữu contract

- `contracts/openapi`: nguồn sự thật cho REST API.
- `contracts/events`: JSON Schema cho integration event.
- `backend/migrations`: nguồn sự thật cho schema database.
- `frontend`: sinh/viết typed API client từ OpenAPI; không tự định nghĩa DTO khác nghĩa.
- `frontend/src/app/shared/ui`: component helm do Spartan CLI copy vào và nhóm sở hữu source.

## 4. Branch và thay đổi

- `main` luôn build được.
- Feature branch ngắn: `feature/DXOS-123-purchase-submit`.
- Migration đã merge không sửa lại; tạo migration mới.
- Breaking API change cần version/deprecation và ADR.
- Dependency upgrade tách riêng khỏi feature.
