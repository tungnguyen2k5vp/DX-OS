# Phát triển local

## 1. Trạng thái máy hiện tại

Kiểm tra ngày 2026-07-29:

- Docker Engine đang chạy.
- Node.js hiện là `24.14.0`; Angular 22 yêu cầu nhánh Node 24 từ `24.15.0` trở lên.
- Go chưa được cài.
- RAM khoảng 16 GB và ổ C còn khoảng 141 GB.

Application Skeleton dùng image `node:24.15.0-alpine` và `golang:1.26.6-alpine`, nên không bắt buộc
nâng Node/cài Go trên host. Chỉ cần cài toolchain host nếu muốn chạy ngoài Docker.

## 2. Prerequisites

- Git.
- Docker Desktop/Engine và Compose.
- Go 1.26.6.
- Node.js 24.15+.
- npm đi kèm Node.
- IDE hỗ trợ Go, Angular và EditorConfig.

Kiểm tra:

```powershell
git --version
docker version
docker compose version
go version
node --version
npm --version
```

`docker version` phải có cả Client và Server.

## 3. Environment

```powershell
Copy-Item .env.example .env
```

Thay tất cả `CHANGEME`. `.env` không được commit. Local chỉ dùng dữ liệu giả lập.

## 4. Chạy toàn bộ stack bằng Docker

```powershell
docker compose --profile foundation --profile application up -d --build
docker compose --profile foundation --profile application ps
```

Chi tiết và cách tạo user phát triển: [Application Skeleton runbook](../runbooks/APPLICATION_SKELETON.md).

## 5. Chạy riêng khi đã cài toolchain host

### Backend

```powershell
Set-Location backend
go mod download
go run ./cmd/migrate
go run ./cmd/api
```

API mặc định: `http://localhost:8081`.

### Frontend

```powershell
Set-Location frontend
npm ci
npm start
```

Angular mặc định: `http://localhost:4200`.

Khi scaffold frontend lần đầu:

```powershell
npm install --save-dev @spartan-ng/cli@1.1.2
ng g @spartan-ng/cli:init
ng g @spartan-ng/cli:ui
```

Chọn component MVP: button, badge, card, field, input, textarea, select, checkbox, dialog,
alert-dialog, sheet, sidebar, table/data-table, pagination, tabs, tooltip, skeleton, spinner và sonner.

## 6. Địa chỉ chạy cục bộ

| Service | URL |
|---|---|
| Angular | `http://localhost:4200` |
| Go API | `http://localhost:8081` |
| Keycloak | `http://localhost:8080` |
| Nextcloud | `http://localhost:8082` khi profile bật |
| Metabase | `http://localhost:3000` khi profile bật |
| RAGFlow | tách profile/máy; port theo compose upstream |

## 7. Proxy cục bộ

Angular dev server proxy `/api` đến Go API để tránh CORS phức tạp:

```json
{
  "/api": {
    "target": "http://localhost:8081",
    "secure": false,
    "changeOrigin": true
  }
}
```

OIDC redirect URI vẫn phải khai báo chính xác `http://localhost:4200/*`.

## 8. Tạo user test

Không commit mật khẩu test thật trong realm export:

```powershell
.\scripts\Initialize-DevUser.ps1
```

Credential tạm nằm trong `data/runtime/dev-user.txt`, không xuất hiện trong log và bị Git ignore.

## 9. Quy trình làm việc hằng ngày

```powershell
git pull --ff-only
docker compose --profile foundation --profile application up -d

# backend terminal
go test ./...
go run ./cmd/api

# frontend terminal
npm ci
npm test -- --watch=false
npm start
```

Trước khi push:

- format/lint;
- unit test;
- OpenAPI lint;
- migration validation;
- không có `.env`, token, database dump hoặc file người dùng.

## 10. Reset local

Reset database là destructive. Chỉ thực hiện với local test volume đã xác nhận:

```powershell
docker compose --profile foundation down
```

Không thêm `-v` trừ khi chắc chắn volume chỉ chứa dữ liệu local có thể tái tạo. Demo/UAT dùng script
reset riêng có xác nhận rõ target.
