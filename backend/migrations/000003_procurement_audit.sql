CREATE TABLE audit_logs (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    resource_type varchar(80) NOT NULL,
    resource_id uuid NOT NULL,
    action varchar(80) NOT NULL,
    actor_id uuid NOT NULL REFERENCES users(id),
    actor_roles text[] NOT NULL DEFAULT '{}',
    from_status varchar(40),
    to_status varchar(40),
    correlation_id varchar(128),
    metadata jsonb NOT NULL DEFAULT '{}',
    occurred_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX audit_logs_resource_occurred_idx
    ON audit_logs (resource_type, resource_id, occurred_at DESC, id DESC);
CREATE INDEX audit_logs_actor_occurred_idx
    ON audit_logs (actor_id, occurred_at DESC);
CREATE INDEX process_events_request_timeline_idx
    ON process_events (purchase_request_id, occurred_at DESC, id DESC);

INSERT INTO audit_logs (
    resource_type,
    resource_id,
    action,
    actor_id,
    actor_roles,
    from_status,
    to_status,
    correlation_id,
    metadata,
    occurred_at
)
SELECT
    'purchase_request',
    purchase_request_id,
    event_type,
    actor_id,
    actor_roles,
    from_status,
    to_status,
    correlation_id,
    jsonb_build_object('source', 'process_event_backfill'),
    occurred_at
FROM process_events;

INSERT INTO app_metadata (key, value)
VALUES ('schema_version', 'procurement-mvp-step-6')
ON CONFLICT (key) DO UPDATE
SET value = EXCLUDED.value,
    updated_at = now();
