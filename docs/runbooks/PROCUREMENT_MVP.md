# Runbook: Procurement MVP — bước 1 đến 7

## Phạm vi đã triển khai

Runbook này bao phủ:

1. schema `organizations`, `departments`, `users`;
2. schema `purchase_requests`, `purchase_request_items`, `process_events` và workflow status;
3. API tạo, cập nhật, liệt kê và xem chi tiết phiếu;
4. state machine submit/resubmit/cancel và hai vòng phê duyệt;
5. Angular UI cho danh sách, form tạo/sửa, trang chi tiết và action panel;
6. timeline phân trang, comments và audit log cho toàn bộ mutation.
7. hạn mức ngân sách, kiểm tra khả dụng, reservation và commitment theo workflow.
8. dashboard quản trị ngân sách và điều chỉnh hạn mức có kiểm soát cho finance/auditor.

Chưa thuộc phạm vi: attachment và transactional outbox.

## Áp migration và chạy API

```powershell
docker compose --profile foundation --profile application up -d --build
docker compose --profile foundation --profile application ps
```

Migration runner áp lần lượt:

- `000001_bootstrap.sql`;
- `000002_procurement_mvp.sql`;
- `000003_procurement_audit.sql`.
- `000004_procurement_budget.sql`.
- `000005_budget_management.sql`.

Migration thứ hai seed organization `DX-OS` và department `GENERAL`. Khi user gọi API lần đầu,
backend ánh xạ `sub` từ Keycloak vào bảng `users`; không lưu password và không nhận
`requesterId`/`departmentId` từ client.

Migration thứ tư tạo kỳ ngân sách, phân bổ và reservation; seed hạn mức demo
`CC-GENERAL`/`VND` cho năm hiện tại. Hạn mức demo là `100.000.000.000 VND`.
Migration thứ năm tạo lịch sử điều chỉnh hạn mức với reason, actor, correlation ID và
`Idempotency-Key`.

## Endpoint

| Method | URL | Kết quả |
|---|---|---|
| `POST` | `/api/v1/purchase-requests` | tạo `DRAFT`, trả `201` + `Location` |
| `GET` | `/api/v1/purchase-requests?page=1&pageSize=20&status=DRAFT` | danh sách theo data scope |
| `GET` | `/api/v1/purchase-requests/{id}` | detail và items theo data scope |
| `PATCH` | `/api/v1/purchase-requests/{id}` | sửa draft với `expectedVersion` |
| `POST` | `/api/v1/purchase-requests/{id}/transitions` | thực hiện action với `Idempotency-Key` |
| `GET` | `/api/v1/purchase-requests/{id}/timeline?page=1&pageSize=20` | lịch sử xử lý theo scope |
| `GET` | `/api/v1/purchase-requests/{id}/budget-check` | khả dụng/reservation của phiếu theo scope |
| `GET` | `/api/v1/budgets/summary?costCenter=CC-GENERAL&currency=VND` | tổng hợp hạn mức cho manager/finance/auditor |
| `GET` | `/api/v1/budgets/dashboard` | allocation, tổng hợp, reservation và lịch sử cho finance/auditor |
| `PATCH` | `/api/v1/budgets/allocations/{id}` | finance điều chỉnh tổng hạn mức |

## Màn hình Angular

| Route | Màn hình |
|---|---|
| `/purchase-requests` | danh sách, lọc trạng thái và phân trang |
| `/purchase-requests/new` | form tạo draft với `FormArray` items |
| `/purchase-requests/{id}/edit` | sửa `DRAFT`/`CHANGES_REQUESTED` và giữ version |
| `/purchase-requests/{id}` | thông tin, items, total và action theo role/status |
| `/approvals` | hàng đợi `SUBMITTED` cho manager hoặc `MANAGER_APPROVED` cho finance |
| `/budgets` | dashboard ngân sách cho finance/auditor; auditor ở chế độ chỉ đọc |

Các route được lazy-load. UI dùng Signals cho loading/error/filter, RxJS cho HTTP, Reactive Forms
cho form tạo và Spartan UI/Tailwind cho button, card, badge cùng design tokens. Employee và manager
mới được vào route tạo; backend vẫn kiểm quyền cuối cùng.

Danh sách có skeleton, error/retry, empty state và pagination. Form hiển thị validation phía client,
field violation từ Problem Details và khóa nút trong lúc gửi. Sau khi tạo thành công, UI điều hướng
sang detail và hiển thị xác nhận. Form sửa tải dữ liệu mới nhất, gửi `expectedVersion` và báo người
dùng tải lại khi server trả `409`.

