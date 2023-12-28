package httpserver

type Book struct {
	BookID int32 `json:"book_id"`
	BookToAdd
}

type BookToAdd struct {
	Title       string  `json:"title"`
	Author      string  `json:"author"`
	Price       float64 `json:"price"`
	Description string  `json:"description"`
	AuthorBio   string  `json:"author_bio"`
}
