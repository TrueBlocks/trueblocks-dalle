package sortspec

type SortOrder string

const (
	Asc SortOrder = "asc"
	Dec SortOrder = "dec"
)

type SortSpec struct {
	Fields []string    `json:"fields,omitempty"`
	Order  []SortOrder `json:"order,omitempty"`
}
