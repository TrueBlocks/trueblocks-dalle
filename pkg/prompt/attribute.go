package prompt

import "strconv"

// Attribute represents a data attribute with metadata used for prompt generation.
type Attribute struct {
	Database string  `json:"database"`
	Name     string  `json:"name"`
	Bytes    string  `json:"bytes"`
	Number   uint64  `json:"number"`
	Factor   float64 `json:"factor"`
	Count    uint64  `json:"count"`
	Selector uint64  `json:"selector"`
	Value    string  `json:"value"`
}

// DatabaseNames lists the databases used to derive attributes from a seed.
var DatabaseNames = []string{
	"adverbs",
	"adjectives",
	"nouns",
	"emotions",
	"occupations",
	"actions",
	"artstyles",
	"artstyles",
	"litstyles",
	"colors",
	"colors",
	"colors",
	"viewpoints",
	"gazes",
	"compositions",
	"places",
	"tropes",
}

var attributeNames = []string{
	"adverb",
	"adjective",
	"noun",
	"emotion",
	"occupation",
	"action",
	"artStyle1",
	"artStyle2",
	"litStyle",
	"color1",
	"color2",
	"color3",
	"viewpoint",
	"gaze",
	"composition",
	"place",
	"trope",
}

// AttributeNames returns the list of attribute names (for test and compatibility).
func AttributeNames() []string { return attributeNames }

// NewAttribute constructs an Attribute from database info and a byte string.
func NewAttribute(dbs map[string][]string, index int, bytes string) Attribute {
	number, err := strconv.ParseUint(bytes, 16, 64)
	if err != nil {
		number = 0
	}
	attr := Attribute{
		Database: DatabaseNames[index],
		Name:     attributeNames[index],
		Bytes:    bytes,
		Number:   number,
		Factor:   float64(number) / float64(1<<24),
		Count:    8,
		Selector: 0,
		Value:    "",
	}
	attr.Count = uint64(len(dbs[attr.Database]))
	attr.Selector = uint64(float64(attr.Count) * attr.Factor)
	attr.Value = dbs[attr.Database][attr.Selector]
	return attr
}
