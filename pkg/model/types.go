package model

import sdk "github.com/TrueBlocks/trueblocks-sdk/v6"

type Database struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	DatabaseName string   `json:"databaseName"` // Internal name from DatabaseNames (e.g., "nouns")
	Count        uint64   `json:"count"`        // Total record count
	Sample       string   `json:"sample"`       // Sample record (first record)
	Filtered     string   `json:"filtered"`     // Filter status indicator
	Version      string   `json:"version"`      // Version from CSV (e.g., "v0.1.0")
	Columns      []string `json:"columns"`      // Column names
	Description  string   `json:"description"`  // Database description
	LastUpdated  int64    `json:"lastUpdated"`  // Timestamp
	CacheHit     bool     `json:"cacheHit"`     // Whether loaded from cache
}

type Item struct {
	ID           string `json:"id"`
	DatabaseName string `json:"databaseName"` // Which database this record belongs to
	Index        uint64 `json:"index"`        // Item index/position
	Version      string `json:"version"`      // Version from CSV (e.g., "v0.1.0")
	Value        string `json:"value"`        // The actual record value
	Remainder    string `json:"remainder"`    // Remaining CSV columns joined
}

type Generator struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func SortItems(items []Item, sortSpec sdk.SortSpec) error {
	_ = items
	_ = sortSpec
	return nil
}
