CREATE TABLE budget_adjustments (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    budget_allocation_id uuid NOT NULL REFERENCES budget_allocations(id),
    previous_amount numeric(19,4) NOT NULL CHECK (previous_amount >= 0),
    adjusted_amount numeric(19,4) NOT NULL CHECK (adjusted_amount >= 0),
    reason varchar(1000) NOT NULL,
    actor_id uuid NOT NULL REFERENCES users(id),
    correlation_id varchar(128),
    idempotency_key varchar(255) NOT NULL UNIQUE,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX budget_adjustments_allocation_created_idx
    ON budget_adjustments (budget_allocation_id, created_at DESC, id DESC);
CREATE INDEX budget_reservations_allocation_updated_idx
    ON budget_reservations (budget_allocation_id, updated_at DESC, id DESC);

INSERT INTO app_metadata (key, value)
VALUES ('schema_version', 'procurement-mvp-budget-management')
ON CONFLICT (key) DO UPDATE
SET value = EXCLUDED.value,
    updated_at = now();
