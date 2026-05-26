package model

type ResultType string

const (
	TypeWeb   ResultType = "web"
	TypeVideo ResultType = "video"
)

type SearchResult struct {
	URL      string
	Title    string
	Desc     string
	Domain   string
	Icon     string
	SiteName string
	Author   string
	Type     ResultType
}
