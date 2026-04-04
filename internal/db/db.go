package db

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"go.etcd.io/bbolt"

	"holetab/internal/model"
)

const bucketLinks = "links"

// Open initialises the bbolt database at the given path, creating it if absent.
// The caller is responsible for calling db.Close().
func Open(path string) (*bbolt.DB, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return nil, fmt.Errorf("create db dir: %w", err)
	}
	db, err := bbolt.Open(path, 0600, nil)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	// Ensure the links bucket exists.
	err = db.Update(func(tx *bbolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(bucketLinks))
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("create bucket: %w", err)
	}

	return db, nil
}

// GetAllLinks returns all links sorted ascending by Position.
func GetAllLinks(db *bbolt.DB) ([]model.Link, error) {
	var links []model.Link

	err := db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(bucketLinks))
		return b.ForEach(func(_, v []byte) error {
			var l model.Link
			if err := json.Unmarshal(v, &l); err != nil {
				return err
			}
			links = append(links, l)
			return nil
		})
	})
	if err != nil {
		return nil, err
	}

	sort.Slice(links, func(i, j int) bool {
		return links[i].Position < links[j].Position
	})
	return links, nil
}

// AddLink inserts a new link. The ID is assigned via NextSequence; Position is
// set to len(existing)+1 so the new link appears last.
func AddLink(db *bbolt.DB, link model.Link) error {
	return db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(bucketLinks))

		id, err := b.NextSequence()
		if err != nil {
			return err
		}
		link.ID = id
		link.Position = b.Stats().KeyN // append after existing entries

		v, err := json.Marshal(link)
		if err != nil {
			return err
		}
		return b.Put(itob(id), v)
	})
}

// UpdateLink overwrites an existing link identified by link.ID.
func UpdateLink(db *bbolt.DB, link model.Link) error {
	return db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(bucketLinks))
		v, err := json.Marshal(link)
		if err != nil {
			return err
		}
		return b.Put(itob(link.ID), v)
	})
}

// DeleteLink removes the link with the given id and recompacts positions
// so there are no gaps.
func DeleteLink(db *bbolt.DB, id uint64) error {
	return db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(bucketLinks))

		if err := b.Delete(itob(id)); err != nil {
			return err
		}

		// Recompact positions to remove gaps.
		return recompactPositions(b)
	})
}

// MoveLink swaps the position of link `id` with its neighbour in direction dir
// ("up" = lower position index, "down" = higher).
func MoveLink(db *bbolt.DB, id uint64, dir string) error {
	return db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(bucketLinks))

		// Load all links to find neighbours.
		var links []model.Link
		if err := b.ForEach(func(_, v []byte) error {
			var l model.Link
			if err := json.Unmarshal(v, &l); err != nil {
				return err
			}
			links = append(links, l)
			return nil
		}); err != nil {
			return err
		}

		sort.Slice(links, func(i, j int) bool { return links[i].Position < links[j].Position })

		// Find the index of the target link.
		idx := -1
		for i, l := range links {
			if l.ID == id {
				idx = i
				break
			}
		}
		if idx == -1 {
			return fmt.Errorf("link %d not found", id)
		}

		var swapIdx int
		switch dir {
		case "up":
			if idx == 0 {
				return nil // already first
			}
			swapIdx = idx - 1
		case "down":
			if idx == len(links)-1 {
				return nil // already last
			}
			swapIdx = idx + 1
		default:
			return fmt.Errorf("invalid direction %q", dir)
		}

		// Swap positions.
		links[idx].Position, links[swapIdx].Position = links[swapIdx].Position, links[idx].Position

		// Persist both.
		for _, l := range []model.Link{links[idx], links[swapIdx]} {
			v, err := json.Marshal(l)
			if err != nil {
				return err
			}
			if err := b.Put(itob(l.ID), v); err != nil {
				return err
			}
		}
		return nil
	})
}

// recompactPositions reassigns sequential 0-based positions to all links
// in their current order. Must be called inside an Update transaction.
func recompactPositions(b *bbolt.Bucket) error {
	var links []model.Link
	if err := b.ForEach(func(_, v []byte) error {
		var l model.Link
		if err := json.Unmarshal(v, &l); err != nil {
			return err
		}
		links = append(links, l)
		return nil
	}); err != nil {
		return err
	}

	sort.Slice(links, func(i, j int) bool { return links[i].Position < links[j].Position })

	for i := range links {
		links[i].Position = i
		v, err := json.Marshal(links[i])
		if err != nil {
			return err
		}
		if err := b.Put(itob(links[i].ID), v); err != nil {
			return err
		}
	}
	return nil
}

// itob encodes a uint64 as an 8-byte big-endian slice (bbolt key ordering).
func itob(v uint64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, v)
	return b
}
