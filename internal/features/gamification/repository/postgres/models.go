package gamification_repository

type StatModel struct {
	ID          int     `db:"id"`
	UserID      int     `db:"user_id"`
	Title       string  `db:"title"`
	Description *string `db:"description"`
	Value       int     `db:"value"`
}
