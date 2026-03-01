CREATE TABLE IF NOT EXISTS events
(
    id          UUID PRIMARY KEY         DEFAULT gen_random_uuid(),
    timestamp   TIMESTAMP WITH TIME ZONE NOT NULL,
    entity_id   UUID                     NOT NULL,
    entity_name VARCHAR(255),
    action      VARCHAR(255),
    user_id     VARCHAR(255),
    ip_address  VARCHAR(255),
    user_agent  TEXT,
    tenant      VARCHAR(255),
    changes     JSONB
);

CREATE INDEX IF NOT EXISTS idx_events_timestamp ON events (timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_events_entity_id ON events (entity_id);
CREATE INDEX IF NOT EXISTS idx_events_entity_name ON events (entity_name);
CREATE INDEX IF NOT EXISTS idx_events_action ON events (action);
CREATE INDEX IF NOT EXISTS idx_events_user_id ON events (user_id);
CREATE INDEX IF NOT EXISTS idx_events_tenant ON events (tenant);
CREATE INDEX IF NOT EXISTS idx_events_changes ON events USING GIN (changes);
