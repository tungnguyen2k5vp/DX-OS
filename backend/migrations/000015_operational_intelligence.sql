-- Operational intelligence: guided requests, configurable approvals, sourcing,
-- delegation and role-governance snapshots. All additions are backwards
-- compatible with the existing purchase-to-pay workflow.

CREATE TABLE procurement_catalog_items (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id uuid NOT NULL REFERENCES organizations(id),
    code varchar(50) NOT NULL,
    name varchar(255) NOT NULL,
    description varchar(500) NOT NULL,
    category varchar(100) NOT NULL,
    unit varchar(50) NOT NULL,
    reference_unit_price numeric(19,4) NOT NULL CHECK (reference_unit_price >= 0),
    currency char(3) NOT NULL CHECK (currency ~ '^[A-Z]{3}$'),
    active boolean NOT NULL DEFAULT true,
    version bigint NOT NULL DEFAULT 1 CHECK (version > 0),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (organization_id, code)
);

CREATE INDEX procurement_catalog_items_org_category_name_idx
    ON procurement_catalog_items (organization_id, active, category, name);

INSERT INTO procurement_catalog_items (
    organization_id, code, name, description, category, unit,
    reference_unit_price, currency
)
SELECT id, item.code, item.name, item.description, item.category, item.unit,
       item.reference_unit_price, 'VND'
FROM organizations o
CROSS JOIN (VALUES
    ('LAPTOP-OFFICE', 'Laptop văn phòng', 'Laptop làm việc tiêu chuẩn cho nhân viên', 'Thiết bị CNTT', 'chiếc', 22000000::numeric),
    ('MONITOR-24', 'Màn hình 24 inch', 'Màn hình IPS 24 inch phục vụ công việc', 'Thiết bị CNTT', 'chiếc', 4200000::numeric),
    ('CHAIR-ERGONOMIC', 'Ghế công thái học', 'Ghế làm việc có hỗ trợ cột sống', 'Nội thất văn phòng', 'chiếc', 3500000::numeric),
    ('SOFTWARE-ANNUAL', 'Bản quyền phần mềm một năm', 'Gói bản quyền phần mềm theo người dùng trong 12 tháng', 'Phần mềm', 'gói', 6000000::numeric)
) AS item(code, name, description, category, unit, reference_unit_price)
WHERE o.code = 'DX-OS'
ON CONFLICT (organization_id, code) DO NOTHING;

CREATE TABLE approval_rules (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id uuid NOT NULL REFERENCES organizations(id),
    department_id uuid REFERENCES departments(id),
    name varchar(255) NOT NULL,
    currency char(3) NOT NULL CHECK (currency ~ '^[A-Z]{3}$'),
    minimum_amount numeric(19,4) NOT NULL DEFAULT 0 CHECK (minimum_amount >= 0),
    maximum_amount numeric(19,4) CHECK (maximum_amount IS NULL OR maximum_amount >= minimum_amount),
    requires_manager boolean NOT NULL DEFAULT true,
    requires_finance boolean NOT NULL DEFAULT true,
    priority integer NOT NULL DEFAULT 100 CHECK (priority >= 0),
    active boolean NOT NULL DEFAULT true,
    version bigint NOT NULL DEFAULT 1 CHECK (version > 0),
    created_by uuid REFERENCES users(id),
    updated_by uuid REFERENCES users(id),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CHECK (requires_manager OR requires_finance)
);

CREATE INDEX approval_rules_resolution_idx
    ON approval_rules (organization_id, currency, active, priority, minimum_amount);

INSERT INTO approval_rules (
    organization_id, name, currency, minimum_amount, maximum_amount,
    requires_manager, requires_finance, priority
)
SELECT id, 'Quy trình phê duyệt mặc định', 'VND', 0, NULL, true, true, 1000
FROM organizations
WHERE code = 'DX-OS';

