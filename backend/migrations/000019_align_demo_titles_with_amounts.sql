-- Align normalized demo titles with their monetary scale while preserving every
-- amount, status, identifier and relationship created by the original workflow.

CREATE TEMP TABLE demo_amount_alignment ON COMMIT DROP AS
WITH normalized_ids AS (
    SELECT DISTINCT resource_id AS id
    FROM audit_logs
    WHERE correlation_id = 'migration-000018'
      AND resource_type = 'purchase_request'
), ranked AS (
    SELECT
        pr.id,
        pr.request_code,
        pr.title AS old_title,
        pr.status,
        pr.total_amount,
        CASE
            WHEN pr.total_amount >= 50000000000 THEN 'STRATEGIC'
            WHEN pr.total_amount >= 100000000 THEN 'ENTERPRISE'
            WHEN pr.total_amount >= 20000000 THEN 'EQUIPMENT'
            WHEN pr.total_amount >= 5000000 THEN 'MEDIUM'
            WHEN pr.total_amount >= 1000000 THEN 'SMALL'
            ELSE 'MICRO'
        END AS amount_band,
        row_number() OVER (
            PARTITION BY CASE
                WHEN pr.total_amount >= 50000000000 THEN 'STRATEGIC'
                WHEN pr.total_amount >= 100000000 THEN 'ENTERPRISE'
                WHEN pr.total_amount >= 20000000 THEN 'EQUIPMENT'
                WHEN pr.total_amount >= 5000000 THEN 'MEDIUM'
                WHEN pr.total_amount >= 1000000 THEN 'SMALL'
                ELSE 'MICRO'
            END
            ORDER BY pr.created_at, pr.id
        ) AS band_no
    FROM purchase_requests pr
    JOIN normalized_ids n ON n.id = pr.id
)
SELECT
    *,
    CASE amount_band
        WHEN 'STRATEGIC' THEN CASE band_no
            WHEN 1 THEN 'Xây dựng trung tâm dữ liệu dự phòng'
            WHEN 2 THEN 'Triển khai hệ thống hoạch định nguồn lực doanh nghiệp'
            WHEN 3 THEN 'Nâng cấp hạ tầng mạng liên chi nhánh'
            WHEN 4 THEN 'Đầu tư nền tảng lưu trữ và sao lưu tập trung'
            WHEN 5 THEN 'Triển khai trung tâm giám sát an toàn thông tin'
            WHEN 6 THEN 'Nâng cấp cụm máy chủ ứng dụng'
            WHEN 7 THEN 'Xây dựng hệ thống khôi phục sau sự cố'
            WHEN 8 THEN 'Hiện đại hóa hạ tầng hội nghị toàn công ty'
            WHEN 9 THEN 'Đầu tư thiết bị đầu cuối cho các chi nhánh'
            WHEN 10 THEN 'Triển khai nền tảng quản trị dữ liệu doanh nghiệp'
            ELSE 'Xây dựng hạ tầng điện toán đám mây riêng'
        END
        WHEN 'ENTERPRISE' THEN 'Trang bị hệ thống trình chiếu cho hội trường lớn'
        WHEN 'EQUIPMENT' THEN CASE band_no
            WHEN 1 THEN 'Bổ sung laptop cho nhóm triển khai miền Bắc'
            WHEN 2 THEN 'Nâng cấp máy trạm cho nhóm phân tích dữ liệu'
            WHEN 3 THEN 'Trang bị màn hình cho bộ phận thiết kế'
            WHEN 4 THEN 'Mua thiết bị mạng cho văn phòng tầng 3'
            WHEN 5 THEN 'Bổ sung ghế công thái học cho khu làm việc'
            WHEN 6 THEN 'Trang bị thiết bị hội nghị cho phòng họp lớn'
            WHEN 7 THEN 'Gia hạn bản quyền phần mềm quản lý dự án'
            WHEN 8 THEN 'Mua bộ lưu điện cho phòng máy chủ'
            WHEN 9 THEN 'Trang bị máy tính cho nhân viên mới'
            WHEN 10 THEN 'Mua thiết bị bảo mật cho mạng nội bộ'
            WHEN 11 THEN 'Bổ sung máy in cho bộ phận hành chính'
            WHEN 12 THEN 'Trang bị máy chiếu cho phòng đào tạo'
            WHEN 13 THEN 'Mua bộ thiết bị kiểm kê tài sản'
            WHEN 14 THEN 'Bổ sung thiết bị lưu trữ cho phòng kỹ thuật'
            WHEN 15 THEN 'Trang bị điện thoại công vụ cho nhóm kinh doanh'
            WHEN 16 THEN 'Mua thiết bị chấm công cho văn phòng mới'
            WHEN 17 THEN 'Bổ sung máy quét tài liệu cho phòng kế toán'
            WHEN 18 THEN 'Trang bị camera cho khu vực kho hàng'
            WHEN 19 THEN 'Mua tủ mạng và phụ kiện lắp đặt'
            ELSE 'Bổ sung thiết bị trình chiếu di động'
        END
        WHEN 'MEDIUM' THEN 'Mua máy in đa chức năng cho văn phòng'
        WHEN 'SMALL' THEN CASE band_no
            WHEN 1 THEN 'Bổ sung ổ cứng sao lưu cho bộ phận kế toán'
            WHEN 2 THEN 'Mua webcam cho nhóm làm việc từ xa'
            WHEN 3 THEN 'Trang bị bộ bàn phím và chuột không dây'
            WHEN 4 THEN 'Mua tai nghe cho bộ phận chăm sóc khách hàng'
            WHEN 5 THEN 'Bổ sung bộ phát Wi-Fi cho phòng họp'
            WHEN 6 THEN 'Mua tủ hồ sơ cho phòng tài chính'
            WHEN 7 THEN 'Gia hạn chứng thư số doanh nghiệp'
            WHEN 8 THEN 'Trang bị bảng viết cho phòng đào tạo'
            WHEN 9 THEN 'Mua bộ dụng cụ bảo trì thiết bị văn phòng'
            WHEN 10 THEN 'Bổ sung vật tư in ấn cho phòng hành chính'
            WHEN 11 THEN 'Mua bộ chuyển đổi trình chiếu cho phòng họp'
            ELSE 'Trang bị chân đế màn hình cho khu làm việc'
        END
        ELSE CASE band_no
            WHEN 1 THEN 'Mua tem nhãn cho hồ sơ lưu trữ'
            WHEN 2 THEN 'Bổ sung bút ký cho quầy tiếp nhận'
            WHEN 3 THEN 'Mua bìa hồ sơ cho phòng hành chính'
            WHEN 4 THEN 'In nhãn tài sản phục vụ kiểm kê'
            ELSE 'Mua sổ bàn giao thiết bị văn phòng'
        END
    END AS new_title,
    CASE amount_band
        WHEN 'STRATEGIC' THEN CASE band_no
            WHEN 1 THEN 'Gói triển khai trung tâm dữ liệu dự phòng'
            WHEN 2 THEN 'Gói triển khai hệ thống ERP doanh nghiệp'
            WHEN 3 THEN 'Gói nâng cấp mạng WAN liên chi nhánh'
            WHEN 4 THEN 'Gói lưu trữ và sao lưu dữ liệu tập trung'
            WHEN 5 THEN 'Gói triển khai trung tâm giám sát an toàn thông tin'
            WHEN 6 THEN 'Gói máy chủ và bản quyền nền tảng ứng dụng'
            WHEN 7 THEN 'Gói giải pháp khôi phục hoạt động sau sự cố'
            WHEN 8 THEN 'Gói thiết bị hội nghị cho toàn công ty'
            WHEN 9 THEN 'Gói thiết bị đầu cuối cho hệ thống chi nhánh'
            WHEN 10 THEN 'Gói nền tảng quản trị dữ liệu doanh nghiệp'
            ELSE 'Gói hạ tầng điện toán đám mây riêng'
        END
        WHEN 'ENTERPRISE' THEN 'Hệ thống màn hình LED và điều khiển trình chiếu'
        WHEN 'EQUIPMENT' THEN CASE band_no
            WHEN 1 THEN 'Laptop Dell Latitude 5450'
            WHEN 2 THEN 'Máy trạm HP Z2 Tower'
            WHEN 3 THEN 'Màn hình Dell UltraSharp 27 inch'
            WHEN 4 THEN 'Bộ thiết bị mạng doanh nghiệp'
            WHEN 5 THEN 'Ghế công thái học Ergo Pro'
            WHEN 6 THEN 'Bộ thiết bị hội nghị trực tuyến'
            WHEN 7 THEN 'Gói bản quyền phần mềm quản lý dự án 12 tháng'
            WHEN 8 THEN 'Bộ lưu điện UPS 3000VA'
            WHEN 9 THEN 'Laptop văn phòng cho nhân viên mới'
            WHEN 10 THEN 'Thiết bị tường lửa doanh nghiệp'
            WHEN 11 THEN 'Máy in đa chức năng Brother'
            WHEN 12 THEN 'Máy chiếu Epson văn phòng'
            WHEN 13 THEN 'Bộ máy quét mã vạch kiểm kê tài sản'
            WHEN 14 THEN 'Thiết bị lưu trữ mạng NAS'
            WHEN 15 THEN 'Điện thoại công vụ Samsung Galaxy A'
            WHEN 16 THEN 'Bộ máy chấm công nhận diện khuôn mặt'
            WHEN 17 THEN 'Máy quét tài liệu tốc độ cao'
            WHEN 18 THEN 'Bộ camera giám sát kho hàng'
            WHEN 19 THEN 'Tủ mạng và phụ kiện lắp đặt'
            ELSE 'Bộ máy chiếu và màn chiếu di động'
        END
        WHEN 'MEDIUM' THEN 'Máy in đa chức năng Brother MFC'
        WHEN 'SMALL' THEN CASE band_no
            WHEN 1 THEN 'Ổ cứng sao lưu dung lượng 4 TB'
            WHEN 2 THEN 'Webcam hội nghị Full HD'
            WHEN 3 THEN 'Bộ bàn phím và chuột không dây'
            WHEN 4 THEN 'Tai nghe có mic cho tổng đài viên'
            WHEN 5 THEN 'Bộ phát Wi-Fi cho phòng họp'
            WHEN 6 THEN 'Tủ hồ sơ chống ẩm'
            WHEN 7 THEN 'Gói chứng thư số doanh nghiệp'
            WHEN 8 THEN 'Bảng viết văn phòng có chân đế'
            WHEN 9 THEN 'Bộ dụng cụ bảo trì thiết bị'
            WHEN 10 THEN 'Mực in và giấy in văn phòng'
            WHEN 11 THEN 'Bộ chuyển đổi trình chiếu USB-C'
            ELSE 'Chân đế màn hình công thái học'
        END
        ELSE CASE band_no
            WHEN 1 THEN 'Tem nhãn hồ sơ lưu trữ'
            WHEN 2 THEN 'Bút ký văn phòng'
            WHEN 3 THEN 'Bìa hồ sơ giấy'
            WHEN 4 THEN 'Nhãn tài sản in theo mẫu'
            ELSE 'Sổ bàn giao thiết bị'
        END
    END AS new_item_description,
    CASE
        WHEN amount_band = 'STRATEGIC' THEN 'dự án'
        WHEN amount_band IN ('ENTERPRISE', 'MEDIUM') THEN 'bộ'
        WHEN amount_band = 'EQUIPMENT' AND band_no IN (4, 6, 8, 13, 16, 18, 19, 20) THEN 'bộ'
        WHEN amount_band = 'EQUIPMENT' AND band_no = 7 THEN 'gói'
        WHEN amount_band = 'SMALL' AND band_no IN (3, 5, 9, 10, 11) THEN 'bộ'
        WHEN amount_band = 'SMALL' AND band_no = 7 THEN 'gói'
        WHEN amount_band = 'MICRO' AND band_no IN (1, 4) THEN 'tờ'
        WHEN amount_band = 'MICRO' AND band_no = 2 THEN 'cây'
        WHEN amount_band = 'MICRO' AND band_no = 3 THEN 'bìa'
        ELSE 'chiếc'
    END AS new_item_unit
