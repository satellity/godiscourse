package models

import (
	"context"
	"database/sql"
	"fmt"
	"godiscourse/internal/configs"
	"godiscourse/internal/durable"
	"strings"
	"time"

	"github.com/gofrs/uuid"
)

const UsersDDL = `
CREATE TABLE IF NOT EXISTS users (
	user_id               VARCHAR(36) PRIMARY KEY,
	email                 VARCHAR(512),
	username              VARCHAR(64) NOT NULL CHECK (username ~* '^[a-z0-9][a-z0-9_]{3,63}$'),
	nickname              VARCHAR(64) NOT NULL DEFAULT '',
	biography             VARCHAR(2048) NOT NULL DEFAULT '',
	encrypted_password    VARCHAR(1024),
	github_id             VARCHAR(1024) UNIQUE,
	created_at            TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
	updated_at            TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS users_emailx ON users ((LOWER(email)));
CREATE UNIQUE INDEX IF NOT EXISTS users_usernamex ON users ((LOWER(username)));
CREATE INDEX IF NOT EXISTS users_createdx ON users (created_at);
`

const DropUsersDDL = `DROP TABLE IF EXISTS users;`

type User struct {
	UserID            string
	Email             sql.NullString
	Username          string
	Nickname          string
	Biography         string
	EncryptedPassword sql.NullString
	GithubID          sql.NullString
	CreatedAt         time.Time
	UpdatedAt         time.Time

	SessionID string
	IsNew     bool
}

type UserInfo struct {
	Email         string
	Username      string
	Nickname      string
	Biography     string
	Password      string
	SessionSecret string
}

var UserColumns = []string{"user_id", "email", "username", "nickname", "biography", "encrypted_password", "github_id", "created_at", "updated_at"}

func (u *User) Values() []interface{} {
	return []interface{}{u.UserID, u.Email, u.Username, u.Nickname, u.Biography, u.EncryptedPassword, u.GithubID, u.CreatedAt, u.UpdatedAt}
}

// Role of an user, contains admin and member for now.
func (u *User) Role() string {
	if configs.Operators[u.Email.String] {
		return "admin"
	}
	return "member"
}

// Name is nickname or username
func (u *User) Name() string {
	if u.Nickname != "" {
		return u.Nickname
	}
	return u.Username
}

func UserFromRows(row durable.Row) (*User, error) {
	var u User
	err := row.Scan(&u.UserID, &u.Email, &u.Username, &u.Nickname, &u.Biography, &u.EncryptedPassword, &u.GithubID, &u.CreatedAt, &u.UpdatedAt)
	return &u, err
}

func FindUserByID(ctx context.Context, tx *sql.Tx, id string) (*User, error) {
	if _, err := uuid.FromString(id); err != nil {
		return nil, nil
	}

	row := tx.QueryRowContext(ctx, fmt.Sprintf("SELECT %s FROM users WHERE user_id=$1", strings.Join(UserColumns, ",")), id)
	u, err := UserFromRows(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return u, err
}

func FindUserByGithubID(ctx context.Context, tx *sql.Tx, id string) (*User, error) {
	rows, err := tx.QueryContext(ctx, fmt.Sprintf("SELECT %s FROM users WHERE github_id=$1", strings.Join(UserColumns, ",")), id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return nil, err
		}
		return nil, nil
	}
	return UserFromRows(rows)
}
