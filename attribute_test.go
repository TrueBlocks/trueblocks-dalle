package dalle

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewAttribute_Basic(t *testing.T) {
	databases := map[string][]string{
		"adverbs":    {"quickly", "slowly", "silently", "loudly"},
		"adjectives": {"red", "blue", "green", "yellow"},
		"nouns":      {"cat", "dog", "bird", "fish"},
	}
	// Use index 0 (adverbs)
	attr := NewAttribute(databases, 0, "01")
	assert.Equal(t, "adverbs", attr.Database)
	assert.Equal(t, "adverb", attr.Name)
	assert.Equal(t, "01", attr.Bytes)
	assert.Equal(t, uint64(len(databases["adverbs"])), attr.Count)
	assert.True(t, attr.Selector < uint64(len(databases["adverbs"])))
	assert.Contains(t, databases["adverbs"], attr.Value)
}

func TestNewAttribute_SelectorBounds(t *testing.T) {
	databases := map[string][]string{
		"adverbs": {"a", "b"},
	}
	attr := NewAttribute(databases, 0, "FFFFFF") // large value, Factor ~1
	assert.Equal(t, uint64(len(databases["adverbs"])), attr.Count)
	assert.True(t, attr.Selector < uint64(len(databases["adverbs"])))
	assert.Contains(t, databases["adverbs"], attr.Value)
}

func TestNewAttribute_DifferentIndexes(t *testing.T) {
	databases := map[string][]string{
		"adverbs":    {"quickly"},
		"adjectives": {"red"},
		"nouns":      {"cat"},
	}
	for idx, db := range []string{"adverbs", "adjectives", "nouns"} {
		attr := NewAttribute(databases, idx, "01")
		assert.Equal(t, db, attr.Database)
		assert.Equal(t, attributeNames[idx], attr.Name)
		assert.Contains(t, databases[db], attr.Value)
	}
}

func TestNewAttribute_SelectorEdge(t *testing.T) {
	databases := map[string][]string{
		"adverbs": {"a", "b", "c"},
	}
	attr := NewAttribute(databases, 0, "000000") // Factor = 0
	assert.Equal(t, uint64(0), attr.Selector)
	attr2 := NewAttribute(databases, 0, "FFFFFF") // Factor ~1
	assert.True(t, attr2.Selector < uint64(len(databases["adverbs"])))
}
