-- Normalize accumulated smoke-test data into a small, readable demo dataset.
-- Historical identifiers and relationships are preserved; no business row is deleted.

CREATE TEMP TABLE demo_supplier_cleanup ON COMMIT DROP AS
WITH ranked AS (
    SELECT
        s.*,
        row_number() OVER (ORDER BY s.created_at, s.id) AS row_no
    FROM suppliers s
    WHERE s.code LIKE 'WF-%'
       OR s.name ILIKE 'Workflow Supplier %'
       OR s.name ILIKE 'Nhà cung cấp kiểm thử %'
)
SELECT
    id,
    name AS old_name,
    code AS old_code,
    status AS old_status,
    row_no,
    CASE row_no
        WHEN 1 THEN 'NCC-CNTT-ML'
        WHEN 2 THEN 'NCC-CNTT-TC'
        WHEN 3 THEN 'NCC-NOITHAT-AP'
        WHEN 4 THEN 'NCC-PHANMEM-SV'
        WHEN 5 THEN 'NCC-VANPHONG-HH'
        WHEN 6 THEN 'NCC-DICHVU-HT'
        ELSE 'KT-LUUTRU-' || lpad((row_no - 6)::text, 3, '0')
    END AS new_code,
    CASE row_no
        WHEN 1 THEN 'Công ty TNHH Công nghệ Minh Long'
        WHEN 2 THEN 'Công ty Cổ phần Thiết bị Thành Công'
        WHEN 3 THEN 'Công ty TNHH Nội thất An Phát'
        WHEN 4 THEN 'Công ty Cổ phần Giải pháp Số Việt'
        WHEN 5 THEN 'Công ty TNHH Văn phòng Hoàng Hà'
        WHEN 6 THEN 'Công ty Cổ phần Dịch vụ Hưng Thịnh'
        ELSE 'Bản ghi kiểm thử đã lưu trữ ' || lpad((row_no - 6)::text, 2, '0')
    END AS new_name,
    CASE WHEN row_no <= 6 THEN 'ACTIVE' ELSE 'INACTIVE' END AS new_status
FROM ranked;

INSERT INTO audit_logs (
    resource_type, resource_id, organization_id, action, actor_id, actor_roles,
    from_status, to_status, correlation_id, metadata
)
SELECT
    'supplier', c.id, organization.id, 'DEMO_DATA_NORMALIZED', actor.id, ARRAY['dx_admin'],
    c.old_status, c.new_status, 'migration-000018',
    jsonb_build_object(
        'source', 'demo_data_hygiene',
        'oldCode', c.old_code,
        'newCode', c.new_code,
        'oldName', c.old_name,
        'newName', c.new_name,
        'deleted', false
    )
FROM demo_supplier_cleanup c
CROSS JOIN organizations organization
CROSS JOIN LATERAL (
    SELECT id
    FROM users
    WHERE username IN ('admin.demo', 'finance.demo')
    ORDER BY CASE username WHEN 'admin.demo' THEN 0 ELSE 1 END
    LIMIT 1
) actor
WHERE organization.code = 'DX-OS';

