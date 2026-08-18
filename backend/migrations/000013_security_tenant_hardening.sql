-- Remove an empty prototype table that existed in an earlier local form of
-- migration 000012 but was never backed by application code.
DROP TABLE IF EXISTS approval_delegations;

ALTER TABLE audit_logs
    ADD COLUMN organization_id uuid REFERENCES organizations(id);

UPDATE audit_logs al
SET organization_id = d.organization_id
FROM users u
JOIN departments d ON d.id = u.department_id
WHERE u.id = al.actor_id
  AND al.organization_id IS NULL;

ALTER TABLE audit_logs
    ALTER COLUMN organization_id SET NOT NULL;

CREATE INDEX audit_logs_organization_occurred_idx
    ON audit_logs (organization_id, occurred_at DESC, id DESC);

INSERT INTO app_metadata (key, value)
VALUES ('schema_version', '000013_security_tenant_hardening')
ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value, updated_at = now();
