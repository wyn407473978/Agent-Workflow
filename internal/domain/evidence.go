package domain

import "time"

// Money 使用货币最小单位表示金额，例如 CNY 的分。
type Money struct {
	AmountMinor int64  `json:"amount_minor"`
	Currency    string `json:"currency"`
}

// EvidenceWindow 保存查询时点和可复用该结果的截止时间。
type EvidenceWindow struct {
	CheckedAt time.Time `json:"checked_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

type AudienceSnapshot struct {
	SnapshotID string `json:"audience_snapshot_id"`
	Count      int64  `json:"audience_count"`
	EvidenceWindow
}

type TemplateStatus string

const TemplateApproved TemplateStatus = "APPROVED"

type TemplateSnapshot struct {
	TemplateID string         `json:"template_id"`
	Name       string         `json:"name,omitempty"`
	Status     TemplateStatus `json:"template_status"`
	EvidenceWindow
}

type QuotaSnapshot struct {
	Balance        Money  `json:"balance"`
	AvailableQuota Money  `json:"available_quota"`
	EstimatedCost  *Money `json:"estimated_cost,omitempty"` // 尚无法估算时为空。
	EvidenceWindow
}

type PolicyPassage struct {
	Text      string `json:"text"`
	SourceRef string `json:"source_ref"`
}

type PolicySnapshot struct {
	Passages []PolicyPassage `json:"passages"`
	EvidenceWindow
}

// DraftEvidence 汇总独立查询取得的业务证据。
type DraftEvidence struct {
	Audience *AudienceSnapshot `json:"audience,omitempty"`
	Template *TemplateSnapshot `json:"template,omitempty"`
	Quota    *QuotaSnapshot    `json:"quota,omitempty"`
	Policy   *PolicySnapshot   `json:"policy,omitempty"`
}