UPDATE suppliers s
SET
    code = c.new_code,
    name = c.new_name,
    status = c.new_status,
    tax_code = CASE c.row_no
        WHEN 1 THEN '0109123401'
        WHEN 2 THEN '0109123402'
        WHEN 3 THEN '0109123403'
        WHEN 4 THEN '0109123404'
        WHEN 5 THEN '0109123405'
        WHEN 6 THEN '0109123406'
        ELSE NULL
    END,
    contact_name = CASE c.row_no
        WHEN 1 THEN 'Nguyễn Hoàng Minh'
        WHEN 2 THEN 'Trần Ngọc Lan'
        WHEN 3 THEN 'Lê Anh Tuấn'
        WHEN 4 THEN 'Phạm Thu Hương'
        WHEN 5 THEN 'Vũ Đức Thành'
        WHEN 6 THEN 'Đặng Khánh Linh'
        ELSE NULL
    END,
    email = CASE c.row_no
        WHEN 1 THEN 'kinhdoanh@minhlong.example'
        WHEN 2 THEN 'baogia@thanhcong.example'
        WHEN 3 THEN 'lienhe@anphat.example'
        WHEN 4 THEN 'doanhnghiep@soviet.example'
        WHEN 5 THEN 'donhang@hoangha.example'
        WHEN 6 THEN 'dichvu@hungthinh.example'
        ELSE NULL
    END,
    phone = CASE c.row_no
        WHEN 1 THEN '024 7300 1001'
        WHEN 2 THEN '024 7300 1002'
        WHEN 3 THEN '024 7300 1003'
        WHEN 4 THEN '028 7300 1004'
        WHEN 5 THEN '024 7300 1005'
        WHEN 6 THEN '028 7300 1006'
        ELSE NULL
    END,
    address = CASE c.row_no
        WHEN 1 THEN 'Quận Cầu Giấy, Hà Nội'
        WHEN 2 THEN 'Quận Nam Từ Liêm, Hà Nội'
        WHEN 3 THEN 'Thành phố Thủ Đức, TP. Hồ Chí Minh'
        WHEN 4 THEN 'Quận 3, TP. Hồ Chí Minh'
        WHEN 5 THEN 'Quận Thanh Xuân, Hà Nội'
        WHEN 6 THEN 'Quận Bình Thạnh, TP. Hồ Chí Minh'
        ELSE NULL
    END,
    bank_name = CASE c.row_no
        WHEN 1 THEN 'Ngân hàng TMCP Ngoại thương Việt Nam'
        WHEN 2 THEN 'Ngân hàng TMCP Đầu tư và Phát triển Việt Nam'
        WHEN 3 THEN 'Ngân hàng TMCP Kỹ thương Việt Nam'
        WHEN 4 THEN 'Ngân hàng TMCP Á Châu'
        WHEN 5 THEN 'Ngân hàng TMCP Quân đội'
        WHEN 6 THEN 'Ngân hàng TMCP Tiên Phong'
        ELSE NULL
    END,
    bank_account_number = NULL,
    contract_reference = CASE WHEN c.row_no <= 6
        THEN 'HĐ-NCC-2026-' || lpad(c.row_no::text, 2, '0')
        ELSE NULL
    END,
    contract_expires_on = CASE WHEN c.row_no <= 6 THEN DATE '2027-12-31' ELSE NULL END,
    compliance_status = CASE
        WHEN c.row_no <= 5 THEN 'VERIFIED'
        WHEN c.row_no = 6 THEN 'PENDING'
        ELSE 'EXPIRED'
    END,
    performance_score = CASE c.row_no
        WHEN 1 THEN 93
        WHEN 2 THEN 88
        WHEN 3 THEN 90
        WHEN 4 THEN 95
        WHEN 5 THEN 84
        WHEN 6 THEN 78
        ELSE NULL
    END,
    risk_level = CASE WHEN c.row_no IN (5, 6) THEN 'MEDIUM' ELSE 'LOW' END,
    business_note = CASE WHEN c.row_no <= 6
        THEN 'Hồ sơ nhà cung cấp mẫu phục vụ trình diễn quy trình mua sắm DX-OS.'
        ELSE 'Bản ghi do kiểm thử tự động tạo trước migration 000018; đã ngừng hoạt động nhưng vẫn được giữ để bảo toàn lịch sử giao dịch.'
    END,
    version = s.version + 1,
    updated_at = now()
FROM demo_supplier_cleanup c
WHERE s.id = c.id;

