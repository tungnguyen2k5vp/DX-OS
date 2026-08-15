CREATE TABLE outbox_events (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    event_type varchar(100) NOT NULL,
    resource_type varchar(100) NOT NULL,
    resource_id uuid NOT NULL,
    organization_id uuid NOT NULL REFERENCES organizations(id),
    department_id uuid REFERENCES departments(id),
    recipient_user_id uuid REFERENCES users(id),
    recipient_role varchar(100),
    actor_id uuid NOT NULL REFERENCES users(id),
    title varchar(255) NOT NULL,
    body text NOT NULL,
    payload jsonb NOT NULL DEFAULT '{}'::jsonb,
    status varchar(20) NOT NULL DEFAULT 'PENDING'
        CHECK (status IN ('PENDING', 'PROCESSED', 'DEAD')),
    attempts integer NOT NULL DEFAULT 0 CHECK (attempts >= 0),
    next_attempt_at timestamptz NOT NULL DEFAULT now(),
    processed_at timestamptz,
    last_error text,
    created_at timestamptz NOT NULL DEFAULT now(),
    CHECK (recipient_user_id IS NOT NULL OR recipient_role IS NOT NULL),
    CHECK (recipient_role IS NULL OR length(btrim(recipient_role)) > 0)
);

CREATE INDEX outbox_events_pending_idx
    ON outbox_events (next_attempt_at, created_at, id)
    WHERE status = 'PENDING';

CREATE TABLE user_notifications (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    outbox_event_id uuid NOT NULL UNIQUE REFERENCES outbox_events(id),
    event_type varchar(100) NOT NULL,
    resource_type varchar(100) NOT NULL,
    resource_id uuid NOT NULL,
    organization_id uuid NOT NULL REFERENCES organizations(id),
    department_id uuid REFERENCES departments(id),
    recipient_user_id uuid REFERENCES users(id),
    recipient_role varchar(100),
    actor_id uuid NOT NULL REFERENCES users(id),
    title varchar(255) NOT NULL,
    body text NOT NULL,
    created_at timestamptz NOT NULL,
    CHECK (recipient_user_id IS NOT NULL OR recipient_role IS NOT NULL)
);

CREATE INDEX user_notifications_audience_idx
    ON user_notifications (organization_id, recipient_role, department_id, created_at DESC);
CREATE INDEX user_notifications_recipient_idx
    ON user_notifications (recipient_user_id, created_at DESC)
    WHERE recipient_user_id IS NOT NULL;

CREATE TABLE notification_reads (
    notification_id uuid NOT NULL REFERENCES user_notifications(id) ON DELETE CASCADE,
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    read_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (notification_id, user_id)
);

INSERT INTO app_metadata (key, value)
VALUES ('schema_version', '000009_notification_outbox')
ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value, updated_at = now();
