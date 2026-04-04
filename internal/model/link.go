package model

// Link represents a bookmark entry stored in bbolt.
type Link struct {
	ID       uint64 `json:"id"`
	Name     string `json:"name"`
	Href     string `json:"href"`
	Img      string `json:"img"`      // favicon URL or empty string
	Position int    `json:"position"` // display order, 0-based
}
