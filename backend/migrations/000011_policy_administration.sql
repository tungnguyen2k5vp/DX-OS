ALTER TABLE reporting.sla_policies
    ADD COLUMN version bigint NOT NULL DEFAULT 1 CHECK (version > 0),
    ADD COLUMN updated_by uuid REFERENCES users(id);

ALTER TABLE attachment_rules
    ADD COLUMN version bigint NOT NULL DEFAULT 1 CHECK (version > 0),
    ADD COLUMN updated_by uuid REFERENCES users(id);

INSERT INTO app_metadata (key, value)
VALUES ('schema_version', '000011_policy_administration')
ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value, updated_at = now();