CREATE TEMP TABLE demo_request_cleanup ON COMMIT DROP AS
WITH candidates AS (
    SELECT
        pr.id,
        pr.request_code,
        pr.title AS old_title,
        pr.status,
        row_number() OVER (ORDER BY pr.created_at, pr.id) AS row_no
    FROM purchase_requests pr
    WHERE pr.title ~* '(smoke test|workflow|budget release|insufficient budget|reporting reconciliation|attachment|kiểm thử)'
       OR pr.title ~* '^[[:space:]]*g+[[:space:]]*$'
       OR pr.title IN ('Mua laptop gaming', 'Mua ghế', 'Mua thiết bị phục vụ báo cáo demo')
), normalized AS (
    SELECT
        *,
        ((row_no - 1) % 15) + 1 AS scenario_no,
        ((row_no - 1) / 15) + 1 AS batch_no
    FROM candidates
)
SELECT
    id,
    request_code,
    old_title,
    status,
    row_no,
    scenario_no,
    batch_no,
    CASE scenario_no
        WHEN 1 THEN 'Bổ sung laptop cho nhóm triển khai miền Bắc'
        WHEN 2 THEN 'Gia hạn bản quyền phần mềm quản lý dự án'
        WHEN 3 THEN 'Trang bị màn hình cho bộ phận thiết kế'
        WHEN 4 THEN 'Mua thiết bị mạng cho văn phòng tầng 3'
        WHEN 5 THEN 'Bổ sung ghế công thái học cho khu làm việc'
        WHEN 6 THEN 'Nâng cấp máy trạm cho nhóm phân tích dữ liệu'
        WHEN 7 THEN 'Mua bộ lưu điện cho phòng máy chủ'
        WHEN 8 THEN 'Trang bị thiết bị hội nghị cho phòng họp lớn'
        WHEN 9 THEN 'Gia hạn dịch vụ sao lưu dữ liệu doanh nghiệp'
        WHEN 10 THEN 'Mua máy in và vật tư cho bộ phận hành chính'
        WHEN 11 THEN 'Bổ sung điện thoại công vụ cho nhóm kinh doanh'
        WHEN 12 THEN 'Nâng cấp thiết bị bảo mật mạng nội bộ'
        WHEN 13 THEN 'Mua tủ hồ sơ cho phòng tài chính'
        WHEN 14 THEN 'Gia hạn chứng thư số và chữ ký điện tử'
        ELSE 'Trang bị máy chiếu cho phòng đào tạo'
    END || CASE WHEN batch_no > 1 THEN ' – đợt ' || lpad(batch_no::text, 2, '0') ELSE '' END AS new_title,
    CASE scenario_no
        WHEN 1 THEN 'Laptop Dell Latitude 5450'
        WHEN 2 THEN 'Gói bản quyền phần mềm quản lý dự án 12 tháng'
        WHEN 3 THEN 'Màn hình Dell 24 inch'
        WHEN 4 THEN 'Bộ phát Wi-Fi doanh nghiệp'
        WHEN 5 THEN 'Ghế công thái học Ergo Pro'
        WHEN 6 THEN 'Máy trạm HP Z2 Tower'
        WHEN 7 THEN 'Bộ lưu điện UPS 2000VA'
        WHEN 8 THEN 'Bộ thiết bị hội nghị trực tuyến'
        WHEN 9 THEN 'Gói sao lưu dữ liệu doanh nghiệp 12 tháng'
        WHEN 10 THEN 'Máy in đa chức năng Brother'
        WHEN 11 THEN 'Điện thoại công vụ Samsung Galaxy A'
        WHEN 12 THEN 'Thiết bị tường lửa doanh nghiệp'
        WHEN 13 THEN 'Tủ hồ sơ chống ẩm'
        WHEN 14 THEN 'Gói chứng thư số doanh nghiệp 3 năm'
        ELSE 'Máy chiếu Epson văn phòng'
    END AS item_description,
    CASE scenario_no
        WHEN 2 THEN 'gói'
        WHEN 4 THEN 'bộ'
        WHEN 7 THEN 'bộ'
        WHEN 8 THEN 'bộ'
        WHEN 9 THEN 'gói'
        WHEN 12 THEN 'thiết bị'
        WHEN 14 THEN 'gói'
        ELSE 'chiếc'
    END AS item_unit
