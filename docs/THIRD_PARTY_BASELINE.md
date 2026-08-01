# Baseline công nghệ

Đối chiếu ngày **2026-07-29**. Baseline dùng để bắt đầu và phải được khóa trong lockfile/image tag.

| Thành phần | Baseline | License/chế độ | Ghi chú |
|---|---:|---|---|
| Go | 1.26.5 | BSD-3-Clause | Go chưa cài trên máy hiện tại |
| Angular Core | 22.0.8 | MIT | npm stable tại thời điểm kiểm tra |
| Angular CLI | 22.0.9 | MIT | dùng cùng major 22 |
| Spartan UI Brain/CLI | 1.1.2 | MIT | Angular primitives + copy-owned helm styles |
| Angular CDK | 22.0.6 | MIT | overlay và accessibility primitives |
| Tailwind CSS | 4.3.3 | MIT | styling/design tokens |
| Node.js | 24.15+ | theo Node.js | máy hiện tại 24.14.0, cần nâng |
| TypeScript | 6.0.x | Apache-2.0 | Angular 22 yêu cầu >=6.0 và <6.1 |
| PostgreSQL | 18.4 | PostgreSQL License | image Alpine đã kiểm tra tồn tại |
| Keycloak | 26.7.0 | Apache-2.0 | image Quay đã kiểm tra tồn tại |
| Nextcloud Server | 34.0.2 | AGPL-3.0-or-later | image Apache cho lab |
| Metabase OSS | 0.63.1 | AGPL | không dùng enterprise image |
| RAGFlow | 0.26.4 | Apache-2.0 | dùng compose upstream đúng tag |

## Nguồn chính thức

- Go releases: <https://go.dev/doc/devel/release>
- Go downloads: <https://go.dev/dl/>
- Angular compatibility: <https://angular.dev/reference/versions>
- Angular release policy: <https://angular.dev/reference/releases>
- Spartan UI installation: <https://www.spartan.ng/documentation/installation>
- Spartan repository: <https://github.com/spartan-ng/spartan>
- PostgreSQL versioning: <https://www.postgresql.org/support/versioning/>
- Keycloak releases: <https://www.keycloak.org/2026/07/keycloak-2670-released>
- Nextcloud repository: <https://github.com/nextcloud/server>
- Nextcloud OIDC: <https://docs.nextcloud.com/server/latest/admin_manual/configuration_user/user_auth_oidc.html>
- Metabase license: <https://github.com/metabase/metabase/blob/master/LICENSE.txt>
- RAGFlow repository: <https://github.com/infiniflow/ragflow>

## Quy tắc nâng cấp

1. Không dùng `latest` cho Demo/UAT/Pilot.
2. Dependency Go/Angular phải có lock/checksum.
3. Đọc migration/security notes.
4. Backup trước platform/database upgrade.
5. Chạy smoke test SSO, workflow, file, dashboard và AI.
6. Cập nhật baseline, SBOM và test evidence cùng thay đổi.
