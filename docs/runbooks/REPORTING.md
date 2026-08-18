# Hướng dẫn vận hành báo cáo và Metabase

## Mục tiêu

Step 9 cung cấp hai lớp báo cáo dùng chung một nguồn dữ liệu:

1. Trang `Báo cáo` trong Angular phục vụ finance, auditor và dx_admin.
2. Metabase phục vụ phân tích ad-hoc và xây dashboard nâng cao.

Go API không đọc trực tiếp các bảng nghiệp vụ để tổng hợp KPI. Mọi truy vấn báo cáo đi qua schema
`reporting`, nhờ đó định nghĩa số liệu được tập trung và có thể đối soát.

## Kiến trúc

```text
purchase_requests / process_events / attachments / budgets
                         |
                  migration 000007
                         |
              schema reporting (curated views)
                    /                 \
       Go API /api/v1/reports       Metabase
                 |                      |
          Angular dashboard       BI / ad-hoc query
```

Metabase dùng hai kết nối PostgreSQL tách biệt:

- `metabase`: application database của chính Metabase, lưu user, dashboard và cấu hình.
- `dxos`: data warehouse nguồn, kết nối bằng `dxos_report_reader` chỉ có quyền đọc schema
  `reporting`.

Không dùng `dxos_app`, `postgres` hoặc tài khoản có quyền ghi để cấu hình data source cho
Metabase.

## Các view báo cáo đã chuẩn hóa

Migration `000007_reporting_curated_views.sql` tạo:

| View | Mục đích |
|---|---|
| `reporting.purchase_request_facts` | Một dòng cho mỗi phiếu; trạng thái, tiền, SLA, lead time, return và attachment compliance. |
| `reporting.daily_procurement_metrics` | Khối lượng, số đã duyệt, số quá SLA và giá trị theo ngày/tiền tệ/phòng ban. |
| `reporting.budget_utilization` | Allocation đang hoạt động, reserved, committed, available và tỷ lệ sử dụng. |

`reporting.sla_policies` chứa SLA theo tổ chức. Dữ liệu demo dùng ngưỡng mặc định 72 giờ. SLA của
phiếu được tính tới thời điểm hoàn thành (`APPROVED`, `REJECTED`, `CANCELLED`) hoặc thời điểm hiện
tại nếu vẫn đang xử lý.

Các view là live view, không phải materialized view. Sau khi giao dịch commit, dashboard đọc được
dữ liệu mới mà không cần job refresh.

## KPI và quy tắc tính

| KPI | Quy tắc |
|---|---|
| Tổng số phiếu | Số phiếu có `created_date` trong khoảng lọc. |
| Tỷ lệ phê duyệt | `approved_count / total_requests * 100`. |
| Lead time trung bình | Trung bình số giờ từ lúc tạo đến lúc kết thúc của các phiếu đã hoàn tất; phiếu đang mở không tham gia mẫu số. |
| Quá SLA | Lead time vượt `target_hours`. |
| Yêu cầu chỉnh sửa | Phiếu từng có sự kiện `CHANGES_REQUESTED`. |
| Attachment compliance | Số phiếu yêu cầu tài liệu và đã có đúng loại tài liệu / tổng số phiếu yêu cầu tài liệu. |
| Budget utilization | `(reserved + committed) / allocated * 100`. |

Giá trị tiền luôn được nhóm theo `currency`; không cộng VND và ngoại tệ thành một tổng chung.

## Phân quyền

| Vai trò | Angular `/reports` | API báo cáo | Phạm vi |
|---|---:|---:|---|
| `finance` | Có | Có | Tổ chức của user |
| `auditor` | Có | Có | Toàn bộ dữ liệu báo cáo |
| `dx_admin` | Có | Có | Toàn bộ dữ liệu báo cáo |
| `employee` | Không | 403 | Không có |
| `department_manager` | Không | 403 | Không có |
| `ai_operator` | Không | 403 | Không có |

Phân quyền được kiểm tra ở Go API. Việc ẩn menu Angular chỉ cải thiện trải nghiệm, không phải hàng
rào bảo mật.

## Khởi động

Điền các biến `REPORTING_DB_*`, `METABASE_DB_*` và `METABASE_ENCRYPTION_SECRET_KEY` trong `.env`,
sau đó chạy:

```powershell
docker compose --profile foundation --profile application --profile reporting config
docker compose --profile foundation --profile application --profile reporting up -d --build
docker compose --profile foundation --profile application --profile reporting ps
```

