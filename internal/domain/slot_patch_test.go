package domain

import (
	"encoding/json"
	"testing"
)

func TestSlotPatchPreservesUnspecifiedAndExplicitValues(t *testing.T) {
	var patch SlotPatch
	input := `{"activity_name":"","scheduled_at":"明天上午十点","audience_rule":{"min_spend_yuan":500.10},"timezone":null}`
	if err := json.Unmarshal([]byte(input), &patch); err != nil {
		t.Fatal(err)
	}
	if !patch.ActivityName.Present || patch.ActivityName.Value == nil || *patch.ActivityName.Value != "" {
		t.Fatal("explicit empty name must be preserved")
	}
	if !patch.TimeZone.Present || patch.TimeZone.Value != nil || patch.TemplateID.Present {
		t.Fatal("explicit clearing and omission must be distinct")
	}
	if patch.ScheduledAt.Value == nil || *patch.ScheduledAt.Value != "明天上午十点" {
		t.Fatal("relative schedule must remain available for normalization")
	}
	if rule := patch.AudienceRule.Value; rule == nil || rule.MinSpendYuan == nil || rule.MinSpendYuan.String() != "500.10" {
		t.Fatal("audience spend threshold must retain its precision")
	}
	encoded, err := json.Marshal(patch)
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &fields); err != nil {
		t.Fatal(err)
	}
	if _, exists := fields["template_id"]; exists || string(fields["timezone"]) != "null" || string(fields["activity_name"]) != `""` {
		t.Fatalf("patch lost omission, null, or explicit value: %s", encoded)
	}
}

func TestSlotPatchDecodeReplacesPreviousPatch(t *testing.T) {
	patch := SlotPatch{ActivityName: SlotValue[string]{Present: true, Value: new("old")}}
	if err := json.Unmarshal([]byte(`{"template_id":"tpl-002"}`), &patch); err != nil {
		t.Fatal(err)
	}
	if patch.ActivityName.Present || patch.TemplateID.Value == nil || *patch.TemplateID.Value != "tpl-002" {
		t.Fatalf("new patch retained fields from the previous round: %+v", patch)
	}
}
