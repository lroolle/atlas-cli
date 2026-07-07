package cmd

import (
	"reflect"
	"strings"
	"testing"

	"github.com/lroolle/atlas-cli/api"
)

func sampleTransitions() []api.Transition {
	return []api.Transition{
		{ID: "4", Name: "Start Progress", To: api.Status{Name: "In Development"}},
		{ID: "711", Name: "Start Review", To: api.Status{Name: "In Review"}},
		{
			ID: "5", Name: "Resolve Issue", To: api.Status{Name: "Resolved"},
			Fields: map[string]api.TransitionField{
				"resolution": {
					Required: true, Name: "Resolution",
					Schema:        api.TransitionSchema{Type: "resolution"},
					AllowedValues: []api.AllowedValue{{Name: "Done"}, {Name: "Fixed"}},
				},
				"customfield_12600": {
					Name:          "Root Cause",
					Schema:        api.TransitionSchema{Type: "option"},
					AllowedValues: []api.AllowedValue{{Value: "功能缺陷"}, {Value: "性能缺陷"}},
				},
				"customfield_12301": {
					Name:          "Test Level",
					Schema:        api.TransitionSchema{Type: "array", Items: "option"},
					AllowedValues: []api.AllowedValue{{Value: "Manual Test"}, {Value: "IT Auto"}},
				},
				"customfield_11202": {
					Name:   "Analysis Notes",
					Schema: api.TransitionSchema{Type: "array", Items: "string"},
				},
				"customfield_10103": {
					Name:   "Story Points",
					Schema: api.TransitionSchema{Type: "number"},
				},
			},
		},
		{ID: "2", Name: "Close Issue", To: api.Status{Name: "Closed"}},
	}
}

func TestMatchTransition(t *testing.T) {
	transitions := sampleTransitions()

	cases := []struct {
		arg  string
		want string // expected transition ID
	}{
		{"Start Progress", "4"},
		{"start progress", "4"},
		{"In Development", "4"}, // target status name
		{"711", "711"},          // transition ID
		{"resolve", "5"},        // unique substring
		{"closed", "2"},         // substring of target status
	}
	for _, tc := range cases {
		t.Run(tc.arg, func(t *testing.T) {
			got, err := matchTransition(transitions, tc.arg)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.ID != tc.want {
				t.Errorf("matched ID = %s, want %s", got.ID, tc.want)
			}
		})
	}
}

func TestMatchTransitionErrors(t *testing.T) {
	transitions := sampleTransitions()

	if _, err := matchTransition(transitions, "nonexistent"); err == nil {
		t.Error("expected not-found error, got nil")
	}
	// "start" matches both "Start Progress" and "Start Review"
	_, err := matchTransition(transitions, "start")
	if err == nil || !strings.Contains(err.Error(), "ambiguous") {
		t.Errorf("expected ambiguous error, got %v", err)
	}
}

func TestBuildTransitionFieldsFlags(t *testing.T) {
	resolve := &sampleTransitions()[2]
	fields, err := buildTransitionFields(transitionOptions{
		Resolution:  "Done",
		FixVersions: []string{"6.1.0"},
	}, resolve)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := map[string]interface{}{
		"resolution":  map[string]string{"name": "Done"},
		"fixVersions": []map[string]string{{"name": "6.1.0"}},
	}
	if !reflect.DeepEqual(fields, want) {
		t.Errorf("fields = %#v, want %#v", fields, want)
	}
}

func TestBuildTransitionFieldsByName(t *testing.T) {
	resolve := &sampleTransitions()[2]
	fields, err := buildTransitionFields(transitionOptions{
		RawFields: []string{
			"Root Cause=功能缺陷",
			"test level=Manual Test,IT Auto",
			"Analysis Notes=race in sink, fixed by lock",
			"Story Points=3",
		},
	}, resolve)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := map[string]interface{}{
		"customfield_12600": map[string]string{"value": "功能缺陷"},
		"customfield_12301": []interface{}{
			map[string]string{"value": "Manual Test"},
			map[string]string{"value": "IT Auto"},
		},
		// free-text array: comma NOT split
		"customfield_11202": []interface{}{"race in sink, fixed by lock"},
		"customfield_10103": float64(3),
	}
	if !reflect.DeepEqual(fields, want) {
		t.Errorf("fields = %#v, want %#v", fields, want)
	}
}

