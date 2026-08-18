# ADR-006: Angular SPA dùng OIDC Authorization Code + PKCE

## Trạng thái

Accepted cho MVP

## Bối cảnh

Angular cần SSO trực tiếp với Keycloak; Go API cần xác thực stateless. BFF an toàn hơn về việc token
không xuất hiện trong browser nhưng tăng session/cookie/CSRF và hạ tầng.

## Quyết định

Angular là public client dùng Authorization Code + PKCE S256. Access token ngắn hạn lưu memory
(session storage chỉ khi thư viện yêu cầu và đã review). Go xác minh JWT/JWKS và thực thi authorization.

## Đánh đổi

- Chuẩn, đơn giản cho SPA/API tách rời.
- Token vẫn tồn tại trong browser và chịu rủi ro XSS.
- Cần CSP, dependency hygiene và không dùng localStorage dài hạn.

## Điều kiện xem xét lại

- Production có dữ liệu nhạy cảm cao.
- Có yêu cầu refresh/session dài hoặc browser policy nghiêm ngặt.
- Threat model yêu cầu token không xuất hiện ở frontend; khi đó chuyển BFF.

