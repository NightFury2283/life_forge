package storage

import (
    "context"
    "fmt"
    "life_forge/internal/models"
    "log"
    "github.com/jackc/pgx/v5/pgxpool"
)

type UserStorage struct {
    pool *pgxpool.Pool
}

func NewUserStorage(pool *pgxpool.Pool) *UserStorage {
    return &UserStorage{pool: pool}
}

//получаем или создаём пользователя
func (s *UserStorage) GetOrCreateUserByGoogleID(ctx context.Context, googleID, email, name string) (*models.User, error) {
    var user models.User
    
    // Пытаемся найти существующего
    query := `SELECT id, google_id, email, name, created_at FROM users WHERE google_id = $1`
    err := s.pool.QueryRow(ctx, query, googleID).Scan(
        &user.ID, &user.GoogleID, &user.Email, &user.Name, &user.CreatedAt,
    )
    
    if err == nil {
        // Пользователь найден, обновляем данные если изменились
        if email != "" && email != user.Email {
            updateQuery := `UPDATE users SET email = $1, name = $2 WHERE id = $3`
            s.pool.Exec(ctx, updateQuery, email, name, user.ID)
            user.Email = email
            user.Name = name
        }
        return &user, nil
    }
    
    // Создаём нового пользователя
    insertQuery := `INSERT INTO users (google_id, email, name) VALUES ($1, $2, $3) RETURNING id, created_at`
    err = s.pool.QueryRow(ctx, insertQuery, googleID, email, name).Scan(&user.ID, &user.CreatedAt)
    if err != nil {
        return nil, fmt.Errorf("failed to create user: %w", err)
    }
    
    user.GoogleID = googleID
    user.Email = email
    user.Name = name
    
    // Создаём запись прогресса
    progressQuery := `INSERT INTO user_progress (user_id, level, xp, xp_for_next_level) VALUES ($1, 1, 0, 100)`
    _, err = s.pool.Exec(ctx, progressQuery, user.ID)
    if err != nil {
        log.Printf("Warning: failed to create progress for user %d: %v", user.ID, err)
    }
    
    log.Printf("✅ New user created: %s (ID: %d)", name, user.ID)
    return &user, nil
}

//получить прогресс пользователя
func (s *UserStorage) GetProgress(ctx context.Context, userID int) (*models.UserProgress, error) {
    var progress models.UserProgress
    query := `SELECT id, user_id, level, xp, xp_for_next_level, updated_at FROM user_progress WHERE user_id = $1`
    err := s.pool.QueryRow(ctx, query, userID).Scan(
        &progress.ID, &progress.UserID, &progress.Level, 
        &progress.XP, &progress.XPForNextLevel, &progress.UpdatedAt,
    )
    if err != nil {
        return nil, err
    }
    return &progress, nil
}

//добавить опыт и обновить уровень
func (s *UserStorage) AddXP(ctx context.Context, userID int, xpAmount int) error {
    // Получаем текущий прогресс
    progress, err := s.GetProgress(ctx, userID)
    if err != nil {
        return err
    }
    
    newXP := progress.XP + xpAmount
    newLevel := progress.Level
    xpForNext := progress.XPForNextLevel
    
    // Повышаем уровень
    for newXP >= xpForNext {
        newXP -= xpForNext
        newLevel++
        xpForNext = 100 + (newLevel-1)*50 // 100, 150, 200, 250...
    }
    
    updateQuery := `UPDATE user_progress SET xp = $1, level = $2, xp_for_next_level = $3, updated_at = CURRENT_TIMESTAMP WHERE user_id = $4`
    _, err = s.pool.Exec(ctx, updateQuery, newXP, newLevel, xpForNext, userID)
    return err
}

//получить все характеристики пользователя
func (s *UserStorage) GetStats(ctx context.Context, userID int) ([]models.UserStat, error) {
    query := `SELECT id, user_id, name, description, value, created_at, updated_at FROM user_stats WHERE user_id = $1 ORDER BY id`
    rows, err := s.pool.Query(ctx, query, userID)
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    
    var stats []models.UserStat
    for rows.Next() {
        var stat models.UserStat
        err := rows.Scan(&stat.ID, &stat.UserID, &stat.Name, &stat.Description, &stat.Value, &stat.CreatedAt, &stat.UpdatedAt)
        if err != nil {
            return nil, err
        }
        stats = append(stats, stat)
    }
    return stats, nil
}

//создать новую характеристику
func (s *UserStorage) CreateStat(ctx context.Context, userID int, name, description string) (*models.UserStat, error) {
    query := `INSERT INTO user_stats (user_id, name, description, value) VALUES ($1, $2, $3, 0) RETURNING id, created_at, updated_at`
    var stat models.UserStat
    err := s.pool.QueryRow(ctx, query, userID, name, description).Scan(&stat.ID, &stat.CreatedAt, &stat.UpdatedAt)
    if err != nil {
        return nil, err
    }
    stat.UserID = userID
    stat.Name = name
    stat.Description = description
    stat.Value = 0
    return &stat, nil
}

//увеличить или уменьшить значение характеристики
func (s *UserStorage) UpdateStatValue(ctx context.Context, statID int, delta int) error {
    query := `UPDATE user_stats SET value = value + $1, updated_at = CURRENT_TIMESTAMP WHERE id = $2`
    _, err := s.pool.Exec(ctx, query, delta, statID)
    return err
}

//удалить характеристику
func (s *UserStorage) DeleteStat(ctx context.Context, statID int) error {
    query := `DELETE FROM user_stats WHERE id = $1`
    _, err := s.pool.Exec(ctx, query, statID)
    return err
}

//проверяем, выполнена ли задача
func (s *UserStorage) IsTaskCompleted(ctx context.Context, userID int, eventID string) (bool, error) {
    query := `SELECT EXISTS(SELECT 1 FROM completed_tasks WHERE user_id = $1 AND event_id = $2)`
    var exists bool
    err := s.pool.QueryRow(ctx, query, userID, eventID).Scan(&exists)
    return exists, err
}

//отметить задачу выполненной
func (s *UserStorage) CompleteTask(ctx context.Context, userID int, eventID, eventTitle string, xpGained int) error {
    query := `INSERT INTO completed_tasks (user_id, event_id, event_title, xp_gained) VALUES ($1, $2, $3, $4)`
    _, err := s.pool.Exec(ctx, query, userID, eventID, eventTitle, xpGained)
    if err != nil {
        return err
    }
    
    // Добавляем опыт
    return s.AddXP(ctx, userID, xpGained)
}

//получить выполненные задачи
func (s *UserStorage) GetCompletedTasks(ctx context.Context, userID int) ([]models.CompletedTask, error) {
    query := `SELECT id, user_id, event_id, event_title, xp_gained, completed_at FROM completed_tasks WHERE user_id = $1 ORDER BY completed_at DESC LIMIT 50`
    rows, err := s.pool.Query(ctx, query, userID)
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    
    var tasks []models.CompletedTask
    for rows.Next() {
        var task models.CompletedTask
        err := rows.Scan(&task.ID, &task.UserID, &task.EventID, &task.EventTitle, &task.XPGained, &task.CompletedAt)
        if err != nil {
            return nil, err
        }
        tasks = append(tasks, task)
    }
    return tasks, nil
}