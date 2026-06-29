package db

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
	_ "modernc.org/sqlite"

	"holetab/internal/model"
)

// Open initialises the sqlite database at the given path, creating it if absent.
// The caller is responsible for calling db.Close().
func Open(path string) (*sql.DB, error) {

	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("get home dir: %w", err)
		}
		path = filepath.Join(home, path[2:])
	}

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return nil, fmt.Errorf("create db dir: %w", err)
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	// Create tables if they don't exist.
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT UNIQUE,
			password_hash TEXT
		);
		CREATE TABLE IF NOT EXISTS sessions (
			token TEXT PRIMARY KEY,
			user_id INTEGER,
			expires_at DATETIME,
			FOREIGN KEY(user_id) REFERENCES users(id)
		);
		CREATE TABLE IF NOT EXISTS links (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER,
			type TEXT,
			name TEXT,
			href TEXT,
			img TEXT,
			position INTEGER,
			FOREIGN KEY(user_id) REFERENCES users(id)
		);
		CREATE TABLE IF NOT EXISTS config (
			key TEXT PRIMARY KEY,
			value TEXT
		);
	`)
	if err != nil {
		return nil, fmt.Errorf("create tables: %w", err)
	}

	// Migration: Add user_id to links if it doesn't exist
	var hasUserID bool
	err = db.QueryRow("SELECT COUNT(*) FROM pragma_table_info('links') WHERE name='user_id'").Scan(&hasUserID)
	if err == nil && !hasUserID {
		_, _ = db.Exec("ALTER TABLE links ADD COLUMN user_id INTEGER REFERENCES users(id)")
		_, _ = db.Exec("UPDATE links SET user_id = 0 WHERE user_id IS NULL")
	}

	return db, nil
}

// GetAllLinks returns all links for a user sorted ascending by Position.
func GetAllLinks(db *sql.DB, userID uint64) ([]model.Link, error) {
	rows, err := db.Query("SELECT id, type, name, href, img, position FROM links WHERE user_id = ? ORDER BY position ASC", userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var links []model.Link
	for rows.Next() {
		var l model.Link
		if err := rows.Scan(&l.ID, &l.Type, &l.Name, &l.Href, &l.Img, &l.Position); err != nil {
			return nil, err
		}
		links = append(links, l)
	}
	return links, nil
}

// AddLinks inserts multiple links for a user.
func AddLinks(db *sql.DB, userID uint64, links []model.Link) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var nextPos int
	err = tx.QueryRow("SELECT COALESCE(MAX(position), -1) + 1 FROM links WHERE user_id = ?", userID).Scan(&nextPos)
	if err != nil {
		return err
	}

	for _, link := range links {
		res, err := tx.Exec("INSERT INTO links (user_id, type, name, href, img, position) VALUES (?, ?, ?, ?, ?, ?)",
			userID, link.Type, link.Name, link.Href, link.Img, nextPos)
		if err != nil {
			return err
		}
		id, err := res.LastInsertId()
		if err != nil {
			return err
		}
		_ = id // We don't strictly need to return it here based on current API but good to know
		nextPos++
	}

	return tx.Commit()
}

// AddLink inserts a new link for a user.
func AddLink(db *sql.DB, userID uint64, link model.Link) error {
	return AddLinks(db, userID, []model.Link{link})
}

// UpdateLink overwrites an existing link identified by link.ID.
func UpdateLink(db *sql.DB, userID uint64, link model.Link) error {
	_, err := db.Exec("UPDATE links SET type = ?, name = ?, href = ?, img = ?, position = ? WHERE id = ? AND user_id = ?",
		link.Type, link.Name, link.Href, link.Img, link.Position, link.ID, userID)
	return err
}

// DeleteLink removes the link with the given id and recompacts positions
// so there are no gaps.
func DeleteLink(db *sql.DB, userID uint64, id uint64) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec("DELETE FROM links WHERE id = ? AND user_id = ?", id, userID)
	if err != nil {
		return err
	}

	// Recompact positions.
	if err := recompactPositions(tx, userID); err != nil {
		return err
	}

	return tx.Commit()
}

// MoveLink swaps the position of link `id` with its neighbour in direction dir
// ("up" = lower position index, "down" = higher).
func MoveLink(db *sql.DB, userID uint64, id uint64, dir string) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var currentPos int
	err = tx.QueryRow("SELECT position FROM links WHERE id = ? AND user_id = ?", id, userID).Scan(&currentPos)
	if err != nil {
		return err
	}

	var swapID uint64
	var swapPos int
	var query string
	if dir == "up" {
		query = "SELECT id, position FROM links WHERE user_id = ? AND position < ? ORDER BY position DESC LIMIT 1"
	} else if dir == "down" {
		query = "SELECT id, position FROM links WHERE user_id = ? AND position > ? ORDER BY position ASC LIMIT 1"
	} else {
		return fmt.Errorf("invalid direction %q", dir)
	}

	err = tx.QueryRow(query, userID, currentPos).Scan(&swapID, &swapPos)
	if err == sql.ErrNoRows {
		return nil // Nothing to move
	}
	if err != nil {
		return err
	}

	_, err = tx.Exec("UPDATE links SET position = ? WHERE id = ? AND user_id = ?", swapPos, id, userID)
	if err != nil {
		return err
	}
	_, err = tx.Exec("UPDATE links SET position = ? WHERE id = ? AND user_id = ?", currentPos, swapID, userID)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// ReorderLinks updates the positions of all links based on the provided ID order.
func ReorderLinks(db *sql.DB, userID uint64, ids []uint64) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for i, id := range ids {
		_, err := tx.Exec("UPDATE links SET position = ? WHERE id = ? AND user_id = ?", i, id, userID)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// recompactPositions reassigns sequential 0-based positions to all links
// in their current order. Must be called inside an Update transaction.
func recompactPositions(tx *sql.Tx, userID uint64) error {
	rows, err := tx.Query("SELECT id FROM links WHERE user_id = ? ORDER BY position ASC", userID)
	if err != nil {
		return err
	}
	defer rows.Close()

	var ids []uint64
	for rows.Next() {
		var id uint64
		if err := rows.Scan(&id); err != nil {
			return err
		}
		ids = append(ids, id)
	}

	for i, id := range ids {
		_, err := tx.Exec("UPDATE links SET position = ? WHERE id = ? AND user_id = ?", i, id, userID)
		if err != nil {
			return err
		}
	}
	return nil
}

// ResetLinks deletes all links for a user from the database.
func ResetLinks(db *sql.DB, userID uint64) error {
	_, err := db.Exec("DELETE FROM links WHERE user_id = ?", userID)
	return err
}

func CreateUser(db *sql.DB, username, password string) (uint64, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return 0, err
	}
	res, err := db.Exec("INSERT INTO users (username, password_hash) VALUES (?, ?)", username, string(hash))
	if err != nil {
		return 0, err
	}
	id, _ := res.LastInsertId()
	return uint64(id), nil
}

func GetUserByUsername(db *sql.DB, username string) (*model.User, error) {
	var u model.User
	err := db.QueryRow("SELECT id, username, password_hash FROM users WHERE username = ?", username).Scan(&u.ID, &u.Username, &u.PasswordHash)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func AuthenticateUser(db *sql.DB, username, password string) (*model.User, error) {
	u, err := GetUserByUsername(db, username)
	if err != nil {
		return nil, err
	}
	err = bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password))
	if err != nil {
		return nil, err
	}
	return u, nil
}

func CreateSession(db *sql.DB, userID uint64) (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	token := hex.EncodeToString(b)
	expiresAt := time.Now().Add(24 * time.Hour)
	_, err := db.Exec("INSERT INTO sessions (token, user_id, expires_at) VALUES (?, ?, ?)", token, userID, expiresAt)
	return token, err
}

func GetUserByToken(db *sql.DB, token string) (*model.User, error) {
	var u model.User
	err := db.QueryRow(`
		SELECT u.id, u.username 
		FROM users u 
		JOIN sessions s ON u.id = s.user_id 
		WHERE s.token = ? AND s.expires_at > ?
	`, token, time.Now()).Scan(&u.ID, &u.Username)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func DeleteSession(db *sql.DB, token string) error {
	_, err := db.Exec("DELETE FROM sessions WHERE token = ?", token)
	return err
}

// GetConfig retrieves a string value from the config table.
func GetConfig(db *sql.DB, key string) (string, error) {
	var value string
	err := db.QueryRow("SELECT value FROM config WHERE key = ?", key).Scan(&value)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return value, err
}

// SetConfig stores a string value in the config table.
func SetConfig(db *sql.DB, key string, value string) error {
	_, err := db.Exec("INSERT INTO config (key, value) VALUES (?, ?) ON CONFLICT(key) DO UPDATE SET value = excluded.value", key, value)
	return err
}
