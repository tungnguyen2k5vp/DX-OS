# ADR-003: Một PostgreSQL cluster, nhiều database trong lab

## Trạng thái

Accepted cho Local/Dev; không tự động áp dụng cho Production

## Bối cảnh

MVP có nhiều dịch vụ dùng PostgreSQL nhưng máy local chỉ có khoảng 16 GB RAM. Tách một container
PostgreSQL cho từng sản phẩm làm tăng chi phí tài nguyên và vận hành.

## Quyết định

Dùng một PostgreSQL 18 cluster cho Local/Dev, tạo database và login role riêng cho Go API,
Keycloak, Nextcloud và Metabase. Không cấp quyền chéo database.

## Đánh đổi

- Giảm RAM và số container.
- Một lỗi cluster ảnh hưởng nhiều dịch vụ.
- Backup/restore và nâng cấp cần tách logic theo database.

## Biện pháp giảm thiểu

- Backup từng database và kiểm thử restore.
- Không cho ứng dụng dùng superuser.
- Tách cluster khi pilot có yêu cầu HA, blast-radius hoặc lịch nâng cấp khác nhau.
