CREATE OR REPLACE FUNCTION lifeforge.create_user_progress()
RETURNS TRIGGER AS $$
BEGIN
    INSERT INTO lifeforge.user_progress (user_id, level, xp, xp_for_next_level, coins)
    VALUES (NEW.id, 1, 0, 100, 0);
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_user_progress_after_inser
AFTER INSERT ON lifeforge.users
FOR EACH ROW
EXECUTE FUNCTION lifeforge.create_user_progress();