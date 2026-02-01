CREATE TABLE IF NOT EXISTS events
(
    id             UUID PRIMARY KEY         DEFAULT gen_random_uuid(),
    timestamp      TIMESTAMP WITH TIME ZONE NOT NULL,
    user_name      VARCHAR(255),
    category       VARCHAR(255),
    action         VARCHAR(255),
    document_name  VARCHAR(255),
    project        VARCHAR(255),
    environment    VARCHAR(255),
    tenant         VARCHAR(255),
    correlation_id VARCHAR(255),
    trace_id       VARCHAR(255),
    details        JSONB,
    created_at     TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_events_timestamp ON events (timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_events_user ON events (user_name);
CREATE INDEX IF NOT EXISTS idx_events_category ON events (category);
CREATE INDEX IF NOT EXISTS idx_events_action ON events (action);
CREATE INDEX IF NOT EXISTS idx_events_document ON events (document_name);
CREATE INDEX IF NOT EXISTS idx_events_project ON events (project);
CREATE INDEX IF NOT EXISTS idx_events_environment ON events (environment);
CREATE INDEX IF NOT EXISTS idx_events_tenant ON events (tenant);
CREATE INDEX IF NOT EXISTS idx_events_correlation_id ON events (correlation_id);
CREATE INDEX IF NOT EXISTS idx_events_trace_id ON events (trace_id);
CREATE INDEX IF NOT EXISTS idx_events_details ON events USING GIN (details);
