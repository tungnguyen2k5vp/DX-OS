---
title: Backlog triển khai
sidebar_position: 90
---

# Backlog triển khai Go + Angular

## Giai đoạn 0 — Chuẩn bị

- [x] Chốt kịch bản dọc: đề nghị mua sắm nội bộ.
- [x] Chốt Go + Angular, loại NocoBase khỏi kiến trúc đích.
- [x] Chọn modular monolith và state machine trong Go.
- [x] Viết kiến trúc, ADR, database, API, auth, deployment và test docs.
- [x] Tạo Compose foundation PostgreSQL + Keycloak.
- [ ] Chốt team, deadline, ngân sách hạ tầng và domain.
- [ ] Chốt provider LLM/embedding và data policy.

## Giai đoạn 1 — Foundation

- [x] Pin Node 24.15 và Go 1.26.5 qua Docker (cài host là tùy chọn).
- [x] Bật Docker Engine.
- [x] Validate và chạy PostgreSQL + Keycloak.
- [x] Xác minh database/user riêng.
- [x] Tạo Keycloak clients `dx-web`, `dx-api`.
- [x] Seed user `employee.demo`/role bằng secret local ngoài Git.
- [x] Test đăng nhập thực tế, role claim và token audience bằng Authorization Code + PKCE.
- [x] Test OIDC discovery, PKCE và cấu hình audience mapper.
- [ ] Backup/restore foundation lần đầu.

**Exit:** PostgreSQL healthy; realm hoạt động; token hợp lệ có audience/role đúng.

## Giai đoạn 2 — Application skeleton

### Backend

- [x] Tạo Go module và `cmd/api`, `cmd/worker`, `cmd/migrate`.
- [x] Config validation, structured log, correlation ID.
- [x] Database pool, migration runner và health endpoints.
- [x] JWT/JWKS validation.
- [x] OpenAPI skeleton và Problem Details.
- [x] Dockerfile multi-stage.

### Frontend

- [x] Tạo Angular 22 standalone application.
- [x] Tailwind CSS v4, Spartan UI, theme DX-OS, layout shell và runtime config.
- [x] OIDC Authorization Code + PKCE.
- [x] HTTP interceptor, bootstrap error state và guard.
- [x] Dashboard placeholder, identity/role display và navigation shell.
- [x] Dockerfile build/static Nginx.

### CI

- [x] Go format/vet/test/build.
- [x] Angular format/test/build và production dependency audit.
- [x] OpenAPI lint bằng Spectral.
- [ ] Event/schema lint mở rộng và secret scan.

**Exit:** login Keycloak -> Angular -> gọi API `/me` thành công; CI xanh.

## Giai đoạn 3 — Quy trình mua sắm MVP

- [x] Migration organization/users/departments.
- [x] Migration purchase requests/items/process events và workflow status.
- [x] Migration audit log và backfill từ process events.
- [ ] Transactional outbox.
- [x] Seed organization/department demo và tự ánh xạ user từ Keycloak subject.
- [x] Seed budget demo theo kỳ, cost center và currency.
- [x] API create/list/detail draft.
- [x] API update draft.
- [x] Angular list, create form và detail page.
- [x] Tính total server-side.
- [x] Transition submit/resubmit/cancel.
- [x] Transition manager approve/reject/request changes.
- [x] Transition finance approve/reject/request changes.
- [x] Rule giữ/cam kết/hoàn ngân sách và chặn vượt hạn mức.
- [x] Dashboard quản trị ngân sách cho finance/auditor.
- [x] Điều chỉnh hạn mức có version, idempotency, reason và audit.
- [x] Danh sách reservation/commitment, lịch sử điều chỉnh và cảnh báo sử dụng.
- [x] Rule attachment threshold và chặn submit/resubmit khi thiếu báo giá.
- [x] Optimistic locking và idempotency.
- [x] Approval inbox theo role và vòng duyệt.
- [x] Timeline sự kiện phân trang và comments history.
- [x] Negative RBAC/ownership tests cho state machine và ownership.

**Exit:** E2E từ tạo phiếu đến APPROVED/REJECTED; requester không tự duyệt.

## Giai đoạn 4 — Tài liệu

- [x] Bật Nextcloud trong application profile và database riêng.
- [x] Giữ Nextcloud nội bộ sau Go API; không yêu cầu người dùng đăng nhập trực tiếp ở MVP.
- [x] Viết adapter WebDAV.
- [x] Metadata hai pha, checksum SHA-256 và ETag.
- [x] File type/size/path policy.
- [x] Angular attachment card.
- [x] Kiểm tra quyền theo scope và trạng thái; bù trừ lỗi WebDAV.
- [ ] Job định kỳ phát hiện/dọn metadata kẹt và orphan.
- [ ] Antivirus/CDR cho tài liệu tải lên.

**Exit:** báo giá gắn đúng phiếu và user ngoài scope không đọc được.

## Giai đoạn 5 — Data và dashboard

- [x] Curated schema/views.
- [x] KPI lead time, return rate, SLA breach, attachment compliance và budget usage.
- [x] Data quality/reconciliation smoke test.
- [x] Metabase database user read-only.
- [x] Angular dashboard và filter thời gian/department/cost center/currency.
- [x] Metabase container và application database riêng.
- [x] Verify KPI với fixture và curated view.
- [x] Provision collection, 8 cards và dashboard Metabase với filter ngày/tiền tệ.

**Exit:** dashboard khớp operational data và có bằng chứng reconciliation.

## Giai đoạn 6 — RAG

- [ ] Chuẩn bị máy/profile đủ tài nguyên cho RAGFlow.
- [ ] Chốt knowledge owner/classification/retention.
- [ ] Ingest tài liệu quy định mua sắm.
- [ ] API proxy/query qua Go.
- [ ] Angular AI assistant.
- [ ] Evaluation set và expected citations.
- [ ] Groundedness/correctness/refusal tests.
- [ ] Prompt injection test.

**Exit:** câu trả lời có nguồn hoặc từ chối đúng khi thiếu bằng chứng.

## Giai đoạn 7 — Agent có kiểm soát

- [ ] Agent run/recommendation/approval/tool-call tables.
- [ ] Structured recommendation contract.
- [ ] Tool allowlist và JSON validation.
- [ ] Decision API và approval expiry/hash.
- [ ] Execution API có idempotency và precondition recheck.
- [ ] Angular recommendation/approval UI.
- [ ] Audit/redaction/rate limit.
- [ ] Test chưa approve/sai quyền/replay/concurrent state change.

**Exit:** Agent không thể thực thi trước approval; tool call truy vết được.

## Giai đoạn 8 — Hardening và nghiệm thu

- [ ] HTTPS/reverse proxy/security headers.
- [ ] Secret manager và rotation runbook.
- [ ] Metrics/log/tracing/alerts.
- [ ] Dependency, image, license và SBOM scan.
- [ ] Backup/restore drill tất cả dữ liệu.
- [ ] Seed/reset dataset demo.
- [ ] Performance baseline và security test report.
- [ ] Demo 10–15 phút và rehearsal.
- [ ] SRS, ERD, API, test report, user/admin guide.

## Sprint 1 đề xuất

1. Sửa prerequisites và chạy foundation.
2. Scaffold Go API + Angular shell.
3. Hoàn thành OIDC end-to-end.
4. Tạo migration tổ chức và purchase request.
5. Implement create/list/detail draft.