FROM normalized;

INSERT INTO audit_logs (
    resource_type, resource_id, organization_id, action, actor_id, actor_roles,
    from_status, to_status, correlation_id, metadata
)
SELECT
    'purchase_request', c.id, organization.id, 'DEMO_DATA_NORMALIZED', actor.id, ARRAY['dx_admin'],
    c.status, c.status, 'migration-000018',
    jsonb_build_object(
        'source', 'demo_data_hygiene',
        'requestCode', c.request_code,
        'oldTitle', c.old_title,
        'newTitle', c.new_title,
        'amountChanged', false,
        'deleted', false
    )
FROM demo_request_cleanup c
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
FROM demo_request_cleanup c
WHERE n.resource_id = c.id
  AND position(c.old_title IN n.body) > 0;

UPDATE user_notifications n
SET body = replace(n.body, c.old_title, c.new_title)
FROM demo_request_cleanup c
WHERE n.resource_id = c.id
  AND position(c.old_title IN n.body) > 0;

UPDATE purchase_request_items item
SET
    description = c.item_description || CASE
        WHEN item.line_number > 1 THEN ' – hạng mục ' || item.line_number::text
        ELSE ''
    END,
    unit = c.item_unit,
    updated_at = now()
FROM demo_request_cleanup c
WHERE item.purchase_request_id = c.id;

UPDATE purchase_requests pr
SET
    title = c.new_title,
    reason = CASE pr.status
        WHEN 'REJECTED' THEN 'Nhu cầu đã dừng sau khi rà soát do chưa đáp ứng điều kiện ngân sách hoặc phê duyệt.'
        WHEN 'CHANGES_REQUESTED' THEN 'Cần bổ sung thông số, số lượng hoặc căn cứ ngân sách trước khi tiếp tục.'
        WHEN 'APPROVED' THEN 'Nhu cầu phục vụ hoạt động vận hành đã được các cấp có thẩm quyền xem xét và phê duyệt.'
        WHEN 'DRAFT' THEN 'Bản nháp đang hoàn thiện phạm vi, số lượng và chứng từ mua sắm.'
        WHEN 'SUBMITTED' THEN 'Hồ sơ đã hoàn thiện thông tin cơ bản và đang chờ cấp có thẩm quyền xem xét.'
        ELSE 'Nhu cầu phục vụ hoạt động vận hành của phòng ban theo kế hoạch đã đăng ký.'
    END,
    version = pr.version + 1,
    updated_at = now()
FROM demo_request_cleanup c
WHERE pr.id = c.id;

CREATE TEMP TABLE demo_order_cleanup ON COMMIT DROP AS
SELECT
    po.id,
    po.status,
    po.external_reference AS old_reference,
    po.note AS old_note,
    'HĐMB-' || replace(po.order_code, 'PO-', '') AS new_reference,
    'Đơn hàng phục vụ phiếu ' || pr.request_code || ' – ' || pr.title || '; phát hành theo quy trình mua sắm đã phê duyệt.' AS new_note
FROM purchase_orders po
JOIN purchase_requests pr ON pr.id = po.purchase_request_id
WHERE po.external_reference LIKE 'DEMO-%'
   OR po.note ILIKE '%smoke test%'
   OR lower(coalesce(po.note, '')) IN ('gggg', 'hhhhh', 'testttt', 'hay');

INSERT INTO audit_logs (
    resource_type, resource_id, organization_id, action, actor_id, actor_roles,
    from_status, to_status, correlation_id, metadata
)
SELECT
    'purchase_order', c.id, organization.id, 'DEMO_DATA_NORMALIZED', actor.id, ARRAY['dx_admin'],
    c.status, c.status, 'migration-000018',
    jsonb_build_object(
        'source', 'demo_data_hygiene',
        'oldReference', c.old_reference,
        'newReference', c.new_reference,
        'oldNote', c.old_note,
        'newNote', c.new_note
    )
