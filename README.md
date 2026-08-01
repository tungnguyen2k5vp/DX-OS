# DX-OS Lab

DX-OS Lab là nguyên mẫu nền tảng vận hành doanh nghiệp số, triển khai một quy trình mua sắm nội bộ
có xác thực, phân quyền, kiểm soát ngân sách, tài liệu đính kèm, audit trail và báo cáo vận hành.

Ứng dụng nghiệp vụ được xây bằng **Go + Angular**. Giao diện sử dụng **Spartan UI + Tailwind CSS
v4**, mang cách tổ chức component theo triết lý shadcn nhưng phù hợp với Angular. Các nền tảng
Keycloak, PostgreSQL, Nextcloud và Metabase chạy độc lập, được ghép lại bằng Docker Compose.

> Đây là dự án lab/MVP. RAG và Agent đã có trong lộ trình kiến trúc nhưng **chưa được triển khai**
> trong phiên bản hiện tại.

## Tính năng đã có

- Đăng nhập một lần qua Keycloak bằng Authorization Code + PKCE S256.
- Sáu role nghiệp vụ: employee, department_manager, finance, auditor, ai_operator và dx_admin.
- Tạo, sửa, lọc, xem chi tiết và theo dõi lịch sử phiếu mua sắm.
- Quy trình duyệt hai cấp: trưởng bộ phận rồi tài chính.
- Chống tự duyệt, kiểm tra data scope, optimistic locking và idempotent transition.
- Giữ ngân sách khi trưởng bộ phận duyệt, cam kết khi tài chính duyệt và hoàn ngân sách khi phiếu
  bị trả sửa hoặc từ chối.
- Dashboard ngân sách, cảnh báo sử dụng và lịch sử điều chỉnh hạn mức.
- Lưu tệp trên Nextcloud; metadata, SHA-256 và ETag được quản lý bởi Go API.
- Bắt buộc báo giá đối với phiếu từ 20.000.000 VND trước khi gửi duyệt.
- Báo cáo Angular và dashboard Metabase đọc từ schema reporting chỉ đọc.
- OpenAPI contract, smoke test PowerShell và CI cho backend, frontend, contract.

## Kiến trúc

```text
Browser
  |
  +-- Angular SPA :4200 ---- đăng nhập ----> Keycloak :8080
  |        |
  |        +---- Bearer JWT ----> Go API :8081
  |                                  |
  |                    +-------------+-------------+
  |                    |                           |
  |              PostgreSQL :5432           Nextcloud :8082
  |                    |
  |             schema reporting
  |                    |
  +--------------------------------------> Metabase :3000
```

- **Angular** chỉ hiển thị chức năng theo role; quyền cuối cùng luôn do Go API kiểm tra.
- **Keycloak** quản lý danh tính, mật khẩu, phiên đăng nhập và realm role. Go API không lưu mật
  khẩu và không tự phát token.
- **PostgreSQL** lưu dữ liệu nghiệp vụ, audit, ngân sách và dữ liệu cấu hình của các dịch vụ.
- **Nextcloud** là kho tệp nội bộ; người dùng DX-OS không cần đăng nhập trực tiếp.
- **Metabase** là công cụ BI độc lập, dùng tài khoản riêng và chỉ đọc curated views.

## Công nghệ

| Lớp           | Công nghệ                                             |
| ------------- | ----------------------------------------------------- |
| Backend       | Go 1.26.5, modular monolith, REST API                 |
| Frontend      | Angular 22, TypeScript 6, Spartan UI, Tailwind CSS v4 |
| Identity      | Keycloak 26.7, OIDC/OAuth 2.0, PKCE S256              |
| Database      | PostgreSQL 18.4                                       |
| File storage  | Nextcloud 34, WebDAV qua backend                      |
| Reporting     | Curated PostgreSQL views, Metabase 0.63               |
| Local runtime | Docker Compose profiles                               |
| Contract/CI   | OpenAPI 3, Spectral, GitHub Actions                   |

Các image được khóa phiên bản trong [.env.example](.env.example). Không tự ý đổi sang tag latest.

