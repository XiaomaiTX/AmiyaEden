package service

import "testing"

func TestCharacterTierMultiplier(t *testing.T) {
	tests := []struct {
		name           string
		position       int
		fullCount      int
		reducedCount   int
		reducedPercent int
		want           float64
	}{
		{name: "position 1 full reward", position: 1, fullCount: 3, reducedCount: 3, reducedPercent: 50, want: 1.0},
		{name: "position 3 full reward boundary", position: 3, fullCount: 3, reducedCount: 3, reducedPercent: 50, want: 1.0},
		{name: "position 4 reduced reward", position: 4, fullCount: 3, reducedCount: 3, reducedPercent: 50, want: 0.5},
		{name: "position 6 reduced reward boundary", position: 6, fullCount: 3, reducedCount: 3, reducedPercent: 50, want: 0.5},
		{name: "position 7 zero reward", position: 7, fullCount: 3, reducedCount: 3, reducedPercent: 50, want: 0.0},
		{name: "position 100 zero reward", position: 100, fullCount: 3, reducedCount: 3, reducedPercent: 50, want: 0.0},
		{name: "fullCount 0 position 1 is reduced", position: 1, fullCount: 0, reducedCount: 3, reducedPercent: 50, want: 0.5},
		{name: "reducedCount 0 position 4 is zero", position: 4, fullCount: 3, reducedCount: 0, reducedPercent: 50, want: 0.0},
		{name: "reducedPercent 0", position: 4, fullCount: 3, reducedCount: 3, reducedPercent: 0, want: 0.0},
		{name: "reducedPercent 100", position: 4, fullCount: 3, reducedCount: 3, reducedPercent: 100, want: 1.0},
		{name: "position 0 zero reward", position: 0, fullCount: 3, reducedCount: 3, reducedPercent: 50, want: 0.0},
		{name: "position negative zero reward", position: -1, fullCount: 3, reducedCount: 3, reducedPercent: 50, want: 0.0},
		{name: "large fullCount no restriction", position: 50, fullCount: 999, reducedCount: 0, reducedPercent: 50, want: 1.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CharacterTierMultiplier(tt.position, tt.fullCount, tt.reducedCount, tt.reducedPercent)
			if got != tt.want {
				t.Fatalf("CharacterTierMultiplier(%d, %d, %d, %d) = %v, want %v",
					tt.position, tt.fullCount, tt.reducedCount, tt.reducedPercent, got, tt.want)
			}
		})
	}
}