FROM demo_order_cleanup c
CROSS JOIN organizations organization
CROSS JOIN LATERAL (
    SELECT id
    FROM users
    WHERE username IN ('admin.demo', 'finance.demo')
    ORDER BY CASE username WHEN 'admin.demo' THEN 0 ELSE 1 END
    LIMIT 1
) actor
WHERE organization.code = 'DX-OS';

UPDATE purchase_orders po
SET
    external_reference = c.new_reference,
    note = c.new_note,
    version = po.version + 1,
    updated_at = now()
FROM demo_order_cleanup c
WHERE po.id = c.id;

CREATE TEMP TABLE demo_invoice_cleanup ON COMMIT DROP AS
WITH ranked AS (
    SELECT
        invoice.id,
        invoice.status,
        invoice.invoice_number AS old_number,
        invoice.note AS old_note,
        po.order_code,
        row_number() OVER (
            PARTITION BY invoice.purchase_order_id
            ORDER BY invoice.created_at, invoice.id
        ) AS invoice_no
    FROM purchase_invoices invoice
    JOIN purchase_orders po ON po.id = invoice.purchase_order_id
    WHERE invoice.invoice_number ~* '^(INV-WF|INV-REPLAY|INV-TEST)'
       OR invoice.note ILIKE '%three-way matching%'
)
SELECT
    *,
    'HD-' || replace(order_code, 'PO-', '') || '-' || lpad(invoice_no::text, 2, '0') AS new_number,
    'Hóa đơn được ghi nhận để đối chiếu với đơn hàng và biên bản nhận hàng.' AS new_note
FROM ranked;

INSERT INTO audit_logs (
    resource_type, resource_id, organization_id, action, actor_id, actor_roles,
    from_status, to_status, correlation_id, metadata
)
SELECT
    'purchase_invoice', c.id, organization.id, 'DEMO_DATA_NORMALIZED', actor.id, ARRAY['dx_admin'],
    c.status, c.status, 'migration-000018',
    jsonb_build_object(
        'source', 'demo_data_hygiene',
        'oldNumber', c.old_number,
        'newNumber', c.new_number,
        'oldNote', c.old_note,
        'newNote', c.new_note,
        'amountChanged', false
    )
FROM demo_invoice_cleanup c
CROSS JOIN organizations organization
CROSS JOIN LATERAL (
    SELECT id
    FROM users
    WHERE username IN ('admin.demo', 'finance.demo')
    ORDER BY CASE username WHEN 'admin.demo' THEN 0 ELSE 1 END
    LIMIT 1
) actor
WHERE organization.code = 'DX-OS';

UPDATE purchase_invoices invoice
SET
    invoice_number = c.new_number,
    note = c.new_note,
    payment_reference = CASE WHEN invoice.payment_reference IS NOT NULL
        THEN 'CK-' || replace(c.order_code, 'PO-', '') || '-01'
        ELSE NULL
    END,
    version = invoice.version + 1,
    updated_at = now()
FROM demo_invoice_cleanup c
WHERE invoice.id = c.id;

WITH ranked_payments AS (
    SELECT
        payment.id,
        c.order_code,
        row_number() OVER (
            PARTITION BY payment.invoice_id
            ORDER BY payment.created_at, payment.id
        ) AS payment_no
    FROM invoice_payments payment
    JOIN demo_invoice_cleanup c ON c.id = payment.invoice_id
)
UPDATE invoice_payments payment
SET
    payment_reference = 'CK-' || replace(r.order_code, 'PO-', '') || '-' || lpad(r.payment_no::text, 2, '0'),
    note = 'Thanh toán theo kế hoạch công nợ đã được phê duyệt.'
FROM ranked_payments r
WHERE payment.id = r.id;