func TestBuildTransitionFieldsRawJSONAndID(t *testing.T) {
	resolve := &sampleTransitions()[2]
	fields, err := buildTransitionFields(transitionOptions{
		RawFields: []string{
			`customfield_12600={"value":"性能缺陷"}`,
			"customfield_99999=freeform", // not on screen: passes through as string
		},
	}, resolve)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := map[string]interface{}{
		"customfield_12600": map[string]interface{}{"value": "性能缺陷"},
		"customfield_99999": "freeform",
	}
	if !reflect.DeepEqual(fields, want) {
		t.Errorf("fields = %#v, want %#v", fields, want)
	}
}

func TestBuildTransitionFieldsRepeatedArrayFlag(t *testing.T) {
	resolve := &sampleTransitions()[2]
	fields, err := buildTransitionFields(transitionOptions{
		RawFields: []string{
			"Analysis Notes=first finding",
			"Analysis Notes=second finding",
		},
	}, resolve)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := map[string]interface{}{
		"customfield_11202": []interface{}{"first finding", "second finding"},
	}
	if !reflect.DeepEqual(fields, want) {
		t.Errorf("fields = %#v, want %#v", fields, want)
	}
}

func TestBuildTransitionFieldsErrors(t *testing.T) {
	resolve := &sampleTransitions()[2]

	if _, err := buildTransitionFields(transitionOptions{RawFields: []string{"noequals"}}, resolve); err == nil {
		t.Error("expected error for missing '=', got nil")
	}
	// Display-name-looking key not on the transition screen
	if _, err := buildTransitionFields(transitionOptions{RawFields: []string{"No Such Field=x"}}, resolve); err == nil {
		t.Error("expected error for unknown display name, got nil")
	}
}

func startProgressTransition() *api.Transition {
	return &api.Transition{
		ID: "4", Name: "Start Progress", To: api.Status{Name: "In Development"},
		Fields: map[string]api.TransitionField{
			"customfield_12203": {Name: "Team Zone", Schema: api.TransitionSchema{Type: "option-with-child"}},
			"customfield_12101": {Name: "Discipline", Schema: api.TransitionSchema{Type: "option"}},
			"customfield_11201": {Name: "Squad", Schema: api.TransitionSchema{Type: "option"}},
			"customfield_10103": {Name: "Story Points", Schema: api.TransitionSchema{Type: "number"}},
			"versions":          {Name: "Affects Version/s", Schema: api.TransitionSchema{Type: "array", Items: "version"}},
			"timetracking":      {Name: "Time Tracking", Schema: api.TransitionSchema{Type: "timetracking"}},
		},
	}
}

func TestBuildTransitionFieldsDefaults(t *testing.T) {
	fields, err := buildTransitionFields(transitionOptions{
		Defaults: map[string]string{
			"team zone":         "Backend / Platform",
			"discipline":        "Services",
			"squad":             "Alpha",
			"story points":      "0",
			"affects version/s": "1.2.0",
			"timetracking":      "4h",
			"fix version/s":     "1.2.0", // not on this screen: skipped
		},
	}, startProgressTransition())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := map[string]interface{}{
		"customfield_12203": map[string]interface{}{
			"value": "Backend",
			"child": map[string]string{"value": "Platform"},
		},
		"customfield_12101": map[string]string{"value": "Services"},
		"customfield_11201": map[string]string{"value": "Alpha"},
		"customfield_10103": float64(0),
		"versions":          []interface{}{map[string]string{"name": "1.2.0"}},
		"timetracking":      map[string]string{"originalEstimate": "4h"},
	}
	if !reflect.DeepEqual(fields, want) {
		t.Errorf("fields = %#v, want %#v", fields, want)
	}
}

