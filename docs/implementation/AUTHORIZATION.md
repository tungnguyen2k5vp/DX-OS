# Authentication và Authorization

## 1. Mô hình

- Keycloak là Identity Provider.
- Angular là public OIDC client dùng Authorization Code + PKCE.
- Go API là OAuth2 Resource Server.
- Go không phát access token và không lưu password.
- Keycloak role cung cấp coarse-grained permission; Go kiểm ownership, department scope và trạng thái.

## 2. Realm và clients

Realm: `dx-os`.

### `dx-web`

- public client;
- standard flow bật;
- PKCE S256 bắt buộc;
- direct access grant tắt;
- redirect URI chính xác theo từng môi trường;
- web origin allowlist, không wildcard production.

### `dx-api`

- audience cho access token;
- không cần browser redirect;
- service account chỉ bật nếu có machine-to-machine use case.

### Platform clients

Nextcloud, Metabase hoặc RAGFlow dùng client riêng nếu tích hợp OIDC. Không dùng chung client secret.

## 3. Role

| Role                 | Mục đích                                                   |
| -------------------- | ---------------------------------------------------------- |
| `employee`           | tạo, sửa và xem phiếu theo scope                           |
| `department_manager` | xử lý bước trưởng bộ phận                                  |
| `finance`            | kiểm ngân sách và duyệt cuối                               |
| `auditor`            | đọc audit/evidence                                         |
| `ai_operator`        | xem và xử lý recommendation theo policy                    |
| `dx_admin`           | cấu hình ứng dụng, không mặc nhiên đọc mọi dữ liệu hạn chế |

Role không tạo hierarchy ngầm quá rộng. `dx_admin` không được dùng như superuser cho mọi thao tác.

## 4. Authorization matrix

| Hành động             |           Employee |           Manager |     Finance | Auditor |        AI operator |          Admin |
| --------------------- | -----------------: | ----------------: | ----------: | ------: | -----------------: | -------------: |
| Tạo phiếu             |                  ✓ |                 ✓ |           — |       — |                  — |              — |
| Sửa draft của mình    |                  ✓ |                 ✓ |           — |       — |                  — |              — |
| Submit phiếu của mình |                  ✓ |                 ✓ |           — |       — |                  — |              — |
| Manager approve       |                  — | ✓ đúng department |           — |       — |                  — | không mặc định |
| Finance approve       |                  — |                 — |           ✓ |       — |                  — | không mặc định |
| Xem audit             | của hồ sơ được xem |         của scope |   của scope |       ✓ |       liên quan AI |       cấu hình |
| Approve AI action     |        theo policy |       theo policy | theo policy |       — |          điều phối | không mặc định |
| Execute tool          |                  — |                 — |           — |       — | qua backend policy |              — |

Đây là ma trận của implementation hiện tại. Nếu sau này có thao tác support-only, thao tác đó phải
có endpoint/quy trình quản trị riêng và audit; không cho admin giả danh người nghiệp vụ qua API
thông thường.

## 5. Data scope

Backend tính scope từ authenticated user profile:

- Employee: phiếu của mình hoặc được giao/xem.
- Manager: phiếu thuộc department được quản lý.
- Finance: phiếu ở bước tài chính, theo organization/cost center được cấp.
- Auditor: read-only theo mandate.

Không tin `departmentId`, `requesterId` hoặc role do Angular gửi.

## 6. Token validation tại Go

Kiểm:

- chữ ký bằng JWKS;
- thuật toán allowlist;
- `iss` đúng issuer;
- `aud` chứa DX API audience;
- `exp`, `nbf`, clock skew nhỏ;
- token type/scope phù hợp;
- role claim có cấu trúc mong đợi.

JWKS được cache và refresh khi `kid` mới. Khi Keycloak tạm mất, token đã ký vẫn xác minh được trong
thời gian cache hợp lệ; fail closed nếu không thể xác minh.

## 7. Token ở Angular

- Không lưu access/refresh token dài hạn trong `localStorage`.
- Ưu tiên memory; `sessionStorage` chỉ nếu thư viện OIDC yêu cầu và đã đánh giá XSS.
- Access token ngắn hạn.
- Logout gọi end-session của Keycloak và xóa state local.
- Không đưa token vào URL, log, analytics hoặc error report.

Nếu production có yêu cầu bảo mật cao, chuyển sang BFF + secure httpOnly cookie bằng ADR riêng.

## 8. Policy enforcement

Mỗi transition đi qua:

```text
authenticated?
-> role phù hợp?
-> resource thuộc scope?
-> actor khác requester khi approve?
-> state hiện tại cho phép?
-> business precondition đạt?
-> optimistic version đúng?
-> execute + audit
```

Denied action cũng ghi security audit ở mức phù hợp, nhưng không ghi toàn bộ request body nhạy cảm.

## 9. Service account và secret

- Mỗi integration có service account riêng.
- Scope tối thiểu và token ngắn hạn.
- Secret nằm trong secret manager hoặc environment injection, không commit.
- Rotation có owner, lịch và runbook.
- Không dùng Keycloak admin account cho runtime integration.

## 10. Threat checklist

- XSS lấy token: CSP, output escaping, dependency scan, không localStorage.
- CSRF: PKCE/state/nonce; nếu dùng cookie thì SameSite + CSRF token.
- Broken access control: negative test cho mọi role/scope.
- Replay: idempotency key, token expiry và one-time approval execution.
- Prompt injection: RAG content là dữ liệu, không phải instruction; tool policy ở Go.
- SSRF: allowlist host cho Nextcloud/RAGFlow/tool.
- File upload: size/type/checksum/quarantine và không thực thi file.
