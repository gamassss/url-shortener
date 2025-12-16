CREATE TABLE IF NOT EXISTS urls (
    id           BIGSERIAL   PRIMARY KEY,
    short_code   VARCHAR(20) UNIQUE NOT NULL,
    original_url TEXT        NOT NULL,
    click_count  BIGINT      DEFAULT 0,
    created_at   TIMESTAMP   NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMP   NOT NULL DEFAULT NOW(),
    expires_at   TIMESTAMP,
    is_active    BOOLEAN     DEFAULT TRUE
);

CREATE INDEX IF NOT EXISTS idx_short_code ON urls (short_code);
CREATE INDEX IF NOT EXISTS idx_created_at ON urls (created_at);
CREATE INDEX IF NOT EXISTS idx_is_active ON urls (is_active);
