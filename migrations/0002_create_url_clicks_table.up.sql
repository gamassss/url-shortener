CREATE TABLE IF NOT EXISTS url_clicks (
    id           BIGSERIAL PRIMARY KEY,
    url_id       BIGINT NOT NULL REFERENCES urls(id) ON DELETE CASCADE,
    clicked_at   TIMESTAMP NOT NULL DEFAULT NOW(),
    user_agent   TEXT,
    referer      TEXT,
    ip_address   VARCHAR(45),
    country_code VARCHAR(2),
    device_type  VARCHAR(20),

    CONSTRAINT fk_url_clicks_url_id FOREIGN KEY (url_id) REFERENCES urls(id)
);

CREATE INDEX IF NOT EXISTS idx_url_clicks_url_id ON url_clicks(url_id);
CREATE INDEX IF NOT EXISTS idx_url_clicks_clicked_at ON url_clicks(clicked_at);
CREATE INDEX IF NOT EXISTS idx_url_clicks_url_id_clicked_at ON url_clicks(url_id, clicked_at DESC);

CREATE OR REPLACE FUNCTION update_url_click_count()
RETURNS TRIGGER AS $$
BEGIN
    UPDATE urls
    SET click_count = click_count + 1,
        updated_at = NOW()
    WHERE id = NEW.url_id;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_update_click_count
    AFTER INSERT ON url_clicks
    FOR EACH ROW
    EXECUTE FUNCTION update_url_click_count();