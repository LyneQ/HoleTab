package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

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
		CREATE TABLE IF NOT EXISTS links (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			type TEXT,
			name TEXT,
			href TEXT,
			img TEXT,
			position INTEGER
		);
		CREATE TABLE IF NOT EXISTS config (
			key TEXT PRIMARY KEY,
			value TEXT
		);
	`)
	if err != nil {
		return nil, fmt.Errorf("create tables: %w", err)
	}

	return db, nil
}

// GetAllLinks returns all links sorted ascending by Position.
func GetAllLinks(db *sql.DB) ([]model.Link, error) {
	rows, err := db.Query("SELECT id, type, name, href, img, position FROM links ORDER BY position ASC")
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

// AddLinks inserts multiple links.
func AddLinks(db *sql.DB, links []model.Link) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var nextPos int
	err = tx.QueryRow("SELECT COALESCE(MAX(position), -1) + 1 FROM links").Scan(&nextPos)
	if err != nil {
		return err
	}

	for _, link := range links {
		res, err := tx.Exec("INSERT INTO links (type, name, href, img, position) VALUES (?, ?, ?, ?, ?)",
			link.Type, link.Name, link.Href, link.Img, nextPos)
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

// AddLink inserts a new link.
func AddLink(db *sql.DB, link model.Link) error {
	return AddLinks(db, []model.Link{link})
}

// UpdateLink overwrites an existing link identified by link.ID.
func UpdateLink(db *sql.DB, link model.Link) error {
	_, err := db.Exec("UPDATE links SET type = ?, name = ?, href = ?, img = ?, position = ? WHERE id = ?",
		link.Type, link.Name, link.Href, link.Img, link.Position, link.ID)
	return err
}

// DeleteLink removes the link with the given id and recompacts positions
// so there are no gaps.
func DeleteLink(db *sql.DB, id uint64) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec("DELETE FROM links WHERE id = ?", id)
	if err != nil {
		return err
	}

	// Recompact positions.
	if err := recompactPositions(tx); err != nil {
		return err
	}

	return tx.Commit()
}

// MoveLink swaps the position of link `id` with its neighbour in direction dir
// ("up" = lower position index, "down" = higher).
func MoveLink(db *sql.DB, id uint64, dir string) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var currentPos int
	err = tx.QueryRow("SELECT position FROM links WHERE id = ?", id).Scan(&currentPos)
	if err != nil {
		return err
	}

	var swapID uint64
	var swapPos int
	var query string
	if dir == "up" {
		query = "SELECT id, position FROM links WHERE position < ? ORDER BY position DESC LIMIT 1"
	} else if dir == "down" {
		query = "SELECT id, position FROM links WHERE position > ? ORDER BY position ASC LIMIT 1"
	} else {
		return fmt.Errorf("invalid direction %q", dir)
	}

	err = tx.QueryRow(query, currentPos).Scan(&swapID, &swapPos)
	if err == sql.ErrNoRows {
		return nil // Nothing to move
	}
	if err != nil {
		return err
	}

	_, err = tx.Exec("UPDATE links SET position = ? WHERE id = ?", swapPos, id)
	if err != nil {
		return err
	}
	_, err = tx.Exec("UPDATE links SET position = ? WHERE id = ?", currentPos, swapID)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// ReorderLinks updates the positions of all links based on the provided ID order.
func ReorderLinks(db *sql.DB, ids []uint64) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for i, id := range ids {
		_, err := tx.Exec("UPDATE links SET position = ? WHERE id = ?", i, id)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// recompactPositions reassigns sequential 0-based positions to all links
// in their current order. Must be called inside an Update transaction.
func recompactPositions(tx *sql.Tx) error {
	rows, err := tx.Query("SELECT id FROM links ORDER BY position ASC")
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
		_, err := tx.Exec("UPDATE links SET position = ? WHERE id = ?", i, id)
		if err != nil {
			return err
		}
	}
	return nil
}

// ResetLinks deletes all links from the database.
func ResetLinks(db *sql.DB) error {
	_, err := db.Exec("DELETE FROM links")
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
