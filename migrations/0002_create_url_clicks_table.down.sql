DROP TRIGGER IF EXISTS trigger_update_click_count ON url_clicks;
DROP FUNCTION IF EXISTS update_url_click_count();
DROP TABLE IF EXISTS url_clicks;