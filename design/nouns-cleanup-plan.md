# Nouns CSV Data Cleanup Plan

## Current State

`nouns.csv` has 3,514 rows after dedup and sort. Columns: `version,commonName,order,family`.

The accessor `Noun(false)` currently produces parenthetical output: `caterpillar (insecta, noctuidae)`. This needs to change to natural language like the other accessors.

## Problem 1: Taxonomy Column Inconsistency

The "order" column mixes taxonomic ranks inconsistently:

| What's in "order" | Actual rank | Count | Examples |
|---|---|---|---|
| Classes | Class (rank above order) | ~2,100 | mammalia, reptilia, insecta, aves, amphibia, arachnida, actinopterygii |
| Real orders | Order (correct) | ~500 | carnivora, rodentia, artiodactyla, primates, cetacea, squamata |
| Phyla | Phylum (two ranks above) | ~40 | mollusca, cnidaria, porifera |
| Made-up labels | Not taxonomic | ~100 | "human", "virus", "other fish", "other mammals" |
| Families used as orders | Family (rank below) | ~25 | bovidae, cyprinidae |

The family column is mostly accurate. The fix: use family → correct order mapping to repair the order column. Most mappings are mechanical (canidae → carnivora, sciuridae → rodentia, felidae → carnivora, etc.).

## Problem 2: Massive Distribution Skew

Top orders by count:
- mammalia: 677 (but this is a class, not an order)
- aves: 423
- reptilia: 389
- insecta: 339
- actinopterygii: 298
- carnivora: 236

Top families by count:
- canidae (dogs): 583 — 16.6% of all nouns are dog breeds
- felidae (cats): 95
- bovidae (cattle): 86

583 dog breeds including designer mixes (puggle, pomsky, schneagle) massively skew the distribution. A cap of ~60 per family was discussed.

## Problem 3: Accessor Format

Current: `caterpillar (insecta, noctuidae)`
Target: natural language without parens, consistent with other accessors we've already fixed.

## Proposed Fix Order

1. Fix the "order" column: build a family-to-order lookup, correct all rows
2. Handle non-animal nouns (human, virus, bacteria) with honest non-taxonomic labels
3. Cap bloated families at ~60, keeping visually distinctive/interesting members, cutting generic breeds
4. Fix the Noun accessor to use natural language instead of parens
5. Rebuild archive

## What's Already Done

- Deduped (5 removed)
- Sorted alphabetically by commonName
- 3,514 rows remain

## What's NOT Done

- Order column correction
- Distribution rebalancing (culling)
- Accessor method change
- Archive rebuild for nouns changes
