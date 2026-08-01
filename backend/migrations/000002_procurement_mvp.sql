CREATE TABLE organizations (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    code varchar(50) NOT NULL UNIQUE,
    name varchar(255) NOT NULL,
    status varchar(30) NOT NULL DEFAULT 'ACTIVE'
        CHECK (status IN ('ACTIVE', 'INACTIVE')),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE departments (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id uuid NOT NULL REFERENCES organizations(id),
    parent_id uuid REFERENCES departments(id),
    code varchar(50) NOT NULL,
    name varchar(255) NOT NULL,
    cost_center varchar(100),
    active boolean NOT NULL DEFAULT true,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (organization_id, code)
);

CREATE TABLE users (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    keycloak_subject varchar(255) NOT NULL UNIQUE,
    username varchar(150) NOT NULL,
    email varchar(255),
    display_name varchar(255) NOT NULL,
    department_id uuid NOT NULL REFERENCES departments(id),
    active boolean NOT NULL DEFAULT true,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE SEQUENCE purchase_request_code_seq;

CREATE TABLE purchase_requests (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    request_code varchar(30) NOT NULL UNIQUE,
    requester_id uuid NOT NULL REFERENCES users(id),
    department_id uuid NOT NULL REFERENCES departments(id),
    title varchar(255) NOT NULL,
    reason text NOT NULL,
    currency char(3) NOT NULL CHECK (currency ~ '^[A-Z]{3}$'),
    total_amount numeric(19,4) NOT NULL DEFAULT 0 CHECK (total_amount >= 0),
    cost_center varchar(100) NOT NULL,
    status varchar(40) NOT NULL DEFAULT 'DRAFT'
        CHECK (status IN (
            'DRAFT',
            'SUBMITTED',
            'MANAGER_APPROVED',
            'CHANGES_REQUESTED',
            'APPROVED',
            'REJECTED',
            'CANCELLED'
        )),
    current_assignee_id uuid REFERENCES users(id),
    submitted_at timestamptz,
    approved_at timestamptz,
    sla_due_at timestamptz,
    version bigint NOT NULL DEFAULT 1 CHECK (version > 0),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE purchase_request_items (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    purchase_request_id uuid NOT NULL REFERENCES purchase_requests(id) ON DELETE CASCADE,
    line_number integer NOT NULL CHECK (line_number > 0),
    description varchar(500) NOT NULL,
    quantity numeric(15,4) NOT NULL CHECK (quantity > 0),
    unit varchar(50) NOT NULL,
    unit_price numeric(19,4) NOT NULL CHECK (unit_price >= 0),
    line_total numeric(19,4) GENERATED ALWAYS AS (quantity * unit_price) STORED,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (purchase_request_id, line_number)
);

CREATE TABLE process_events (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    purchase_request_id uuid NOT NULL REFERENCES purchase_requests(id) ON DELETE CASCADE,
    event_type varchar(80) NOT NULL,
    from_status varchar(40),
    to_status varchar(40) NOT NULL,
    actor_id uuid NOT NULL REFERENCES users(id),
    actor_roles text[] NOT NULL DEFAULT '{}',
    comment text,
    metadata jsonb NOT NULL DEFAULT '{}',
    occurred_at timestamptz NOT NULL DEFAULT now(),
    correlation_id varchar(128),
    idempotency_key varchar(255)
);

CREATE UNIQUE INDEX process_events_idempotency_key_unique
    ON process_events (idempotency_key)
    WHERE idempotency_key IS NOT NULL;
CREATE INDEX purchase_requests_department_status_created_idx
    ON purchase_requests (department_id, status, created_at DESC);
CREATE INDEX purchase_requests_requester_created_idx
    ON purchase_requests (requester_id, created_at DESC);
CREATE INDEX purchase_requests_assignee_status_idx
    ON purchase_requests (current_assignee_id, status);
CREATE INDEX process_events_request_occurred_idx
    ON process_events (purchase_request_id, occurred_at);

INSERT INTO organizations (id, code, name)
VALUES ('00000000-0000-4000-8000-000000000001', 'DX-OS', 'DX-OS Demo Organization');

INSERT INTO departments (id, organization_id, code, name, cost_center)
VALUES (
    '00000000-0000-4000-8000-000000000101',
    '00000000-0000-4000-8000-000000000001',
    'GENERAL',
    'General Department',
    'CC-GENERAL'
);

INSERT INTO app_metadata (key, value)
VALUES ('schema_version', 'procurement-mvp-step-3')
ON CONFLICT (key) DO UPDATE
SET value = EXCLUDED.value,
    updated_at = now();
