-- Удаляем триггеры
DROP TRIGGER IF EXISTS update_user_stats_updated_at ON user_stats;

-- Удаляем функцию
DROP FUNCTION IF EXISTS update_updated_at_column();

-- Удаляем индексы
DROP INDEX IF EXISTS idx_user_stats_user_id;
DROP INDEX IF EXISTS idx_completed_tasks_user_id;
DROP INDEX IF EXISTS idx_completed_tasks_event_id;

-- Удаляем таблицы (в правильном порядке из-за зависимостей)
DROP TABLE IF EXISTS completed_tasks;
DROP TABLE IF EXISTS user_stats;
DROP TABLE IF EXISTS user_progress;
DROP TABLE IF EXISTS users;