package dalle

import (
	"github.com/TrueBlocks/trueblocks-core/src/apps/chifra/pkg/base"
)

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

func NewAttribute(databases map[string][]string, index int, bytes string) Attribute {
	attr := Attribute{
		Database: DatabaseNames[index],
		Name:     attributeNames[index],
		Bytes:    bytes,
		Number:   base.MustParseUint64("0x" + bytes),
		Factor:   float64(base.MustParseUint64("0x"+bytes)) / float64(1<<24),
		Count:    8,
		Selector: 0,
		Value:    "",
	}
	attr.Count = uint64(len(databases[attr.Database]))
	attr.Selector = uint64(float64(attr.Count) * attr.Factor)
	attr.Value = databases[attr.Database][attr.Selector]
	return attr
}
