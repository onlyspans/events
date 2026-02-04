CREATE TABLE IF NOT EXISTS settings
(
    id                    VARCHAR(50) PRIMARY KEY,
    retention_period_days INTEGER                  NOT NULL,
    updated_at            TIMESTAMP WITH TIME ZONE NOT NULL,
    updated_by            VARCHAR(255)             NOT NULL
);

INSERT INTO settings (id, retention_period_days, updated_at, updated_by)
VALUES ('global', 90, CURRENT_TIMESTAMP, 'system')
ON CONFLICT (id) DO NOTHING;
