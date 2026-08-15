CREATE TABLE suppliers (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id uuid NOT NULL REFERENCES organizations(id),
    code varchar(50) NOT NULL,
    name varchar(255) NOT NULL,
    tax_code varchar(50),
    contact_name varchar(255),
    email varchar(255),
    phone varchar(50),
    status varchar(20) NOT NULL DEFAULT 'ACTIVE'
        CHECK (status IN ('ACTIVE', 'INACTIVE')),
    risk_level varchar(20) NOT NULL DEFAULT 'LOW'
        CHECK (risk_level IN ('LOW', 'MEDIUM', 'HIGH')),
    version bigint NOT NULL DEFAULT 1 CHECK (version > 0),
    created_by uuid NOT NULL REFERENCES users(id),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (organization_id, code)
);

CREATE UNIQUE INDEX suppliers_organization_tax_code_uq
    ON suppliers (organization_id, tax_code)
    WHERE tax_code IS NOT NULL;

CREATE INDEX suppliers_organization_status_name_idx
    ON suppliers (organization_id, status, name);

CREATE SEQUENCE purchase_order_code_seq;

CREATE TABLE purchase_orders (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id uuid NOT NULL REFERENCES organizations(id),
    purchase_request_id uuid NOT NULL UNIQUE REFERENCES purchase_requests(id),
    supplier_id uuid NOT NULL REFERENCES suppliers(id),
    order_code varchar(30) NOT NULL UNIQUE,
    external_reference varchar(100),
    expected_delivery_on date NOT NULL,
    actual_delivery_on date,
    status varchar(20) NOT NULL DEFAULT 'ORDERED'
        CHECK (status IN ('ORDERED', 'RECEIVED')),
    note varchar(2000),
    version bigint NOT NULL DEFAULT 1 CHECK (version > 0),
    ordered_by uuid NOT NULL REFERENCES users(id),
    received_by uuid REFERENCES users(id),
    ordered_at timestamptz NOT NULL DEFAULT now(),
    received_at timestamptz,
    updated_at timestamptz NOT NULL DEFAULT now(),
    idempotency_key varchar(255) NOT NULL UNIQUE,
    CHECK (
        (status = 'ORDERED' AND actual_delivery_on IS NULL AND received_by IS NULL AND received_at IS NULL)
        OR
        (status = 'RECEIVED' AND actual_delivery_on IS NOT NULL AND received_by IS NOT NULL AND received_at IS NOT NULL)
    )
);

CREATE INDEX purchase_orders_organization_status_delivery_idx
    ON purchase_orders (organization_id, status, expected_delivery_on);

INSERT INTO app_metadata (key, value)
VALUES ('schema_version', '000008_procurement_operations')
ON CONFLICT (key) DO UPDATE
SET value = EXCLUDED.value,
    updated_at = now();