Menu `Phê duyệt` chỉ hiện cho `department_manager`/`finance`. Dashboard lấy số phiếu đang chờ từ
API; hàng đợi dẫn tới detail để approve, reject hoặc request changes.

Timeline hiển thị event mới nhất trước, actor/role, from/to status, comment và thời gian. Endpoint
không trả metadata hoặc idempotency key nội bộ. Quyền đọc timeline luôn kế thừa quyền đọc detail;
resource ngoài scope trả `404`.

Mỗi create/update/transition ghi đồng thời `process_events` và `audit_logs` trong cùng transaction.
Migration `000003` backfill audit từ các process event đã tồn tại trước khi tính năng audit được bật.

Trang detail có thẻ kiểm tra ngân sách tải song song với detail và timeline. Thẻ hiển thị tổng hạn
mức, đang giữ, đã cam kết, còn khả dụng và giá trị phiếu; cảnh báo rõ khi chưa cấu hình hoặc không đủ
ngân sách. UI chỉ trình bày kết quả, backend vẫn là nơi quyết định cuối cùng.

Ví dụ body:

```json
{
  "title": "Mua máy tính cho phòng kỹ thuật",
  "reason": "Thay thế thiết bị đã hết khấu hao.",
  "currency": "VND",
  "costCenter": "CC-GENERAL",
  "items": [
    {
      "description": "Laptop",
      "quantity": "2",
      "unit": "chiếc",
      "unitPrice": "25000000"
    }
  ]
}
```

Quantity, unit price, line total và total amount dùng chuỗi decimal trong JSON. PostgreSQL dùng
`numeric`, `line_total` là generated column và `total_amount` được tính lại trong transaction.

## Workflow hiện có

Database chấp nhận:

```text
DRAFT
SUBMITTED
MANAGER_APPROVED
CHANGES_REQUESTED
APPROVED
REJECTED
CANCELLED
```

Create luôn tạo `DRAFT` và event append-only `DRAFT_CREATED`. Luồng được backend cho phép:

| Trạng thái | Actor | Action | Trạng thái mới |
|---|---|---|---|
| `DRAFT` | requester | `SUBMIT` | `SUBMITTED` |
| `DRAFT` | requester | `CANCEL` | `CANCELLED` |
| `CHANGES_REQUESTED` | requester | `RESUBMIT` | `SUBMITTED` |
| `CHANGES_REQUESTED` | requester | `CANCEL` | `CANCELLED` |
| `SUBMITTED` | manager cùng department | `APPROVE` | `MANAGER_APPROVED` |
| `SUBMITTED` | manager cùng department | `REJECT` | `REJECTED` |
| `SUBMITTED` | manager cùng department | `REQUEST_CHANGES` | `CHANGES_REQUESTED` |
| `MANAGER_APPROVED` | finance cùng organization | `APPROVE` | `APPROVED` |
| `MANAGER_APPROVED` | finance cùng organization | `REJECT` | `REJECTED` |
| `MANAGER_APPROVED` | finance cùng organization | `REQUEST_CHANGES` | `CHANGES_REQUESTED` |

Requester không được tự phê duyệt dù tài khoản đồng thời có role manager/finance. `REJECT` và
`REQUEST_CHANGES` bắt buộc có comment.

Mọi update/transition tăng `version`; client phải gửi `expectedVersion`. Version cũ trả `409`.
Transition bắt buộc header `Idempotency-Key` dài 8–255 ký tự. Phát lại cùng key, phiếu, actor và
action trả trạng thái đã có mà không tăng version; dùng lại key cho ý định khác trả `409`.

## Quy tắc ngân sách

Ngân sách được quản lý theo `organization + period + costCenter + currency`. Các số tiền dùng
`numeric(19,4)`; API trả chuỗi decimal để không mất độ chính xác.

| Workflow | Tác động ngân sách |
|---|---|
| manager `APPROVE` từ `SUBMITTED` | khóa allocation và tăng `reserved_amount` |
| finance `APPROVE` từ `MANAGER_APPROVED` | giảm reserved, tăng `committed_amount` |
| finance `REJECT`/`REQUEST_CHANGES` | giảm reserved và đánh dấu reservation `RELEASED` |
| manager approve khi thiếu cấu hình | rollback toàn bộ, trả `409 budget-not-configured` |
| manager approve khi không đủ tiền | rollback toàn bộ, trả `409 insufficient-budget` |