CREATE TABLE approval_delegations (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id uuid NOT NULL REFERENCES organizations(id),
    department_id uuid NOT NULL REFERENCES departments(id),
    delegator_user_id uuid NOT NULL REFERENCES users(id),
    delegate_user_id uuid NOT NULL REFERENCES users(id),
    starts_on date NOT NULL,
    ends_on date NOT NULL,
    reason varchar(1000) NOT NULL,
    active boolean NOT NULL DEFAULT true,
    version bigint NOT NULL DEFAULT 1 CHECK (version > 0),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CHECK (delegator_user_id <> delegate_user_id),
    CHECK (ends_on >= starts_on)
);

CREATE INDEX approval_delegations_delegate_window_idx
    ON approval_delegations (delegate_user_id, active, starts_on, ends_on);
CREATE INDEX approval_delegations_delegator_idx
    ON approval_delegations (delegator_user_id, created_at DESC);

CREATE TABLE sourcing_cases (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id uuid NOT NULL REFERENCES organizations(id),
    purchase_request_id uuid NOT NULL UNIQUE REFERENCES purchase_requests(id),
    status varchar(20) NOT NULL DEFAULT 'OPEN'
        CHECK (status IN ('OPEN', 'AWARDED', 'CANCELLED')),
    selected_quote_id uuid,
    created_by uuid NOT NULL REFERENCES users(id),
    awarded_by uuid REFERENCES users(id),
    awarded_at timestamptz,
    version bigint NOT NULL DEFAULT 1 CHECK (version > 0),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE supplier_quotes (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    sourcing_case_id uuid NOT NULL REFERENCES sourcing_cases(id) ON DELETE CASCADE,
    supplier_id uuid NOT NULL REFERENCES suppliers(id),
    quote_reference varchar(100) NOT NULL,
    amount numeric(19,4) NOT NULL CHECK (amount > 0),
    currency char(3) NOT NULL CHECK (currency ~ '^[A-Z]{3}$'),
    delivery_on date NOT NULL,
    warranty_months integer NOT NULL DEFAULT 0 CHECK (warranty_months >= 0 AND warranty_months <= 240),
    payment_terms varchar(500) NOT NULL,
    note varchar(2000),
    status varchar(20) NOT NULL DEFAULT 'SUBMITTED'
        CHECK (status IN ('SUBMITTED', 'SELECTED', 'REJECTED', 'WITHDRAWN')),
    version bigint NOT NULL DEFAULT 1 CHECK (version > 0),
    created_by uuid NOT NULL REFERENCES users(id),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (sourcing_case_id, supplier_id, quote_reference)
);

ALTER TABLE sourcing_cases
    ADD CONSTRAINT sourcing_cases_selected_quote_fk
    FOREIGN KEY (selected_quote_id) REFERENCES supplier_quotes(id);

CREATE UNIQUE INDEX supplier_quotes_one_selected_per_case_idx
    ON supplier_quotes (sourcing_case_id)
    WHERE status = 'SELECTED';
CREATE INDEX supplier_quotes_case_score_inputs_idx
    ON supplier_quotes (sourcing_case_id, status, amount, delivery_on);

CREATE TABLE sourcing_events (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    sourcing_case_id uuid NOT NULL REFERENCES sourcing_cases(id) ON DELETE CASCADE,
    quote_id uuid REFERENCES supplier_quotes(id),
    event_type varchar(80) NOT NULL,
    actor_id uuid NOT NULL REFERENCES users(id),
    actor_roles text[] NOT NULL DEFAULT '{}',
    comment text,
    correlation_id varchar(128),
    idempotency_key varchar(255) UNIQUE,
    occurred_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX sourcing_events_case_time_idx
    ON sourcing_events (sourcing_case_id, occurred_at DESC, id DESC);

CREATE TABLE user_role_snapshots (
    user_id uuid PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    organization_id uuid NOT NULL REFERENCES organizations(id),
    roles text[] NOT NULL DEFAULT '{}',
    captured_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX user_role_snapshots_org_idx
    ON user_role_snapshots (organization_id, captured_at DESC);
CREATE INDEX user_role_snapshots_roles_gin_idx
    ON user_role_snapshots USING gin (roles);

INSERT INTO app_metadata (key, value)
VALUES ('schema_version', '000015_operational_intelligence')
ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value, updated_at = now();