WITH ranked_quotes AS (
    SELECT
        quote.id,
        row_number() OVER (ORDER BY quote.created_at, quote.id) AS quote_no
    FROM supplier_quotes quote
    WHERE quote.quote_reference ~* '^(BG-A-|BG-B-)'
       OR quote.note ILIKE '%kiểm thử%'
)
UPDATE supplier_quotes quote
SET
    quote_reference = 'BG-2026-' || lpad(r.quote_no::text, 6, '0'),
    note = 'Báo giá thiết bị và điều kiện giao hàng theo hồ sơ yêu cầu.',
    version = quote.version + 1,
    updated_at = now()
FROM ranked_quotes r
WHERE quote.id = r.id;

UPDATE process_events
SET comment = CASE comment
    WHEN 'Department approval.' THEN 'Trưởng bộ phận đã phê duyệt nhu cầu.'
    WHEN 'Budget approved.' THEN 'Bộ phận tài chính đã xác nhận ngân sách.'
    WHEN 'Adjust the specification.' THEN 'Cần điều chỉnh thông số kỹ thuật và làm rõ nhu cầu.'
    WHEN 'Insufficient budget test cleanup.' THEN 'Dừng xử lý do ngân sách khả dụng chưa đáp ứng.'
    WHEN 'Please confirm the delivery date before approval.' THEN 'Vui lòng xác nhận ngày giao dự kiến trước khi phê duyệt.'
    WHEN 'Order created by the end-to-end smoke test.' THEN 'Đơn hàng được tạo trong quá trình kiểm tra luồng nghiệp vụ.'
    WHEN 'hay' THEN 'Đã kiểm tra và chuyển bước xử lý tiếp theo.'
    WHEN 'hhhhh' THEN 'Đã xác nhận thông tin nghiệp vụ.'
    WHEN 'testttt' THEN 'Đã kiểm tra dữ liệu và hoàn tất thao tác.'
    ELSE comment
END
WHERE comment IN (
    'Department approval.',
    'Budget approved.',
    'Adjust the specification.',
    'Insufficient budget test cleanup.',
    'Please confirm the delivery date before approval.',
    'Order created by the end-to-end smoke test.',
    'hay', 'hhhhh', 'testttt'
);

UPDATE invoice_events
SET comment = CASE comment
    WHEN 'Invoice created before receipt to verify three-way matching.' THEN 'Hóa đơn được ghi nhận trước khi nhận hàng để kiểm tra đối chiếu ba bên.'
    WHEN 'Payment completed.' THEN 'Đã hoàn tất thanh toán.'
    WHEN 'Invoice verified.' THEN 'Hóa đơn đã được đối chiếu và xác nhận.'
    ELSE comment
END
WHERE comment IN (
    'Invoice created before receipt to verify three-way matching.',
    'Payment completed.',
    'Invoice verified.'
);

UPDATE purchase_order_receipts
SET note = CASE note
    WHEN 'Partial receipt for workflow verification.' THEN 'Nhận hàng một phần theo tiến độ giao thực tế.'
    WHEN 'Complete receipt for workflow verification.' THEN 'Đã nhận đủ hàng và kiểm tra số lượng.'
    WHEN 'Receipt recorded by the end-to-end smoke test.' THEN 'Đã ghi nhận kết quả nhận hàng theo đơn mua.'
    WHEN 'hay' THEN 'Hàng hóa đã được kiểm tra khi bàn giao.'
    WHEN 'hhhhh' THEN 'Đã xác nhận số lượng hàng thực nhận.'
    ELSE note
END
WHERE note IN (
    'Partial receipt for workflow verification.',
    'Complete receipt for workflow verification.',
    'Receipt recorded by the end-to-end smoke test.',
    'hay', 'hhhhh'
);

UPDATE approval_rules
SET
    name = CASE
        WHEN name ILIKE '%kiểm thử%' THEN 'Quy tắc phê duyệt theo hạn mức phòng ban'
        WHEN name ILIKE '%moi%' THEN 'Quy trình phê duyệt theo hạn mức tiêu chuẩn'
        ELSE name
    END,
    updated_at = now(),
    version = version + 1
WHERE name ILIKE '%kiểm thử%'
   OR name ILIKE '%moi%';

