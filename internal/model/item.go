package model

// Item is the domain model for a todo entry.
// Kept minimal on purpose; it’s easy to evolve.
type Item struct {
	Title string `json:"title"`
	Done  bool   `json:"done"`
}
