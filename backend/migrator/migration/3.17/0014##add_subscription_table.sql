CREATE TABLE IF NOT EXISTS subscription (
    workspace   text        NOT NULL REFERENCES workspace(resource_id) PRIMARY KEY,
    -- Stored as SubscriptionPayload (proto/store/store/subscription.proto)
    payload     jsonb       NOT NULL DEFAULT '{}',
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now()
);
