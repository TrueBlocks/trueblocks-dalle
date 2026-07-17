package dalle

import "testing"

// The seed must not depend on the series. Two series over one input have to trace
// the same trajectory through every database so they differ only on the axes they
// actually filter (see NormalizeSeed).
func TestNormalizeSeedIgnoresSeries(t *testing.T) {
	const input = "Jesus loves us. This we know."

	first, err := NormalizeSeed(input, "", "five-tone-postal-protozoa")
	if err != nil {
		t.Fatalf("NormalizeSeed: %v", err)
	}
	second, err := NormalizeSeed(input, "", "warm-tone-tender-symbionts")
	if err != nil {
		t.Fatalf("NormalizeSeed: %v", err)
	}
	empty, err := NormalizeSeed(input, "", "")
	if err != nil {
		t.Fatalf("NormalizeSeed: %v", err)
	}

	if first != second {
		t.Errorf("seed changed with series:\n  %s\n  %s", first, second)
	}
	if first != empty {
		t.Errorf("seed changed when series omitted:\n  %s\n  %s", first, empty)
	}
	if first != stableSeed(input) {
		t.Errorf("seed = %s, want plain digest %s", first, stableSeed(input))
	}
}

func TestNormalizeSeedKeepsExplicitSeed(t *testing.T) {
	const explicit = "19b6cb98910d5b58749b196661a9c532ffe6f483dc678b09e863ca00a99b9f62"

	got, err := NormalizeSeed("some input", explicit, "five-tone-postal-protozoa")
	if err != nil {
		t.Fatalf("NormalizeSeed: %v", err)
	}
	if got != explicit {
		t.Errorf("explicit seed was re-hashed: got %s, want %s", got, explicit)
	}
}

func TestNormalizeSeedRequiresInput(t *testing.T) {
	if _, err := NormalizeSeed("   ", "", "empty"); err == nil {
		t.Error("expected an error for blank input, got nil")
	}
}

// A stable seed is only safe because per-series identity is keyed elsewhere.
// If ComputeImageID ever stops folding in the series, seeds would collide.
func TestImageIDDistinguishesSeriesAtSameSeed(t *testing.T) {
	const seed = "19b6cb98910d5b58749b196661a9c532ffe6f483dc678b09e863ca00a99b9f62"

	first := NewImageMetadata("Jesus loves us. This we know.", seed, "five-tone-postal-protozoa")
	second := NewImageMetadata("Jesus loves us. This we know.", seed, "warm-tone-tender-symbionts")

	if ComputeImageID(first) == ComputeImageID(second) {
		t.Error("image IDs collide across series at the same seed")
	}
}
