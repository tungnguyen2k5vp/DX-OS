ALTER TABLE suppliers
    ADD COLUMN address text,
    ADD COLUMN bank_name varchar(255),
    ADD COLUMN bank_account_number varchar(100),
    ADD COLUMN contract_reference varchar(100),
    ADD COLUMN contract_expires_on date,
    ADD COLUMN compliance_status varchar(20) NOT NULL DEFAULT 'PENDING'
        CHECK (compliance_status IN ('PENDING', 'VERIFIED', 'EXPIRED', 'BLOCKED')),
    ADD COLUMN performance_score numeric(5,2)
        CHECK (performance_score IS NULL OR (performance_score >= 0 AND performance_score <= 100)),
    ADD COLUMN business_note text;

ALTER TABLE users ADD COLUMN version bigint NOT NULL DEFAULT 1 CHECK (version > 0);
ALTER TABLE departments ADD COLUMN version bigint NOT NULL DEFAULT 1 CHECK (version > 0);

ALTER TABLE purchase_orders
    ADD COLUMN cancelled_by uuid REFERENCES users(id),
    ADD COLUMN cancelled_at timestamptz,
    ADD COLUMN cancellation_reason text;

ALTER TABLE purchase_orders DROP CONSTRAINT IF EXISTS purchase_orders_status_check;
ALTER TABLE purchase_orders DROP CONSTRAINT IF EXISTS purchase_orders_check;
ALTER TABLE purchase_orders
    ADD CONSTRAINT purchase_orders_status_check
        CHECK (status IN ('ORDERED', 'PARTIALLY_RECEIVED', 'RECEIPT_EXCEPTION', 'RECEIVED', 'CANCELLED')),
    ADD CONSTRAINT purchase_orders_lifecycle_check CHECK (
        (status = 'ORDERED' AND actual_delivery_on IS NULL AND received_by IS NULL AND received_at IS NULL
            AND cancelled_by IS NULL AND cancelled_at IS NULL)
        OR
        (status IN ('PARTIALLY_RECEIVED', 'RECEIPT_EXCEPTION') AND actual_delivery_on IS NOT NULL
            AND received_by IS NOT NULL AND received_at IS NOT NULL
            AND cancelled_by IS NULL AND cancelled_at IS NULL)
        OR
        (status = 'RECEIVED' AND actual_delivery_on IS NOT NULL AND received_by IS NOT NULL
            AND received_at IS NOT NULL AND cancelled_by IS NULL AND cancelled_at IS NULL)
        OR
        (status = 'CANCELLED' AND cancelled_by IS NOT NULL AND cancelled_at IS NOT NULL
            AND cancellation_reason IS NOT NULL)
    );

CREATE TABLE purchase_order_receipts (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    purchase_order_id uuid NOT NULL REFERENCES purchase_orders(id) ON DELETE CASCADE,
    receipt_number bigint GENERATED ALWAYS AS IDENTITY,
    outcome varchar(30) NOT NULL
        CHECK (outcome IN ('PARTIAL', 'COMPLETE', 'DAMAGED', 'WRONG_ITEM', 'REJECTED')),
    received_on date NOT NULL,
    note text NOT NULL,
    created_by uuid NOT NULL REFERENCES users(id),
    created_at timestamptz NOT NULL DEFAULT now(),
    correlation_id varchar(128),
    idempotency_key varchar(255) NOT NULL UNIQUE
);

CREATE TABLE purchase_order_receipt_items (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    receipt_id uuid NOT NULL REFERENCES purchase_order_receipts(id) ON DELETE CASCADE,
    purchase_request_item_id uuid NOT NULL REFERENCES purchase_request_items(id),
    quantity_received numeric(15,4) NOT NULL CHECK (quantity_received >= 0),
    condition varchar(20) NOT NULL DEFAULT 'ACCEPTED'
        CHECK (condition IN ('ACCEPTED', 'DAMAGED', 'WRONG_ITEM', 'REJECTED')),
    note varchar(1000),
    UNIQUE (receipt_id, purchase_request_item_id)
);

CREATE INDEX purchase_order_receipts_order_created_idx
    ON purchase_order_receipts (purchase_order_id, created_at DESC);

