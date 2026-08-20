-- Repair historical lab records that were created through a non-UTF-8 input
-- path. This migration deliberately targets only the known corrupted test
-- strings; it does not alter valid question marks or production data.

UPDATE approval_delegations
SET reason = 'Kiểm thử luồng ủy quyền phê duyệt có kiểm soát'
WHERE reason LIKE 'Ki?m th? lu?ng ?y quy?n ph%';

UPDATE approval_rules
SET name = 'Quy tắc kiểm thử 1787219162'
WHERE name LIKE 'Quy t?c ki?m th? 1787219162';

UPDATE purchase_requests
SET title = 'Kiểm thử so sánh báo giá 1787219205',
    reason = 'Kiểm thử đầy đủ luồng lựa chọn nhà cung cấp sau phê duyệt'
WHERE title LIKE 'Ki?m th? so s%nh b%o gi% 1787219205';

UPDATE purchase_request_items
SET description = 'Thiết bị kiểm thử báo giá'
WHERE description LIKE 'Thi?t b? ki?m th? b%o gi%';

UPDATE process_events
SET comment = CASE
    WHEN comment LIKE 'D?ng % nhu c?u ki?m th? b%o gi%' THEN 'Đồng ý nhu cầu kiểm thử báo giá'
    WHEN comment LIKE 'Ng%n s%ch d%p ?ng cho ki?m th? b%o gi%' THEN 'Ngân sách đáp ứng cho kiểm thử báo giá'
    ELSE comment
END
WHERE comment LIKE 'D?ng % nhu c?u ki?m th? b%o gi%'
   OR comment LIKE 'Ng%n s%ch d%p ?ng cho ki?m th? b%o gi%';

UPDATE sourcing_events
SET comment = CASE
    WHEN comment LIKE 'B%o gi% A ki?m th?' THEN 'Báo giá A kiểm thử'
    WHEN comment LIKE 'B%o gi% B ki?m th?' THEN 'Báo giá B kiểm thử'
    WHEN comment LIKE 'Ch?n b%o gi% c% di?m t%ng h?p cao nh?t d? ki?m th?' THEN 'Chọn báo giá có điểm tổng hợp cao nhất để kiểm thử'
    ELSE comment
END
WHERE comment LIKE 'B%o gi% A ki?m th?'
   OR comment LIKE 'B%o gi% B ki?m th?'
   OR comment LIKE 'Ch?n b%o gi% c% di?m t%ng h?p cao nh?t d? ki?m th?';

UPDATE supplier_quotes
SET note = CASE
        WHEN note LIKE 'B%o gi% A ki?m th?' THEN 'Báo giá A kiểm thử'
        WHEN note LIKE 'B%o gi% B ki?m th?' THEN 'Báo giá B kiểm thử'
        ELSE note
    END,
    payment_terms = CASE
        WHEN payment_terms LIKE 'Thanh to%n 30 ng%y sau nghi?m thu' THEN 'Thanh toán 30 ngày sau nghiệm thu'
        ELSE payment_terms
    END
WHERE note LIKE 'B%o gi% A ki?m th?'
   OR note LIKE 'B%o gi% B ki?m th?'
   OR payment_terms LIKE 'Thanh to%n 30 ng%y sau nghi?m thu';

UPDATE outbox_events
SET body = 'PR-2026-000046 - Kiểm thử so sánh báo giá 1787219205'
WHERE body LIKE 'PR-2026-000046 - Ki?m th? so s%nh b%o gi% 1787219205';

UPDATE user_notifications
SET body = 'PR-2026-000046 - Kiểm thử so sánh báo giá 1787219205'
WHERE body LIKE 'PR-2026-000046 - Ki?m th? so s%nh b%o gi% 1787219205';
