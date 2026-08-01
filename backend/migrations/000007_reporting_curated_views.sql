CREATE SCHEMA reporting;
REVOKE ALL ON SCHEMA reporting FROM PUBLIC;

CREATE TABLE reporting.sla_policies (
    organization_id uuid NOT NULL REFERENCES organizations(id),
    process_name varchar(80) NOT NULL,
    target_hours integer NOT NULL CHECK (target_hours > 0),
    active boolean NOT NULL DEFAULT true,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (organization_id, process_name)
);

INSERT INTO reporting.sla_policies (organization_id, process_name, target_hours)
SELECT id, 'PURCHASE_REQUEST_APPROVAL', 72
FROM organizations
WHERE code = 'DX-OS';

CREATE VIEW reporting.purchase_request_facts
WITH (security_barrier = true)
AS
SELECT
    pr.id AS purchase_request_id,
    pr.request_code,
    pr.title,
    o.id AS organization_id,
    o.code AS organization_code,
    o.name AS organization_name,
    d.id AS department_id,
    d.code AS department_code,
    d.name AS department_name,
    pr.requester_id,
    u.username AS requester_username,
    pr.cost_center,
    pr.currency,
    pr.total_amount,
    pr.status,
    pr.created_at,
    pr.created_at::date AS created_date,
    pr.submitted_at,
    CASE
        WHEN pr.status = 'APPROVED' THEN pr.approved_at
        WHEN pr.status IN ('REJECTED', 'CANCELLED') THEN pr.updated_at
        ELSE NULL
    END AS completed_at,
    CASE
        WHEN pr.status = 'APPROVED' AND pr.approved_at IS NOT NULL
            THEN round((extract(epoch FROM (pr.approved_at - pr.created_at)) / 3600.0)::numeric, 2)
        WHEN pr.status IN ('REJECTED', 'CANCELLED')
            THEN round((extract(epoch FROM (pr.updated_at - pr.created_at)) / 3600.0)::numeric, 2)
        ELSE NULL
    END AS lead_time_hours,
    COALESCE(
        pr.sla_due_at,
        CASE
            WHEN pr.submitted_at IS NOT NULL
                THEN pr.submitted_at + make_interval(hours => COALESCE(sp.target_hours, 72))
            ELSE NULL
        END
    ) AS effective_sla_due_at,
    CASE
        WHEN pr.submitted_at IS NULL THEN false
        WHEN COALESCE(
            pr.sla_due_at,
            pr.submitted_at + make_interval(hours => COALESCE(sp.target_hours, 72))
        ) < COALESCE(
            CASE
                WHEN pr.status = 'APPROVED' THEN pr.approved_at
                WHEN pr.status IN ('REJECTED', 'CANCELLED') THEN pr.updated_at
                ELSE NULL
            END,
            now()
        ) THEN true
        ELSE false
    END AS sla_breached,
    COALESCE(events.has_changes_requested, false) AS returned_for_changes,
    COALESCE(attachments.attachment_count, 0)::integer AS attachment_count,
    CASE
        WHEN ar.id IS NOT NULL AND pr.total_amount >= ar.threshold_amount THEN true
        ELSE false
    END AS attachment_required,
    CASE
        WHEN ar.id IS NULL OR pr.total_amount < ar.threshold_amount THEN true
        ELSE COALESCE(attachments.required_attachment_count, 0) > 0
    END AS attachment_compliant
FROM purchase_requests pr
JOIN users u ON u.id = pr.requester_id
JOIN departments d ON d.id = pr.department_id
JOIN organizations o ON o.id = d.organization_id
LEFT JOIN reporting.sla_policies sp
    ON sp.organization_id = o.id
   AND sp.process_name = 'PURCHASE_REQUEST_APPROVAL'
   AND sp.active
LEFT JOIN attachment_rules ar
    ON ar.organization_id = o.id
   AND ar.currency = pr.currency
   AND ar.active
LEFT JOIN LATERAL (
    SELECT
        count(*) FILTER (WHERE pa.status = 'ACTIVE') AS attachment_count,
        count(*) FILTER (
            WHERE pa.status = 'ACTIVE'
              AND pa.document_type = ar.required_document_type
        ) AS required_attachment_count
    FROM purchase_request_attachments pa
    WHERE pa.purchase_request_id = pr.id
) attachments ON true
LEFT JOIN LATERAL (
    SELECT bool_or(pe.event_type = 'CHANGES_REQUESTED') AS has_changes_requested
    FROM process_events pe
    WHERE pe.purchase_request_id = pr.id
) events ON true;

CREATE VIEW reporting.daily_procurement_metrics
WITH (security_barrier = true)
AS
SELECT
    created_date AS metric_date,
    organization_id,
    organization_code,
    department_id,
    department_code,
    department_name,
    cost_center,
    currency,
    count(*) AS request_count,
    sum(total_amount) AS total_amount,
    count(*) FILTER (WHERE status = 'APPROVED') AS approved_count,
    count(*) FILTER (WHERE status = 'REJECTED') AS rejected_count,
    count(*) FILTER (WHERE returned_for_changes) AS returned_count,
    count(*) FILTER (WHERE sla_breached) AS sla_breached_count,
    count(*) FILTER (WHERE attachment_required) AS attachment_required_count,
    count(*) FILTER (
        WHERE attachment_required AND attachment_compliant
    ) AS attachment_compliant_count
FROM reporting.purchase_request_facts
GROUP BY
    created_date,
    organization_id,
    organization_code,
    department_id,
    department_code,
    department_name,
    cost_center,
    currency;

CREATE VIEW reporting.budget_utilization
WITH (security_barrier = true)
AS
SELECT
    bp.organization_id,
    o.code AS organization_code,
    bp.id AS budget_period_id,
    bp.code AS period_code,
    bp.starts_on AS period_start,
    bp.ends_on AS period_end,
    bp.status AS period_status,
    ba.id AS allocation_id,
    ba.cost_center,
    ba.currency,
    ba.allocated_amount,
    ba.reserved_amount,
    ba.committed_amount,
    ba.allocated_amount - ba.reserved_amount - ba.committed_amount AS available_amount,
    CASE
        WHEN ba.allocated_amount = 0 THEN 0
        ELSE round(
            ((ba.reserved_amount + ba.committed_amount) / ba.allocated_amount * 100)::numeric,
            2
        )
    END AS utilization_percent
FROM budget_allocations ba
JOIN budget_periods bp ON bp.id = ba.budget_period_id
JOIN organizations o ON o.id = bp.organization_id;

INSERT INTO app_metadata (key, value)
VALUES ('schema_version', '000007_reporting_curated_views')
ON CONFLICT (key) DO UPDATE
SET value = EXCLUDED.value,
    updated_at = now();