Purchase request, allocation và reservation được cập nhật trong cùng PostgreSQL transaction.
Allocation bị khóa bằng `FOR UPDATE`, vì vậy hai manager phê duyệt đồng thời không thể cùng tiêu
phần tiền còn lại. Event `BUDGET_RESERVED`, `BUDGET_COMMITTED`, `BUDGET_RELEASED` được ghi vào
timeline và audit trong cùng transaction. Idempotent replay kết thúc trước thao tác ngân sách nên
không giữ hoặc cam kết trùng.

Quyền xem tổng hợp:

- `department_manager`: cost center của chính department;
- `finance` và `auditor`: các cost center trong organization của user;
- `employee`: không đọc endpoint tổng hợp, nhưng được xem budget check của phiếu thuộc sở hữu.

## Dashboard quản trị ngân sách

Menu `Ngân sách` chỉ xuất hiện khi access token có role `finance` hoặc `auditor`; route `/budgets`
có guard Angular và API tiếp tục kiểm role phía server.

Dashboard hiển thị:

- tổng hạn mức, đang giữ, đã cam kết và còn khả dụng theo currency;
- allocation theo kỳ/cost center, version và phần trăm sử dụng;
- cảnh báo `WARNING` từ 75% và `CRITICAL` từ 90% hoặc hết khả dụng;
- tối đa 50 reservation/commitment/release mới nhất, liên kết về phiếu;
- tối đa 50 lần điều chỉnh hạn mức với số trước/sau, actor, thời gian và lý do.

`finance` có thể điều chỉnh `allocated_amount`. Request bắt buộc có `expectedVersion`,
`Idempotency-Key`, reason từ 10 đến 1000 ký tự và hạn mức mới không thấp hơn
`reserved_amount + committed_amount`.

Mỗi điều chỉnh tăng version, ghi `budget_adjustments` và `audit_logs` trong cùng transaction.
`auditor` nhận `canManage=false` và backend trả `403` nếu thử gọi PATCH. `employee` bị chặn cả
dashboard lẫn API.

## Authorization

- `employee`: tạo và đọc phiếu của mình.
- `department_manager`: tạo và đọc phiếu cùng department.
- `finance`: chỉ đọc phiếu đã tới scope tài chính.
- `auditor`: read-only toàn bộ phiếu.
- `dx_admin` không được dùng như superuser nghiệp vụ.

Resource nằm ngoài scope trả `404`.

## Kiểm thử

```powershell
docker run --rm -v "${PWD}\backend:/workspace" -w /workspace `
  golang:1.26.5-alpine sh -c "go vet ./... && go test ./..."

.\scripts\Test-Foundation.ps1
.\scripts\Test-Application.ps1
.\scripts\Test-OIDCLogin.ps1 -TestProcurement
.\scripts\Test-ProcurementWorkflow.ps1
.\scripts\Test-BudgetManagement.ps1
```

Workflow smoke test tạo ba user Keycloak riêng biệt, đăng nhập bằng Authorization Code + PKCE rồi
kiểm tra `DRAFT -> SUBMITTED -> MANAGER_APPROVED -> APPROVED`. Test còn xác minh stale update trả
`409`, idempotent replay không tăng version, timeline workflow/budget, reservation chuyển sang
commitment, request changes hoàn tiền, vượt hạn mức bị chặn `409`, comments đúng và response không
lộ metadata/idempotency key. Script không in access token/password và luôn xóa token tạm;
credential development nằm trong `data/runtime` đã được Git ignore.

Budget management smoke test đăng nhập ba user `finance`, `auditor`, `employee`; xác minh finance
đọc/ghi, auditor chỉ đọc, employee nhận `403`, idempotent replay không tăng version, stale version
trả `409`, adjustment xuất hiện trong history và hạn mức được khôi phục sau test.

Frontend:

```powershell
docker build --target test -t dxos-frontend-test frontend
docker build --target build -t dxos-frontend-build frontend
```

Unit test kiểm typed HTTP service (bao gồm PATCH/transition/header idempotency/budget), form validation và
pipe định dạng decimal; production build kiểm toàn bộ lazy route và strict Angular template.

OpenAPI:

```powershell
docker run --rm -v "${PWD}:/workspace" -w /workspace node:24.15.0-alpine `
  npx --yes @stoplight/spectral-cli@6.15.0 lint contracts/openapi/dx-os-v1.yaml
```
