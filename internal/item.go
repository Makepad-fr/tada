package internal

// Item is the domain model for a todo entry.
type Item struct {
	Title string `json:"title"`
	Done  bool   `json:"done"`
}
