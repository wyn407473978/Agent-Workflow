package domain

import "encoding/json"

// SlotValue 区分本轮未提及、显式清空和设置具体值三种情况。
type SlotValue[T any] struct {
	Present bool
	Value   *T
}

func (s SlotValue[T]) IsZero() bool { return !s.Present }

func (s SlotValue[T]) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.Value)
}

func (s *SlotValue[T]) UnmarshalJSON(data []byte) error {
	var value *T
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	*s = SlotValue[T]{Present: true, Value: value}
	return nil
}

// SlotPatch 只包含本轮需要修改的字段。JSON 中的 null 表示清空该字段。
type SlotPatch struct {
	ActivityName SlotValue[string]                     `json:"activity_name,omitzero"`
	AudienceRule SlotValue[AudienceRule]               `json:"audience_rule,omitzero"`
	ScheduledAt  SlotValue[string]                     `json:"scheduled_at,omitzero"` // 待结合事件基准时间解析的时间表达式。
	TimeZone     SlotValue[string]                     `json:"timezone,omitzero"`
	TemplateID   SlotValue[string]                     `json:"template_id,omitzero"`
	ExtraSlots   SlotValue[map[string]json.RawMessage] `json:"extra_slots,omitzero"`
}

func (p *SlotPatch) UnmarshalJSON(data []byte) error {
	type patch SlotPatch
	var decoded patch
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	*p = SlotPatch(decoded)
	return nil
}
