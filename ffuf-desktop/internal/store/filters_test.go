package store

import (
	"fmt"
	"math/rand"
	"os"
	"reflect"
	"sort"
	"testing"

	"ffuf-desktop/internal/output"
)

func TestFilterPresetsCRUD(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "ffuf-store-presets-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	st, err := NewStore(tempDir)
	if err != nil {
		t.Fatalf("failed to initialize store: %v", err)
	}
	defer st.Close()

	// 1. Save preset
	presetJSON := `{"id":"p1","conjunction":"AND","rules":[{"field":"status","operator":"!=","value":"404","enabled":true}]}`
	if err := st.SaveFilterPreset("Hide 404s", presetJSON); err != nil {
		t.Fatalf("SaveFilterPreset failed: %v", err)
	}

	// 2. Read preset
	presets, err := st.GetFilterPresets()
	if err != nil {
		t.Fatalf("GetFilterPresets failed: %v", err)
	}
	if len(presets) != 1 || presets[0].Name != "Hide 404s" || presets[0].FilterJSON != presetJSON {
		t.Fatalf("unexpected presets list: %+v", presets)
	}

	// 3. Update preset (upsert on name)
	updatedJSON := `{"id":"p1","conjunction":"AND","rules":[{"field":"status","operator":"!=","value":"500","enabled":true}]}`
	if err := st.SaveFilterPreset("Hide 404s", updatedJSON); err != nil {
		t.Fatalf("SaveFilterPreset upsert failed: %v", err)
	}
	presets, _ = st.GetFilterPresets()
	if len(presets) != 1 || presets[0].FilterJSON != updatedJSON {
		t.Fatalf("preset update failed: %+v", presets)
	}

	// 4. Delete preset
	if err := st.DeleteFilterPreset("Hide 404s"); err != nil {
		t.Fatalf("DeleteFilterPreset failed: %v", err)
	}
	presets, _ = st.GetFilterPresets()
	if len(presets) != 0 {
		t.Fatalf("expected 0 presets after delete, got %d", len(presets))
	}
}