Các địa chỉ:

- Angular: `http://localhost:4200/reports`
- Go report API: `http://localhost:8081/api/v1/reports/procurement`
- Metabase: `http://localhost:3000`

`reporting-bootstrap` chạy sau migration, tạo/cập nhật mật khẩu role `dxos_report_reader`, thu hồi
quyền không cần thiết và chỉ cấp `USAGE`/`SELECT` trên schema `reporting`.

## Thiết lập Metabase lần đầu

Sau khi container healthy, chạy script provision idempotent:

```powershell
.\scripts\Initialize-Metabase.ps1
```

Script thực hiện:

1. Tạo admin Metabase local nếu instance chưa được setup.
2. Lưu thông tin đăng nhập trong `data/runtime/metabase-admin.txt` bị Git bỏ qua.
3. Kết nối PostgreSQL bằng `REPORTING_DB_USER`/`REPORTING_DB_PASSWORD`.
4. Giới hạn data source chỉ còn schema `reporting`.
5. Tạo collection `DX-OS Procurement`.
6. Tạo dashboard `DX-OS - Procurement Overview` gồm 8 cards và filter từ ngày, đến ngày,
   tiền tệ.

Chạy lại script không tạo trùng database, collection, card hoặc dashboard. Nếu muốn thiết lập thủ
công, dùng host `postgres`, port `5432`, database `DXOS_DB`, tài khoản `REPORTING_DB_USER` và chỉ
chọn schema `reporting`.

Không đưa mật khẩu báo cáo vào source code, ảnh chụp hoặc tài liệu. Đổi mật khẩu trong `.env` rồi
chạy lại `reporting-bootstrap` nếu cần rotate.

## API

```http
GET /api/v1/reports/procurement?from=2026-07-01&to=2026-07-31&currency=VND
Authorization: Bearer <access-token>
```

Bộ lọc hỗ trợ:

- `from`, `to`: `YYYY-MM-DD`; mặc định 30 ngày gần nhất; tối đa 367 ngày.
- `departmentId`: UUID.
- `costCenter`: chữ hoa, số, `.`, `_`, `-`.
- `currency`: ba chữ cái.

API trả `422` nếu bộ lọc sai hoặc có query parameter không hỗ trợ, `403` nếu role không được phép.
Số tiền và tỷ lệ decimal được trả dưới dạng string để tránh sai số số thực.

## Kiểm thử và đối soát

```powershell
.\scripts\Test-Reporting.ps1
```

Smoke test kiểm tra:

- finance và auditor đọc được báo cáo;
- employee nhận HTTP 403;
- filter ngày/currency được phản ánh trong response;
- tổng số phiếu API bằng số dòng trong curated view khi truy cập qua role read-only;
- role báo cáo không thể tạo bảng;
- health endpoint Metabase trả trạng thái thành công;
- data source chỉ thấy schema `reporting`;
- dashboard có đúng 8 cards, 3 filter và mọi card thực thi thành công.

Kiểm tra thủ công quyền:

```powershell
docker compose --profile foundation --profile application --profile reporting run --rm --no-deps --entrypoint /bin/sh reporting-bootstrap `
  -ec 'PGPASSWORD="$REPORTING_DB_PASSWORD" psql -h postgres -U "$REPORTING_DB_USER" -d "$DXOS_DB" -c "\dp reporting.*"'
```

## Vận hành

- Backup database `metabase` cùng PostgreSQL; volume/container Metabase không thay thế backup.
- Theo dõi log bằng
  `docker compose --profile reporting logs --tail 200 metabase reporting-bootstrap`.
- Khi số liệu lệch, so sánh filter, timezone và currency trước; sau đó query trực tiếp curated view.
- Nếu dữ liệu lớn làm dashboard chậm, đo query plan và index trước khi chuyển sang materialized
  view. Materialized view bắt buộc có lịch refresh, monitoring và chỉ báo `last refreshed`.
- Không public Metabase trực tiếp ra Internet. Production cần HTTPS, reverse proxy, SSO và chính
  sách session trước khi cấp quyền người dùng.

## Khôi phục phiên bản

1. Dừng Metabase: `docker compose --profile reporting stop metabase`.
2. UI và API nghiệp vụ vẫn hoạt động; chỉ route báo cáo/BI bị ảnh hưởng.
3. Nếu migration báo cáo cần rollback, giữ nguyên các bảng nghiệp vụ và drop schema `reporting`
   trong migration rollback có kiểm soát. Không xóa database `dxos`.
