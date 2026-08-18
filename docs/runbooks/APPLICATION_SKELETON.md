# Hướng dẫn vận hành: Khung ứng dụng

## 1. Thành phần

- Go API: `http://localhost:8081`.
- Angular/Nginx: `http://localhost:4200`.
- Keycloak: `http://localhost:8080`.
- PostgreSQL: `127.0.0.1:5432`.

API công khai `/health/live`, `/health/ready`. Endpoint `/api/v1/me` yêu cầu access token có issuer
`dx-os`, audience `dx-api` và chữ ký RS256 hợp lệ từ JWKS của Keycloak.

## 2. Khởi động

Từ repository root:

```powershell
docker compose --profile foundation --profile application up -d --build
docker compose --profile foundation --profile application ps
```

Service `migrate` là one-shot: trạng thái đúng sau khi chạy là `Exited (0)`. API chỉ bắt đầu sau khi
migration thành công; Web chỉ bắt đầu sau khi API healthy.

## 3. Tạo/reset user phát triển

```powershell
.\scripts\Initialize-DevUser.ps1
```

Script tạo `employee.demo`, gán role `employee`, sinh mật khẩu local ngẫu nhiên và lưu tại
`data/runtime/dev-user.txt`. File này bị Git ignore.

Không gửi file credential qua chat/email và không sử dụng user này ngoài local development.

## 4. Đăng nhập

1. Mở `http://localhost:4200`.
2. Angular chuyển sang màn hình đăng nhập realm `dx-os`.
3. Dùng credential trong `data/runtime/dev-user.txt`.
4. Sau callback, dashboard phải hiển thị username và role do `/api/v1/me` trả về.

Trang quản trị Keycloak `http://localhost:8080/admin/` dùng tài khoản quản trị trong `.env`; không
dùng tài khoản đó để đăng nhập ứng dụng.

## 5. Smoke test không cần token

```powershell
.\scripts\Test-Foundation.ps1
.\scripts\Test-Application.ps1
.\scripts\Test-OIDCLogin.ps1
```

Smoke test OIDC thực hiện Authorization Code + PKCE bằng user local, đổi code lấy access token và gọi
`/api/v1/me`. Script không in credential, code hoặc token ra log.

## 6. Build và test độc lập

Backend:

```powershell
docker run --rm -v "${PWD}/backend:/src" -w /src golang:1.26.6-alpine `
  sh -c "gofmt -w cmd internal && go vet ./... && go test ./..."
```

Frontend build:

```powershell
docker run --rm -e NG_CLI_ANALYTICS=false -v "${PWD}:/workspace" `
  -w /workspace/frontend node:24.15.0-alpine npm run build
```

Frontend test nên chạy trong Docker build layer trên Windows để Vitest worker không bị chậm bởi bind
mount:

```powershell
docker build --target test -t dx-os-frontend-test ./frontend
```

## 7. Dừng

Giữ dữ liệu:

```powershell
docker compose --profile foundation --profile application stop
```

Không dùng `down -v` nếu muốn giữ PostgreSQL.
