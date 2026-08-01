CREATE TABLE attachment_rules (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id uuid NOT NULL REFERENCES organizations(id),
    currency char(3) NOT NULL,
    threshold_amount numeric(19, 4) NOT NULL CHECK (threshold_amount >= 0),
    required_document_type varchar(32) NOT NULL DEFAULT 'QUOTATION'
        CHECK (required_document_type IN ('QUOTATION', 'SPECIFICATION', 'CONTRACT', 'OTHER')),
    active boolean NOT NULL DEFAULT true,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (organization_id, currency)
);

CREATE TABLE purchase_request_attachments (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    purchase_request_id uuid NOT NULL REFERENCES purchase_requests(id) ON DELETE CASCADE,
    document_type varchar(32) NOT NULL
        CHECK (document_type IN ('QUOTATION', 'SPECIFICATION', 'CONTRACT', 'OTHER')),
    original_name varchar(255) NOT NULL,
    content_type varchar(128) NOT NULL,
    size_bytes bigint NOT NULL CHECK (size_bytes > 0 AND size_bytes <= 10485760),
    checksum_sha256 char(64) NOT NULL,
    storage_path varchar(1000) NOT NULL UNIQUE,
    storage_etag varchar(255),
    uploaded_by uuid NOT NULL REFERENCES users(id),
    status varchar(16) NOT NULL DEFAULT 'UPLOADING'
        CHECK (status IN ('UPLOADING', 'ACTIVE', 'DELETING', 'DELETED')),
    uploaded_at timestamptz,
    deleted_at timestamptz,
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_purchase_request_attachments_active
    ON purchase_request_attachments (purchase_request_id, uploaded_at DESC)
    WHERE status = 'ACTIVE';

INSERT INTO attachment_rules (
    organization_id,
    currency,
    threshold_amount,
    required_document_type
)
SELECT id, 'VND', 20000000, 'QUOTATION'
FROM organizations
WHERE code = 'DX-OS'
ON CONFLICT (organization_id, currency) DO NOTHING;

INSERT INTO app_metadata (key, value)
VALUES ('schema_version', '000006_purchase_request_attachments')
ON CONFLICT (key) DO UPDATE
SET value = EXCLUDED.value,
    updated_at = now();