UPDATE approval_delegations
SET
    reason = 'Ủy quyền xử lý phiếu trong thời gian trưởng bộ phận vắng mặt.',
    updated_at = now(),
    version = version + 1
WHERE reason ILIKE '%kiểm thử%';

UPDATE audit_cases
SET
    title = 'Rà soát kiểm soát vận hành mua sắm',
    description = 'Kiểm tra việc tuân thủ quy trình phê duyệt, ngân sách, nhận hàng và thanh toán.',
    updated_at = now(),
    version = version + 1
WHERE title ILIKE '%smoke test%'
   OR description ILIKE '%smoke test%';

CREATE TEMP TABLE demo_user_names(username text PRIMARY KEY, display_name text) ON COMMIT DROP;
INSERT INTO demo_user_names(username, display_name) VALUES
    ('employee.demo', 'Nguyễn Minh Anh'),
    ('manager.demo', 'Trần Thu Hà'),
    ('finance.demo', 'Lê Hoàng Nam'),
    ('auditor.demo', 'Phạm Quỳnh Trang'),
    ('admin.demo', 'Đỗ Đức Long'),
    ('ai.operator.demo', 'Trợ lý kiểm soát DX-OS'),
    ('workflow.employee', 'Kiểm thử tự động – Nhân viên quy trình'),
    ('workflow.manager', 'Kiểm thử tự động – Trưởng bộ phận'),
    ('workflow.finance', 'Kiểm thử tự động – Tài chính'),
    ('workflow.auditor', 'Kiểm thử tự động – Kiểm toán'),
    ('workflow.admin', 'Kiểm thử tự động – Quản trị'),
    ('workflow.outsider', 'Kiểm thử tự động – Nhân viên ngoài phạm vi'),
    ('budget.finance', 'Kiểm thử tự động – Tài chính ngân sách'),
    ('budget.auditor', 'Kiểm thử tự động – Kiểm toán ngân sách'),
    ('attachments.employee', 'Kiểm thử tự động – Nhân viên đính kèm'),
    ('attachments.outsider', 'Kiểm thử tự động – Người ngoài phạm vi tệp'),
    ('reporting.employee', 'Kiểm thử tự động – Nhân viên báo cáo'),
    ('reporting.finance', 'Kiểm thử tự động – Tài chính báo cáo'),
    ('reporting.auditor', 'Kiểm thử tự động – Kiểm toán báo cáo');

INSERT INTO audit_logs (
    resource_type, resource_id, organization_id, action, actor_id, actor_roles,
    from_status, to_status, correlation_id, metadata
)
SELECT
    'user_profile', u.id, organization.id, 'DEMO_DATA_NORMALIZED', actor.id, ARRAY['dx_admin'],
    CASE WHEN u.active THEN 'ACTIVE' ELSE 'INACTIVE' END,
    CASE WHEN u.active THEN 'ACTIVE' ELSE 'INACTIVE' END,
    'migration-000018',
    jsonb_build_object(
        'source', 'demo_data_hygiene',
        'username', u.username,
        'oldDisplayName', u.display_name,
        'newDisplayName', n.display_name
    )
FROM users u
JOIN demo_user_names n ON n.username = u.username
CROSS JOIN organizations organization
CROSS JOIN LATERAL (
    SELECT id
    FROM users
    WHERE username IN ('admin.demo', 'finance.demo')
    ORDER BY CASE username WHEN 'admin.demo' THEN 0 ELSE 1 END
    LIMIT 1
) actor
WHERE organization.code = 'DX-OS'
  AND u.display_name IS DISTINCT FROM n.display_name;

UPDATE users u
SET
    display_name = n.display_name,
    version = u.version + 1,
    updated_at = now()
FROM demo_user_names n
WHERE u.username = n.username
  AND u.display_name IS DISTINCT FROM n.display_name;

INSERT INTO app_metadata (key, value)
VALUES ('schema_version', '000018_normalize_demo_business_data')
ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value, updated_at = now();
