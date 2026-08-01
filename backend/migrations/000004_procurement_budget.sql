CREATE TABLE budget_periods (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id uuid NOT NULL REFERENCES organizations(id),
    code varchar(50) NOT NULL,
    name varchar(255) NOT NULL,
    starts_on date NOT NULL,
    ends_on date NOT NULL,
    status varchar(20) NOT NULL DEFAULT 'ACTIVE'
        CHECK (status IN ('DRAFT', 'ACTIVE', 'CLOSED')),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CHECK (ends_on >= starts_on),
    UNIQUE (organization_id, code)
);

CREATE TABLE budget_allocations (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    budget_period_id uuid NOT NULL REFERENCES budget_periods(id),
    cost_center varchar(100) NOT NULL,
    currency char(3) NOT NULL CHECK (currency ~ '^[A-Z]{3}$'),
    allocated_amount numeric(19,4) NOT NULL CHECK (allocated_amount >= 0),
    reserved_amount numeric(19,4) NOT NULL DEFAULT 0 CHECK (reserved_amount >= 0),
    committed_amount numeric(19,4) NOT NULL DEFAULT 0 CHECK (committed_amount >= 0),
    version bigint NOT NULL DEFAULT 1 CHECK (version > 0),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CHECK (reserved_amount + committed_amount <= allocated_amount),
    UNIQUE (budget_period_id, cost_center, currency)
);

CREATE TABLE budget_reservations (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    budget_allocation_id uuid NOT NULL REFERENCES budget_allocations(id),
    purchase_request_id uuid NOT NULL REFERENCES purchase_requests(id),
    amount numeric(19,4) NOT NULL CHECK (amount > 0),
    currency char(3) NOT NULL CHECK (currency ~ '^[A-Z]{3}$'),
    status varchar(20) NOT NULL CHECK (status IN ('RESERVED', 'COMMITTED', 'RELEASED')),
    reserved_by uuid NOT NULL REFERENCES users(id),
    committed_by uuid REFERENCES users(id),
    released_by uuid REFERENCES users(id),
    reserved_at timestamptz NOT NULL DEFAULT now(),
    committed_at timestamptz,
    released_at timestamptz,
    updated_at timestamptz NOT NULL DEFAULT now(),
    CHECK (
        (status = 'RESERVED' AND committed_at IS NULL AND released_at IS NULL) OR
        (status = 'COMMITTED' AND committed_by IS NOT NULL AND committed_at IS NOT NULL AND released_at IS NULL) OR
        (status = 'RELEASED' AND released_by IS NOT NULL AND released_at IS NOT NULL AND committed_at IS NULL)
    )
);

CREATE UNIQUE INDEX budget_reservations_active_request_unique
    ON budget_reservations (purchase_request_id)
    WHERE status = 'RESERVED';
CREATE UNIQUE INDEX budget_reservations_committed_request_unique
    ON budget_reservations (purchase_request_id)
    WHERE status = 'COMMITTED';
CREATE INDEX budget_periods_organization_active_idx
    ON budget_periods (organization_id, starts_on, ends_on)
    WHERE status = 'ACTIVE';
CREATE INDEX budget_allocations_lookup_idx
    ON budget_allocations (budget_period_id, cost_center, currency);
CREATE INDEX budget_reservations_request_created_idx
    ON budget_reservations (purchase_request_id, reserved_at DESC);

INSERT INTO budget_periods (
    id,
    organization_id,
    code,
    name,
    starts_on,
    ends_on,
    status
)
VALUES (
    '00000000-0000-4000-8000-000000000201',
    '00000000-0000-4000-8000-000000000001',
    'FY-' || extract(year FROM CURRENT_DATE)::integer,
    'Demo budget ' || extract(year FROM CURRENT_DATE)::integer,
    date_trunc('year', CURRENT_DATE)::date,
    (date_trunc('year', CURRENT_DATE) + interval '1 year - 1 day')::date,
    'ACTIVE'
);

INSERT INTO budget_allocations (
    id,
    budget_period_id,
    cost_center,
    currency,
    allocated_amount
)
VALUES (
    '00000000-0000-4000-8000-000000000301',
    '00000000-0000-4000-8000-000000000201',
    'CC-GENERAL',
    'VND',
    100000000000.0000
);

INSERT INTO app_metadata (key, value)
VALUES ('schema_version', 'procurement-mvp-step-7')
ON CONFLICT (key) DO UPDATE
SET value = EXCLUDED.value,
    updated_at = now();
