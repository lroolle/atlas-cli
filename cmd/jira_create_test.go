package cmd

import (
	"reflect"
	"testing"

	"github.com/lroolle/atlas-cli/api"
)

func TestBuildIssueFieldsMinimal(t *testing.T) {
	fields, err := buildIssueFields(issueCreateOptions{
		Project: "MYPROJ",
		Type:    "Story",
		Summary: "test story",
	}, agileFieldIDs{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := map[string]interface{}{
		"project":   map[string]string{"key": "MYPROJ"},
		"issuetype": map[string]string{"name": "Story"},
		"summary":   "test story",
	}
	if !reflect.DeepEqual(fields, want) {
		t.Errorf("fields = %#v, want %#v", fields, want)
	}
}

func TestBuildIssueFieldsValidation(t *testing.T) {
	cases := []struct {
		name string
		opts issueCreateOptions
	}{
		{"missing type", issueCreateOptions{Project: "MYPROJ", Summary: "s"}},
		{"missing summary", issueCreateOptions{Project: "MYPROJ", Type: "Story"}},
		{"epic without resolved field", issueCreateOptions{Project: "MYPROJ", Type: "Story", Summary: "s", Epic: "MYPROJ-1"}},
		{"sprint without resolved field", issueCreateOptions{Project: "MYPROJ", Type: "Story", Summary: "s", Sprint: 7}},
		{"bad raw field", issueCreateOptions{Project: "MYPROJ", Type: "Story", Summary: "s", RawFields: []string{"noequals"}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := buildIssueFields(tc.opts, agileFieldIDs{}); err == nil {
				t.Error("expected error, got nil")
			}
		})
	}
}

func TestBuildIssueFieldsSubtask(t *testing.T) {
	fields, err := buildIssueFields(issueCreateOptions{
		Project:  "MYPROJ",
		Type:     "Sub-task",
		Summary:  "dev subtask work",
		Parent:   "24318",
		Priority: "Major",
	}, agileFieldIDs{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	parent, ok := fields["parent"].(map[string]string)
	if !ok || parent["key"] != "MYPROJ-24318" {
		t.Errorf("parent = %#v, want key MYPROJ-24318 (auto-prefixed)", fields["parent"])
	}
	priority, ok := fields["priority"].(map[string]string)
	if !ok || priority["name"] != "Major" {
		t.Errorf("priority = %#v, want name Major", fields["priority"])
	}
}

func TestBuildIssueFieldsAgile(t *testing.T) {
	agile := agileFieldIDs{
		EpicLink:    "customfield_10501",
		Sprint:      "customfield_10401",
		StoryPoints: "customfield_10103",
	}
	fields, err := buildIssueFields(issueCreateOptions{
		Project:     "MYPROJ",
		Type:        "Story",
		Summary:     "agile story",
		Epic:        "23743",
		Sprint:      1946,
		StoryPoints: 3,
	}, agile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if fields["customfield_10501"] != "MYPROJ-23743" {
		t.Errorf("epic = %v, want MYPROJ-23743", fields["customfield_10501"])
	}
	if fields["customfield_10401"] != 1946 {
		t.Errorf("sprint = %v, want 1946", fields["customfield_10401"])
	}
	if fields["customfield_10103"] != 3.0 {
		t.Errorf("story points = %v, want 3", fields["customfield_10103"])
	}
}

func TestBuildIssueFieldsRawFieldJSON(t *testing.T) {
	fields, err := buildIssueFields(issueCreateOptions{
		Project: "MYPROJ",
		Type:    "Story",
		Summary: "raw fields",
		RawFields: []string{
			`customfield_12113={"value":"商业论证"}`,
			"customfield_10700=plain",
		},
	}, agileFieldIDs{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	obj, ok := fields["customfield_12113"].(map[string]interface{})
	if !ok || obj["value"] != "商业论证" {
		t.Errorf("json raw field = %#v, want parsed object", fields["customfield_12113"])
	}
	if fields["customfield_10700"] != "plain" {
		t.Errorf("plain raw field = %v, want plain", fields["customfield_10700"])
	}
}

func TestFindAgileFields(t *testing.T) {
	jiraFields := []api.JiraField{
		{ID: "summary", Name: "Summary"},
		{ID: "customfield_10501", Name: "Epic Link", Custom: true},
		{ID: "customfield_10401", Name: "Sprint", Custom: true},
		{ID: "customfield_10103", Name: "Story Points", Custom: true},
	}
	jiraFields[1].Schema.Custom = "com.pyxis.greenhopper.jira:gh-epic-link"
	jiraFields[2].Schema.Custom = "com.pyxis.greenhopper.jira:gh-sprint"
	jiraFields[3].Schema.Custom = "com.atlassian.jira.plugin.system.customfieldtypes:float"

	agile := findAgileFields(jiraFields)

	want := agileFieldIDs{
		EpicLink:    "customfield_10501",
		Sprint:      "customfield_10401",
		StoryPoints: "customfield_10103",
	}
	if agile != want {
		t.Errorf("agile = %+v, want %+v", agile, want)
	}
}