func TestDifferentialFilterEvaluation(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "ffuf-store-diff-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	st, err := NewStore(tempDir)
	if err != nil {
		t.Fatalf("failed to initialize store: %v", err)
	}
	defer st.Close()

	sessID, err := st.CreateSession("{}", "{}")
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	// Seed 200 synthetic items with known distributions
	rng := rand.New(rand.NewSource(42))
	statuses := []int64{200, 204, 301, 302, 400, 403, 404, 500, 502}
	var dataset []output.ResultDTO

	for i := int64(1); i <= 200; i++ {
		stCode := statuses[rng.Intn(len(statuses))]
		lenVal := int64(100 + rng.Intn(9900))
		wordsVal := int64(10 + rng.Intn(500))
		linesVal := int64(1 + rng.Intn(100))
		durVal := int64(10 + rng.Intn(990))
		pathWord := fmt.Sprintf("word_%d", rng.Intn(20))
		if i%5 == 0 {
			pathWord = fmt.Sprintf("admin_%d", i)
		} else if i%7 == 0 {
			pathWord = fmt.Sprintf("api/v%d/user", rng.Intn(3)+1)
		}
		item := output.ResultDTO{
			ID:         i,
			Position:   int(i),
			Status:     stCode,
			Length:     lenVal,
			Words:      wordsVal,
			Lines:      linesVal,
			DurationMs: durVal,
			URL:        fmt.Sprintf("http://target.local/%s", pathWord),
			Host:       "target.local",
			Inputs:     map[string]string{"FUZZ": pathWord},
		}
		dataset = append(dataset, item)
	}

	if err := st.InsertResults(sessID, dataset); err != nil {
		t.Fatalf("InsertResults failed: %v", err)
	}

	// Define various test filter groups
	testCases := []struct {
		name  string
		group FilterGroup
	}{
		{
			name: "Status Equal 200",
			group: FilterGroup{
				ID:          "g1",
				Conjunction: "AND",
				Enabled:     true,
				Rules: []FilterRule{
					{Field: "status", Operator: "=", Value: "200", Enabled: true},
				},
			},
		},
		{
			name: "Status Not Equal 404",
			group: FilterGroup{
				ID:          "g2",
				Conjunction: "AND",
				Enabled:     true,
				Rules: []FilterRule{
					{Field: "status", Operator: "!=", Value: "404", Enabled: true},
				},
			},
		},
		{
			name: "Status IN (200, 301, 302)",
			group: FilterGroup{
				ID:          "g3",
				Conjunction: "AND",
				Enabled:     true,
				Rules: []FilterRule{
					{Field: "status", Operator: "in", Value: "200, 301, 302", Enabled: true},
				},
			},
		},
		{
			name: "Length Between 1000 and 5000",
			group: FilterGroup{
				ID:          "g4",
				Conjunction: "AND",
				Enabled:     true,
				Rules: []FilterRule{
					{Field: "length", Operator: "between", Value: "1000", Value2: "5000", Enabled: true},
				},
			},
		},
		{
			name: "Words > 100 AND Lines <= 50",
			group: FilterGroup{
				ID:          "g5",
				Conjunction: "AND",
				Enabled:     true,
				Rules: []FilterRule{
					{Field: "words", Operator: ">", Value: "100", Enabled: true},
					{Field: "lines", Operator: "<=", Value: "50", Enabled: true},
				},
			},
		},
		{
			name: "URL Contains 'admin'",
			group: FilterGroup{
				ID:          "g6",
				Conjunction: "AND",
				Enabled:     true,
				Rules: []FilterRule{
					{Field: "url", Operator: "contains", Value: "admin", Enabled: true},
				},
			},
		},
		{
			name: "URL Regex api/v[1-3]",
			group: FilterGroup{
				ID:          "g7",
				Conjunction: "AND",
				Enabled:     true,
				Rules: []FilterRule{
					{Field: "url", Operator: "regex", Value: "api/v[1-3]", Enabled: true},
				},
			},
		},
		{
			name: "Compound (Status = 200 AND Words > 200) OR (Status = 301)",
			group: FilterGroup{
				ID:          "g8",
				Conjunction: "OR",
				Enabled:     true,
				Groups: []FilterGroup{
					{
						ID:          "sub1",
						Conjunction: "AND",
						Enabled:     true,
						Rules: []FilterRule{
							{Field: "status", Operator: "=", Value: "200", Enabled: true},
							{Field: "words", Operator: ">", Value: "200", Enabled: true},
						},
					},
					{
						ID:          "sub2",
						Conjunction: "AND",
						Enabled:     true,
						Rules: []FilterRule{
							{Field: "status", Operator: "=", Value: "301", Enabled: true},
						},
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// 1. In-memory evaluation
			var expectedIDs []int64
			for _, item := range dataset {
				if EvaluateFilterGroup(tc.group, &item) {
					expectedIDs = append(expectedIDs, item.ID)
				}
			}
			sort.Slice(expectedIDs, func(i, j int) bool { return expectedIDs[i] < expectedIDs[j] })

			// 2. SQLite query evaluation
			results, totalCount, err := st.GetResultsFiltered(sessID, tc.group, 500, 0)
			if err != nil {
				t.Fatalf("GetResultsFiltered failed: %v", err)
			}

			if int64(len(expectedIDs)) != totalCount {
				t.Fatalf("total count mismatch: in-memory=%d, sqlite=%d", len(expectedIDs), totalCount)
			}

			var actualIDs []int64
			for _, r := range results {
				actualIDs = append(actualIDs, r.ID)
			}
			sort.Slice(actualIDs, func(i, j int) bool { return actualIDs[i] < actualIDs[j] })

			if !reflect.DeepEqual(expectedIDs, actualIDs) {
				t.Fatalf("differential mismatch for %q:\nexpected: %v\ngot:      %v", tc.name, expectedIDs, actualIDs)
			}
		})
	}
}

func TestFilterReDoSSafety(t *testing.T) {
	// 1. Regex exceeding 256 characters should be rejected
	longPattern := ""
	for i := 0; i < 300; i++ {
		longPattern += "a"
	}

	group := FilterGroup{
		ID:          "redos",
		Conjunction: "AND",
		Enabled:     true,
		Rules: []FilterRule{
			{Field: "url", Operator: "regex", Value: longPattern, Enabled: true},
		},
	}

	_, _, err := BuildFilterSQL(group)
	if err == nil {
		t.Fatalf("expected error for regex > 256 chars, got nil")
	}

	// 2. In-memory evaluation also returns false on overly long regex
	item := output.ResultDTO{URL: "http://example.com/aaa"}
	if EvaluateFilterGroup(group, &item) {
		t.Fatalf("expected overly long regex to fail in memory evaluation")
	}

	// 3. Invalid regex syntax returns error
	invalidGroup := FilterGroup{
		ID:          "invalid",
		Conjunction: "AND",
		Enabled:     true,
		Rules: []FilterRule{
			{Field: "url", Operator: "regex", Value: "[a-z(", Enabled: true},
		},
	}
	_, _, err = BuildFilterSQL(invalidGroup)
	if err == nil {
		t.Fatalf("expected error for invalid regex syntax, got nil")
	}
}