func TestBuildTransitionFieldsFlagOverridesDefault(t *testing.T) {
	resolve := &sampleTransitions()[2]
	fields, err := buildTransitionFields(transitionOptions{
		Resolution: "Fixed",
		RawFields:  []string{"Test Level=IT Auto"},
		Defaults: map[string]string{
			"resolution": "Done",
			"test level": "Manual Test",
			"root cause": "功能缺陷",
		},
	}, resolve)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := map[string]interface{}{
		"resolution":        map[string]string{"name": "Fixed"},
		"customfield_12301": []interface{}{map[string]string{"value": "IT Auto"}}, // replaced, not appended
		"customfield_12600": map[string]string{"value": "功能缺陷"},
	}
	if !reflect.DeepEqual(fields, want) {
		t.Errorf("fields = %#v, want %#v", fields, want)
	}
}

func TestPopDefault(t *testing.T) {
	defaults := map[string]string{"time spent": "2h", "team zone": "Backend"}
	v, ok := popDefault(defaults, "Time Spent")
	if !ok || v != "2h" {
		t.Errorf("popDefault = %q, %v", v, ok)
	}
	if _, exists := defaults["time spent"]; exists {
		t.Error("popDefault did not remove the key")
	}
	if _, ok := popDefault(nil, "Time Spent"); ok {
		t.Error("popDefault on nil map should return false")
	}
}

func TestCoerceTransitionValueScalar(t *testing.T) {
	cases := []struct {
		name string
		meta api.TransitionField
		raw  string
		want interface{}
	}{
		{"option", api.TransitionField{Schema: api.TransitionSchema{Type: "option"}}, "建议", map[string]string{"value": "建议"}},
		{"resolution", api.TransitionField{Schema: api.TransitionSchema{Type: "resolution"}}, "Done", map[string]string{"name": "Done"}},
		{"user", api.TransitionField{Schema: api.TransitionSchema{Type: "user"}}, "eric.wang", map[string]string{"name": "eric.wang"}},
		{"number", api.TransitionField{Schema: api.TransitionSchema{Type: "number"}}, "2.5", 2.5},
		{"bad number stays string", api.TransitionField{Schema: api.TransitionSchema{Type: "number"}}, "abc", "abc"},
		{"string", api.TransitionField{Schema: api.TransitionSchema{Type: "string"}}, "plain", "plain"},
		{"cascading no child", api.TransitionField{Schema: api.TransitionSchema{Type: "option-with-child"}}, "Backend",
			map[string]interface{}{"value": "Backend"}},
		{"timetracking", api.TransitionField{Schema: api.TransitionSchema{Type: "timetracking"}}, "4h",
			map[string]string{"originalEstimate": "4h"}},
		{"version array", api.TransitionField{Schema: api.TransitionSchema{Type: "array", Items: "version"}}, "6.1.0, 6.2.0",
			[]interface{}{map[string]string{"name": "6.1.0"}, map[string]string{"name": "6.2.0"}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := coerceTransitionValue(tc.meta, tc.raw)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("coerce(%q) = %#v, want %#v", tc.raw, got, tc.want)
			}
		})
	}
}

func TestFormatAllowedValues(t *testing.T) {
	vals := []api.AllowedValue{{Name: "Done"}, {Value: "功能缺陷"}}
	if got := formatAllowedValues(vals); got != "Done, 功能缺陷" {
		t.Errorf("got %q", got)
	}

	var many []api.AllowedValue
	for i := 0; i < 20; i++ {
		many = append(many, api.AllowedValue{Name: "v"})
	}
	if got := formatAllowedValues(many); !strings.Contains(got, "+8 more") {
		t.Errorf("expected truncation marker, got %q", got)
	}

	if got := formatAllowedValues(nil); got != "" {
		t.Errorf("expected empty for nil, got %q", got)
	}
}