## Cấu trúc repository

```text
dx-os-lab/
├── backend/              Go API, worker, migration và domain modules
├── frontend/             Angular SPA và source-owned UI components
├── contracts/openapi/    OpenAPI contract
├── compose/              Foundation, application và reporting services
├── data/postgres/        Bootstrap PostgreSQL/reporting
├── docs-site/            Docusaurus documentation portal
├── iam/keycloak/realm/   Realm export, clients và sáu realm role
├── scripts/              Provisioning và smoke tests PowerShell
├── docs/                 Kiến trúc, triển khai, runbook và hướng dẫn
├── docker-compose.yml    Compose entry point
└── .env.example          Mẫu cấu hình local
```

## Yêu cầu cài đặt

Đường chạy khuyến nghị là Docker, nên máy host chỉ bắt buộc có:

- Git;
- Docker Desktop hoặc Docker Engine đang chạy;
- Docker Compose 2.26 trở lên;
- PowerShell 7 được khuyến nghị để chạy các script.

Kiểm tra môi trường:

```powershell
git --version
docker version
docker compose version
$PSVersionTable.PSVersion
```

Nếu chạy source trực tiếp ngoài container, cần thêm Go 1.26.5, Node.js 24.15+ và npm 11.12.1.

## Cài đặt nhanh bằng Docker

### 1. Tạo cấu hình local

Từ thư mục dx-os-lab:

```powershell
Copy-Item .env.example .env
```

Mở file .env và thay **toàn bộ** giá trị có tiền tố CHANGEME bằng mật khẩu ngẫu nhiên riêng. Không
commit .env, token hoặc các tệp trong data/runtime.

Kiểm tra không còn secret mẫu:

```powershell
Select-String -Path .env -Pattern 'CHANGEME'
```

Lệnh đúng sẽ không trả về dòng nào.

### 2. Kiểm tra và dựng stack

```powershell
docker compose --profile foundation --profile application --profile reporting config --quiet
docker compose --profile foundation --profile application --profile reporting up -d --build
docker compose --profile foundation --profile application --profile reporting ps
```

Lần đầu Docker cần tải nhiều image và Nextcloud/Metabase cần thời gian bootstrap. Chờ các service
postgres, nextcloud, api, web và metabase chuyển sang running/healthy trước bước tiếp theo.

Xem log nếu service chưa sẵn sàng:

```powershell
docker compose --profile foundation --profile application --profile reporting logs --tail 200
```

### 3. Tạo tài khoản nghiệp vụ local

Script tạo hoặc cập nhật user, cấp đúng realm role, sinh mật khẩu ngẫu nhiên và ghi credential vào
data/runtime (thư mục đã được Git ignore):

```powershell
.\scripts\Initialize-DevUser.ps1 -Username employee.demo -Role employee -CredentialsPath data\runtime\employee-demo.txt
.\scripts\Initialize-DevUser.ps1 -Username manager.demo -Role department_manager -CredentialsPath data\runtime\manager-demo.txt
.\scripts\Initialize-DevUser.ps1 -Username finance.demo -Role finance -CredentialsPath data\runtime\finance-demo.txt
.\scripts\Initialize-DevUser.ps1 -Username auditor.demo -Role auditor -CredentialsPath data\runtime\auditor-demo.txt
.\scripts\Initialize-DevUser.ps1 -Username ai.operator.demo -Role ai_operator -CredentialsPath data\runtime\ai-operator-demo.txt
.\scripts\Initialize-DevUser.ps1 -Username admin.demo -Role dx_admin -CredentialsPath data\runtime\admin-demo.txt
```

Mỗi lần chạy lại script, mật khẩu của user tương ứng được đổi. Không chia sẻ hoặc commit các file
credential này.

### 4. Provision Metabase

```powershell
.\scripts\Initialize-Metabase.ps1
```

Script idempotent sẽ tạo data source chỉ đọc, collection, 8 câu hỏi và dashboard có 3 bộ lọc.
Tài khoản Metabase local được ghi tại data/runtime/metabase-admin.txt. Đây là tài khoản riêng,
không phải tài khoản Keycloak.