FROM ranked;

INSERT INTO audit_logs (
    resource_type, resource_id, organization_id, action, actor_id, actor_roles,
    from_status, to_status, correlation_id, metadata
)
SELECT
    'purchase_request', c.id, organization.id, 'DEMO_AMOUNT_ALIGNED', actor.id, ARRAY['dx_admin'],
    c.status, c.status, 'migration-000019',
    jsonb_build_object(
        'source', 'demo_data_hygiene',
        'requestCode', c.request_code,
        'amountBand', c.amount_band,
        'amount', c.total_amount,
        'oldTitle', c.old_title,
        'newTitle', c.new_title,
        'amountChanged', false,
        'deleted', false
    )
FROM demo_amount_alignment c
CROSS JOIN organizations organization
CROSS JOIN LATERAL (
    SELECT id
    FROM users
    WHERE username IN ('admin.demo', 'finance.demo')
    ORDER BY CASE username WHEN 'admin.demo' THEN 0 ELSE 1 END
    LIMIT 1
) actor
WHERE organization.code = 'DX-OS';

UPDATE outbox_events n
SET body = replace(n.body, c.old_title, c.new_title)
FROM demo_amount_alignment c
WHERE n.resource_id = c.id
  AND position(c.old_title IN n.body) > 0;

UPDATE user_notifications n
SET body = replace(n.body, c.old_title, c.new_title)
FROM demo_amount_alignment c
WHERE n.resource_id = c.id
  AND position(c.old_title IN n.body) > 0;

UPDATE purchase_orders po
SET
    note = replace(po.note, c.old_title, c.new_title),
    version = po.version + 1,
    updated_at = now()
FROM demo_amount_alignment c
WHERE po.purchase_request_id = c.id
  AND position(c.old_title IN coalesce(po.note, '')) > 0;

UPDATE purchase_request_items item
SET
    description = c.new_item_description || CASE
        WHEN item.line_number > 1 THEN ' – hạng mục ' || item.line_number::text
        ELSE ''
    END,
    unit = c.new_item_unit,
    updated_at = now()
FROM demo_amount_alignment c
WHERE item.purchase_request_id = c.id;

UPDATE purchase_requests pr
SET
    title = c.new_title,
    version = pr.version + 1,
    updated_at = now()
FROM demo_amount_alignment c
WHERE pr.id = c.id;

INSERT INTO app_metadata (key, value)
VALUES ('schema_version', '000019_align_demo_titles_with_amounts')
ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value, updated_at = now();
