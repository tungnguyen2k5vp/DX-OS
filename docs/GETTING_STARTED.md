---
id: getting-started
title: Bắt đầu với DX-OS
description: Cài đặt, khởi chạy và xác nhận DX-OS Lab trên máy local.
slug: /bat-dau
sidebar_position: 1
---

# Bắt đầu với DX-OS

Trang này đưa một máy mới từ source code đến trạng thái đăng nhập và sử dụng được toàn bộ DX-OS
Lab.

## Bạn sẽ dựng những gì?

| Thành phần | Vai trò                      | URL local             |
| ---------- | ---------------------------- | --------------------- |
| Angular    | Ứng dụng nghiệp vụ           | http://localhost:4200 |
| Go API     | REST API và authorization    | http://localhost:8081 |
| Keycloak   | Đăng nhập và realm role      | http://localhost:8080 |
| PostgreSQL | Dữ liệu nghiệp vụ và dịch vụ | 127.0.0.1:5432        |
| Nextcloud  | Kho tệp nội bộ               | http://localhost:8082 |
| Metabase   | BI và dashboard nâng cao     | http://localhost:3000 |
| Docusaurus | Website tài liệu này         | http://localhost:4300 |

## 1. Yêu cầu máy

- Git.
- Docker Desktop hoặc Docker Engine đang chạy.
- Docker Compose 2.26 trở lên.
- PowerShell 7 được khuyến nghị.

Kiểm tra:

```powershell
git --version
docker version
docker compose version
$PSVersionTable.PSVersion
```

Nếu Docker chỉ hiện Client mà không có Server, hãy khởi động Docker Desktop trước khi tiếp tục.

## 2. Tạo file môi trường

Từ thư mục dx-os-lab:

```powershell
Copy-Item .env.example .env
```

Mở .env và thay toàn bộ giá trị bắt đầu bằng CHANGEME bằng mật khẩu ngẫu nhiên riêng.

```powershell
Select-String -Path .env -Pattern 'CHANGEME'
```

Lệnh trên phải không trả về dòng nào. Không commit .env hoặc các file trong data/runtime.

## 3. Dựng application stack

```powershell
docker compose --profile foundation --profile application --profile reporting config --quiet
docker compose --profile foundation --profile application --profile reporting up -d --build
docker compose --profile foundation --profile application --profile reporting ps
```

Lần đầu cần thời gian tải image và bootstrap Nextcloud/Metabase. Chờ postgres, nextcloud, api, web
và metabase ở trạng thái running/healthy.

Nếu service chưa sẵn sàng:

```powershell
docker compose --profile foundation --profile application --profile reporting logs --tail 200
```

## 4. Tạo sáu tài khoản demo

```powershell
.\scripts\Initialize-DevUser.ps1 -Username employee.demo -Role employee -CredentialsPath data\runtime\employee-demo.txt
.\scripts\Initialize-DevUser.ps1 -Username manager.demo -Role department_manager -CredentialsPath data\runtime\manager-demo.txt
.\scripts\Initialize-DevUser.ps1 -Username finance.demo -Role finance -CredentialsPath data\runtime\finance-demo.txt
.\scripts\Initialize-DevUser.ps1 -Username auditor.demo -Role auditor -CredentialsPath data\runtime\auditor-demo.txt
.\scripts\Initialize-DevUser.ps1 -Username ai.operator.demo -Role ai_operator -CredentialsPath data\runtime\ai-operator-demo.txt
.\scripts\Initialize-DevUser.ps1 -Username admin.demo -Role dx_admin -CredentialsPath data\runtime\admin-demo.txt
```

Credential được sinh ngẫu nhiên trong data/runtime. Chạy lại script sẽ đổi password của user tương
ứng.

## 5. Provision Metabase

```powershell
.\scripts\Initialize-Metabase.ps1
```

Script tạo data source read-only, collection, 8 card và dashboard có bộ lọc ngày/currency. Tài
khoản Metabase nằm trong data/runtime/metabase-admin.txt và độc lập với Keycloak.

## 6. Dựng website tài liệu

```powershell
docker compose --profile documentation up -d --build docs
```

Mở http://localhost:4300. Website đọc trực tiếp nội dung từ thư mục docs, vì vậy không có bản sao
tài liệu thứ hai cần đồng bộ.

Nếu muốn chạy dev server:

```powershell
Set-Location docs-site
npm ci
npm start -- --port 4300
```

## 7. Chạy smoke test

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

Sau khi các script đều pass, mở http://localhost:4200 và đăng nhập bằng employee.demo.

## 8. Kiểm tra website tài liệu khi sửa nội dung

```powershell
Set-Location docs-site
npm run typecheck
npm run build
```

Production build sẽ thất bại nếu có internal link hoặc Markdown link bị hỏng.

## Bước tiếp theo

- Người dùng nghiệp vụ: [Hướng dẫn sử dụng](USER_GUIDE.md).
- Developer: [Backend Go](implementation/BACKEND_GO.md) và
  [Frontend Angular](implementation/FRONTEND_ANGULAR.md).
- DevOps: [Triển khai](implementation/DEPLOYMENT.md) và
  [Vận hành](implementation/OPERATIONS.md).
- QA: [Chiến lược kiểm thử](implementation/TESTING.md).
