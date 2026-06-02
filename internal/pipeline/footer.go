package pipeline

// FooterData holds footer configuration for injection into HTML.
type FooterData struct {
	Position       string // "left", "center", "right" (default: "right")
	ShowPageNumber bool
	Date           string
	Status         string
	Text           string
	DocumentID     string // Document reference number
}