### 5. Chạy smoke tests

```powershell
.\scripts\Test-Foundation.ps1
.\scripts\Test-Application.ps1
.\scripts\Test-OIDCLogin.ps1
.\scripts\Test-OIDCLogin.ps1 -TestProcurement
.\scripts\Test-ProcurementWorkflow.ps1
.\scripts\Test-BudgetManagement.ps1
.\scripts\Test-Attachments.ps1
.\scripts\Test-Reporting.ps1
```

Các script kiểm tra từ cấu hình Keycloak, JWT, CRUD/workflow, ngân sách, Nextcloud đến quyền
read-only và các card của Metabase.

### 6. Mở ứng dụng

| Thành phần    | Địa chỉ               | Ghi chú                                           |
| ------------- | --------------------- | ------------------------------------------------- |
| DX-OS Angular | http://localhost:4200 | Điểm vào cho người dùng nghiệp vụ                 |
| Go API        | http://localhost:8081 | Health: /health/live và /health/ready             |
| Keycloak      | http://localhost:8080 | Realm dx-os; console admin dùng secret trong .env |
| PostgreSQL    | 127.0.0.1:5432        | Chỉ bind loopback cho local                       |
| Nextcloud     | http://localhost:8082 | Dịch vụ nội bộ, không cần user đăng nhập          |
| Metabase      | http://localhost:3000 | Credential trong data/runtime/metabase-admin.txt  |
| DX-OS Docs    | http://localhost:4300 | Website tài liệu Docusaurus                       |

Luôn mở ứng dụng bằng **http://localhost:4200**. Không đổi thành 127.0.0.1 nếu chưa cập nhật
Keycloak, vì client dx-web chỉ cho phép redirect URI http://localhost:4200/*.

Website tài liệu chạy bằng profile độc lập:

```powershell
docker compose --profile documentation up -d --build docs
```

## Chạy source ngoài Docker

Vẫn nên giữ PostgreSQL, Keycloak và Nextcloud trong Docker.

Backend:

```powershell
Set-Location backend
go mod download
go run ./cmd/migrate
go run ./cmd/api
```

Frontend, trong terminal khác:

```powershell
Set-Location frontend
npm ci
npm start
```

Các biến môi trường phải trỏ đúng tới dịch vụ local. Xem
[hướng dẫn phát triển local](docs/implementation/LOCAL_DEVELOPMENT.md) trước khi dùng cách này.

## Kiểm thử khi phát triển

```powershell
# Backend
Set-Location backend
Get-ChildItem -Recurse -Filter *.go | ForEach-Object { gofmt -w $_.FullName }
go vet ./...
go test -race ./...
go build ./cmd/api ./cmd/worker ./cmd/migrate

# Frontend
Set-Location ..\frontend
npm ci
npm run format:check
npm test -- --watch=false
npm run build

# OpenAPI, chạy từ repository root
Set-Location ..
npx --yes @stoplight/spectral-cli@6.15.0 lint contracts/openapi/dx-os-v1.yaml

# Website tài liệu
Set-Location docs-site
npm ci
npm run typecheck
npm run build
npm audit --audit-level=high
```

Khi container tài liệu đang chạy:

```powershell
.\scripts\Test-Documentation.ps1
```

API contract hiện hành: [contracts/openapi/dx-os-v1.yaml](contracts/openapi/dx-os-v1.yaml).

## Vận hành local

Xem trạng thái và log:

```powershell
docker compose --profile foundation --profile application --profile reporting ps
docker compose --profile foundation --profile application --profile reporting logs -f api web
```

Khởi động lại mà không build:

```powershell
docker compose --profile foundation --profile application --profile reporting up -d
```

Dừng container nhưng giữ dữ liệu:

```powershell
docker compose --profile foundation --profile application --profile reporting down
```

Không thêm cờ -v trừ khi chắc chắn muốn xóa toàn bộ volume PostgreSQL/Nextcloud local. Việc đó làm
mất dữ liệu và buộc bootstrap lại từ đầu.

