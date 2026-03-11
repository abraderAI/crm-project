package scoring

import (
	"testing"
)

func FuzzScoringEvaluate(f *testing.F) {
	// ≥50 seed entries for scoring metadata fuzzing.
	seeds := []string{
		// Valid metadata.
		`{}`,
		`{"stage":"new_lead"}`,
		`{"stage":"contacted"}`,
		`{"stage":"qualified"}`,
		`{"stage":"proposal"}`,
		`{"stage":"negotiation"}`,
		`{"stage":"closed_won"}`,
		`{"stage":"closed_lost"}`,
		`{"stage":"new_lead","priority":"high"}`,
		`{"stage":"qualified","priority":"medium"}`,
		`{"stage":"proposal","priority":"high","company":"Acme"}`,
		`{"stage":"new_lead","contact_email":"test@example.com"}`,
		`{"stage":"qualified","company":"BigCorp","contact_email":"a@b.com","deal_value":50000}`,
		`{"deal_value":100}`,
		`{"deal_value":1000}`,
		`{"deal_value":5000}`,
		`{"deal_value":10000}`,
		`{"deal_value":50000}`,
		`{"deal_value":100000}`,
		`{"deal_value":"5000"}`,
		`{"deal_value":"not_a_number"}`,
		`{"company":"Test Co"}`,
		`{"contact_email":"someone@example.com"}`,
		`{"priority":"low"}`,
		`{"priority":"high"}`,
		`{"priority":"medium"}`,
		// Nested.
		`{"contact":{"email":"test@test.com"}}`,
		`{"contact":{"name":"John","email":"j@test.com"}}`,
		// Edge cases.
		``,
		`null`,
		`"string"`,
		`[]`,
		`true`,
		`false`,
		`0`,
		`not-json-at-all`,
		`{"stage":123}`,
		`{"stage":null}`,
		`{"stage":true}`,
		`{"stage":[]}`,
		`{"stage":{}}`,
		`{"deal_value":null}`,
		`{"deal_value":true}`,
		`{"deal_value":[]}`,
		`{"deal_value":{}}`,
		`{"deal_value":-100}`,
		`{"deal_value":0}`,
		`{"deal_value":9999999999}`,
		// Adversarial.
		`<script>alert(1)</script>`,
		`{"stage":"<script>"}`,
		`{"stage":"' OR 1=1 --"}`,
		`{"\x00":"\x00"}`,
	}

	for _, s := range seeds {
		f.Add(s)
	}

	rules := DefaultRules()

	f.Fuzz(func(t *testing.T, metadata string) {
		// Must not panic.
		result := Evaluate(rules, metadata)
		if result == nil {
			t.Fatal("Evaluate returned nil")
		}
		if result.TotalScore < 0 {
			t.Fatal("TotalScore should not be negative")
		}
	})
}

func FuzzScoringCustomRules(f *testing.F) {
	// ≥50 seed entries for custom rule JSON parsing.
	seeds := []string{
		`{}`,
		`{"scoring_rules":[]}`,
		`{"scoring_rules":[{"name":"test","path":"stage","operator":"eq","value":"hot","points":50}]}`,
		`{"scoring_rules":[{"name":"a","path":"x","operator":"gt","value":"100","points":10}]}`,
		`{"scoring_rules":[{"name":"b","path":"y","operator":"contains","value":"urgent","points":20}]}`,
		`{"scoring_rules":[{"name":"c","path":"z","operator":"exists","value":"","points":5}]}`,
		`{"scoring_rules":[{"name":"d","path":"a","operator":"gte","value":"50","points":15}]}`,
		`{"scoring_rules":[{"name":"e","path":"b","operator":"lt","value":"10","points":3}]}`,
		`{"scoring_rules":[{"name":"f","path":"c","operator":"lte","value":"5","points":2}]}`,
		`{"scoring_rules":"not-an-array"}`,
		`{"scoring_rules":null}`,
		`{"scoring_rules":123}`,
		`{"scoring_rules":true}`,
		`{"scoring_rules":{"not":"array"}}`,
		`{"other":"field"}`,
		``,
		`not-json`,
		`null`,
		`[]`,
		`"string"`,
		`{"scoring_rules":[{"invalid":"fields"}]}`,
		`{"scoring_rules":[{"name":"","path":"","operator":"","value":"","points":0}]}`,
		`{"scoring_rules":[{"name":"a","path":"x.y.z","operator":"eq","value":"deep","points":100}]}`,
		// Multiple rules.
		`{"scoring_rules":[{"name":"r1","path":"stage","operator":"eq","value":"hot","points":50},{"name":"r2","path":"priority","operator":"eq","value":"high","points":20}]}`,
		// Adversarial.
		`<script>alert(1)</script>`,
		`{"scoring_rules":"<script>"}`,
	}

	for _, s := range seeds {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, metadataJSON string) {
		// Must not panic.
		_ = ParseRulesFromMetadata(metadataJSON)
	})
}
