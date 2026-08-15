CREATE TABLE purchase_invoices (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id uuid NOT NULL REFERENCES organizations(id),
    purchase_order_id uuid NOT NULL UNIQUE REFERENCES purchase_orders(id),
    invoice_number varchar(100) NOT NULL,
    issued_on date NOT NULL,
    due_on date NOT NULL,
    amount numeric(19,4) NOT NULL CHECK (amount > 0),
    currency char(3) NOT NULL CHECK (currency ~ '^[A-Z]{3}$'),
    status varchar(20) NOT NULL DEFAULT 'RECORDED'
        CHECK (status IN ('RECORDED', 'VERIFIED', 'DISPUTED', 'PAID')),
    note text,
    version bigint NOT NULL DEFAULT 1 CHECK (version > 0),
    created_by uuid NOT NULL REFERENCES users(id),
    verified_by uuid REFERENCES users(id),
    disputed_by uuid REFERENCES users(id),
    paid_by uuid REFERENCES users(id),
    payment_reference varchar(100),
    paid_on date,
    idempotency_key varchar(255) NOT NULL UNIQUE,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (organization_id, invoice_number),
    CHECK (due_on >= issued_on),
    CHECK (
        (status = 'PAID' AND paid_by IS NOT NULL AND paid_on IS NOT NULL AND payment_reference IS NOT NULL)
        OR status <> 'PAID'
    )
);

CREATE INDEX purchase_invoices_status_due_idx
    ON purchase_invoices (organization_id, status, due_on);

CREATE TABLE invoice_events (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    invoice_id uuid NOT NULL REFERENCES purchase_invoices(id) ON DELETE CASCADE,
    event_type varchar(100) NOT NULL,
    from_status varchar(20),
    to_status varchar(20) NOT NULL,
    actor_id uuid NOT NULL REFERENCES users(id),
    actor_roles text[] NOT NULL DEFAULT '{}',
    comment text,
    correlation_id varchar(128),
    idempotency_key varchar(255) UNIQUE,
    occurred_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX invoice_events_invoice_time_idx
    ON invoice_events (invoice_id, occurred_at DESC, id DESC);

INSERT INTO app_metadata (key, value)
VALUES ('schema_version', '000010_invoice_payment')
ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value, updated_at = now();