## Lỗi thường gặp

### Keycloak báo Invalid parameter: redirect_uri

- Truy cập đúng http://localhost:4200, không dùng 127.0.0.1 hoặc một port khác.
- Kiểm tra redirect URI của client dx-web là http://localhost:4200/*.
- Xóa URL đăng nhập cũ trên thanh địa chỉ, mở lại trang gốc và đăng nhập lại.

### Invalid username or password

Chạy lại Initialize-DevUser.ps1 cho đúng username/role rồi đọc mật khẩu mới trong file credential
đã chỉ định. Script đổi mật khẩu sau mỗi lần chạy.

### Trang có HTML nhưng mất CSS

Rebuild web container và kiểm tra asset:

```powershell
docker compose --profile foundation --profile application build --no-cache web
docker compose --profile foundation --profile application up -d web
docker compose logs --tail 100 web
```

Sau đó hard refresh trình duyệt bằng Ctrl+F5.

### API trả 401 hoặc 403

- 401: phiên/token không hợp lệ; đăng xuất rồi đăng nhập lại.
- 403: tài khoản đã đăng nhập nhưng role hoặc data scope không cho phép thao tác.
- Sau khi đổi role, cần đăng nhập lại để nhận token mới.

### Metabase chỉ hiện 0 hoặc không có dữ liệu

- Đảm bảo đã có phiếu phù hợp với khoảng ngày, trạng thái và tiền tệ đang lọc.
- Chạy lại Initialize-Metabase.ps1 rồi Test-Reporting.ps1.
- Kiểm tra timezone Asia/Bangkok và bộ lọc ngày.

### Port đã được sử dụng

Đổi port host tương ứng trong .env hoặc dừng tiến trình đang giữ 3000, 4200, 5432, 8080, 8081,
8082; sau đó chạy lại docker compose config và up.

## Phân quyền và bảo mật

- Sáu role là quyền nghiệp vụ của realm dx-os, không phải sáu tài khoản cố định.
- Một tài khoản có thể có nhiều role, nhưng tài khoản demo nên tách riêng để kiểm thử rõ ràng.
- dx_admin không phải superuser dữ liệu nghiệp vụ và không được mặc nhiên duyệt thay người dùng.
- Việc ẩn menu trên Angular chỉ là UX; Go API luôn kiểm role, ownership, department và organization.
- Không dùng mật khẩu local cho demo công khai hoặc production.
- Không public Keycloak admin, PostgreSQL, Nextcloud hay Metabase trực tiếp ra Internet.

Chi tiết thao tác cho từng role nằm trong [Hướng dẫn sử dụng](docs/USER_GUIDE.md).

## Tài liệu

- [Website tài liệu DX-OS](https://vanvuong2005827.github.io/DX-OS/)
- [Hướng dẫn sử dụng và role](docs/USER_GUIDE.md)
- [Chỉ mục tài liệu](docs/INDEX.md)
- [Implementation Guide](docs/IMPLEMENTATION_GUIDE.md)
- [Kiến trúc hệ thống](docs/architecture/CONTEXT.md)
- [Authentication và Authorization](docs/implementation/AUTHORIZATION.md)
- [Procurement MVP runbook](docs/runbooks/PROCUREMENT_MVP.md)
- [Attachment runbook](docs/runbooks/ATTACHMENTS.md)
- [Reporting runbook](docs/runbooks/REPORTING.md)
- [Backlog và lộ trình](docs/BACKLOG.md)

Tài liệu yêu cầu gốc: [DX_OS_Implementation_Guide.pdf](../DX_OS_Implementation_Guide.pdf).

## Lộ trình tiếp theo

1. RAG có citation và bộ đánh giá retrieval/answer.
2. Agent có tool allowlist, human approval và audit đầy đủ.
3. Hardening: observability, backup/restore drill, bảo mật và acceptance test.
4. Chuẩn hóa triển khai Demo/UAT rồi mới thiết kế production pilot.