ALTER TABLE purchase_invoices DROP CONSTRAINT IF EXISTS purchase_invoices_purchase_order_id_key;
ALTER TABLE purchase_invoices
    ADD COLUMN invoice_type varchar(20) NOT NULL DEFAULT 'STANDARD'
        CHECK (invoice_type IN ('STANDARD', 'ADVANCE', 'FINAL', 'CREDIT_NOTE')),
    ADD COLUMN subtotal_amount numeric(19,4) NOT NULL DEFAULT 0 CHECK (subtotal_amount >= 0),
    ADD COLUMN tax_amount numeric(19,4) NOT NULL DEFAULT 0 CHECK (tax_amount >= 0),
    ADD COLUMN discount_amount numeric(19,4) NOT NULL DEFAULT 0 CHECK (discount_amount >= 0),
    ADD COLUMN paid_amount numeric(19,4) NOT NULL DEFAULT 0,
    ADD CONSTRAINT purchase_invoices_paid_amount_nonnegative_check CHECK (paid_amount >= 0),
    ADD CONSTRAINT purchase_invoices_paid_amount_check CHECK (paid_amount <= amount);

UPDATE purchase_invoices
SET subtotal_amount = amount,
    paid_amount = CASE WHEN status = 'PAID' THEN amount ELSE 0 END;

CREATE TABLE invoice_payments (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    invoice_id uuid NOT NULL REFERENCES purchase_invoices(id) ON DELETE CASCADE,
    amount numeric(19,4) NOT NULL CHECK (amount > 0),
    paid_on date NOT NULL,
    payment_reference varchar(100) NOT NULL,
    note text,
    created_by uuid NOT NULL REFERENCES users(id),
    created_at timestamptz NOT NULL DEFAULT now(),
    correlation_id varchar(128),
    idempotency_key varchar(255) NOT NULL UNIQUE,
    UNIQUE (invoice_id, payment_reference)
);

CREATE INDEX invoice_payments_invoice_created_idx
    ON invoice_payments (invoice_id, created_at DESC);

CREATE SEQUENCE audit_case_code_seq;
CREATE TABLE audit_cases (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id uuid NOT NULL REFERENCES organizations(id),
    case_code varchar(30) NOT NULL UNIQUE,
    title varchar(255) NOT NULL,
    description text NOT NULL,
    severity varchar(20) NOT NULL CHECK (severity IN ('LOW', 'MEDIUM', 'HIGH', 'CRITICAL')),
    status varchar(30) NOT NULL DEFAULT 'OPEN'
        CHECK (status IN ('OPEN', 'IN_REMEDIATION', 'RESOLVED', 'CLOSED')),
    resource_type varchar(80),
    resource_id uuid,
    owner_user_id uuid REFERENCES users(id),
    due_on date,
    resolution text,
    created_by uuid NOT NULL REFERENCES users(id),
    version bigint NOT NULL DEFAULT 1 CHECK (version > 0),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE audit_case_events (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    audit_case_id uuid NOT NULL REFERENCES audit_cases(id) ON DELETE CASCADE,
    event_type varchar(80) NOT NULL,
    from_status varchar(30),
    to_status varchar(30) NOT NULL,
    actor_id uuid NOT NULL REFERENCES users(id),
    comment text,
    correlation_id varchar(128),
    occurred_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX audit_cases_organization_status_due_idx
    ON audit_cases (organization_id, status, due_on);

CREATE TABLE ai_recommendations (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id uuid NOT NULL REFERENCES organizations(id),
    purchase_request_id uuid REFERENCES purchase_requests(id) ON DELETE CASCADE,
    recommendation_type varchar(50) NOT NULL,
    title varchar(255) NOT NULL,
    summary text NOT NULL,
    risk_level varchar(20) NOT NULL CHECK (risk_level IN ('LOW', 'MEDIUM', 'HIGH', 'CRITICAL')),
    evidence jsonb NOT NULL DEFAULT '[]',
    status varchar(20) NOT NULL DEFAULT 'PENDING'
        CHECK (status IN ('PENDING', 'APPROVED', 'REJECTED', 'DISMISSED')),
    fingerprint varchar(128) NOT NULL UNIQUE,
    generated_by uuid NOT NULL REFERENCES users(id),
    generated_at timestamptz NOT NULL DEFAULT now(),
    decided_by uuid REFERENCES users(id),
    decided_at timestamptz,
    decision_comment text,
    version bigint NOT NULL DEFAULT 1 CHECK (version > 0)
);

CREATE INDEX ai_recommendations_organization_status_risk_idx
    ON ai_recommendations (organization_id, status, risk_level, generated_at DESC);

INSERT INTO app_metadata (key, value)
VALUES ('schema_version', '000012_enterprise_completion')
ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value, updated_at = now();
