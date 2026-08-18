# Hướng dẫn vận hành: Hạ tầng nền tảng

## 1. Kiểm tra điều kiện

```powershell
docker version
docker compose version
```

Lệnh `docker version` phải hiển thị cả Client và Server. Nếu chỉ có Client, mở Docker Desktop và
đợi Engine chuyển sang trạng thái Running.

## 2. Tạo cấu hình local

```powershell
Copy-Item .env.example .env
```

Thay tất cả `CHANGEME` bằng chuỗi ngẫu nhiên riêng. Không gửi `.env` qua chat/email và không commit.

## 3. Validate Compose

```powershell
docker compose --profile foundation config --quiet
```

Nếu port 5432 hoặc 8080 đã dùng, đổi `POSTGRES_HOST_PORT` hoặc `KEYCLOAK_HOST_PORT` trong `.env`.

## 4. Khởi động

```powershell
docker compose --profile foundation pull
docker compose --profile foundation up -d
docker compose --profile foundation ps
docker compose --profile foundation logs --tail 100 postgres keycloak
```

Mở `http://localhost:8080`, đăng nhập bằng `KEYCLOAK_ADMIN_USER` và
`KEYCLOAK_ADMIN_PASSWORD`. Realm `dx-os` phải xuất hiện với các role đã định nghĩa.

## 5. Smoke test

```powershell
.\scripts\Test-Foundation.ps1
```

Script kiểm tra:

- Compose hợp lệ;
- bốn database và bốn login role dịch vụ đã được tạo;
- OIDC discovery của realm `dx-os` hoạt động và hỗ trợ PKCE S256;
- sáu role nghiệp vụ tồn tại;
- `dx-web` dùng Authorization Code + PKCE, không bật Direct Access Grants;
- `dx-api` là bearer-only và audience mapper trỏ tới `dx-api`.

## 6. Dừng an toàn

```powershell
docker compose --profile foundation stop
```

Không dùng `down -v` trong môi trường có dữ liệu cần giữ; tùy chọn `-v` xóa volume database.

## Lưu ý init database

Script trong `data/postgres/init` chỉ chạy khi volume PostgreSQL được tạo lần đầu. Thay mật khẩu
trong `.env` sau đó không tự đổi mật khẩu đã lưu trong PostgreSQL. Việc rotate secret cần runbook
riêng và lệnh `ALTER ROLE`.

Từ PostgreSQL 18, volume phải mount tại `/var/lib/postgresql`; dữ liệu thật nằm trong thư mục con
theo major version. Không đổi lại mount thành `/var/lib/postgresql/data`.
