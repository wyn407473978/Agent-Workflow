BEGIN;

CREATE SCHEMA agent_workflow;

CREATE TABLE agent_workflow.workflow_session (
    session_id uuid,
    user_id text,
    account_id text,
    active_workflow_id uuid,
    created_at timestamptz,
    updated_at timestamptz
);

CREATE TABLE agent_workflow.workflow_instance (
    workflow_id uuid,
    session_id uuid,
    status text,
    current_node text,
    intent_name text,
    workflow_version text,
    state_revision bigint,
    draft_version bigint,
    approval_id uuid,
    activity_id text,
    last_error_code text,
    last_error_detail jsonb,
    created_at timestamptz,
    updated_at timestamptz
);

CREATE TABLE agent_workflow.activity_draft (
    workflow_id uuid,
    draft_version bigint,
    activity_name text,
    audience_rule jsonb,
    scheduled_at timestamptz,
    timezone text,
    template_id text,
    extra_slots jsonb,
    missing_slots jsonb,
    audience_snapshot_id text,
    audience_count bigint,
    template_status text,
    estimated_cost_minor bigint,
    currency text,
    snapshot_json jsonb,
    rule_version text,
    validation_json jsonb,
    preview_id uuid,
    previewed_at timestamptz,
    preview_expires_at timestamptz,
    confirmation_version bigint,
    confirmation_event_id uuid,
    confirmed_by text,
    confirmed_at timestamptz,
    confirmation_expires_at timestamptz,
    created_at timestamptz,
    updated_at timestamptz
);

CREATE TABLE agent_workflow.workflow_event (
    event_id uuid,
    session_id uuid,
    workflow_id uuid,
    source text,
    source_event_id text,
    event_type text,
    actor_type text,
    actor_id text,
    request_hash text,
    payload jsonb,
    result_json jsonb,
    status text,
    attempt_count integer,
    next_attempt_at timestamptz,
    lease_token uuid,
    lease_expires_at timestamptz,
    last_error_code text,
    reference_at timestamptz,
    created_at timestamptz,
    updated_at timestamptz,
    processed_at timestamptz
);

CREATE TABLE agent_workflow.workflow_transition (
    workflow_id uuid,
    state_revision bigint,
    event_id uuid,
    draft_version bigint,
    from_status text,
    to_status text,
    reason_code text,
    reason_detail jsonb,
    actor_type text,
    actor_id text,
    created_at timestamptz
);

CREATE TABLE agent_workflow.approval_request (
    approval_id uuid,
    workflow_id uuid,
    draft_version bigint,
    preview_id uuid,
    confirmation_event_id uuid,
    status text,
    request_event_id uuid,
    request_json jsonb,
    decision_event_id uuid,
    decided_by text,
    decided_at timestamptz,
    reason text,
    invalidated_at timestamptz,
    created_at timestamptz,
    updated_at timestamptz
);

CREATE TABLE agent_workflow.activity_create_operation (
    idempotency_key text,
    workflow_id uuid,
    draft_version bigint,
    confirmation_event_id uuid,
    authorization_event_id uuid,
    request_hash text,
    request_json jsonb,
    status text,
    activity_id text,
    attempt_count integer,
    reconcile_count integer,
    next_attempt_at timestamptz,
    lease_token uuid,
    lease_expires_at timestamptz,
    last_error_code text,
    last_error_detail jsonb,
    created_at timestamptz,
    updated_at timestamptz,
    finished_at timestamptz
);

CREATE TABLE agent_workflow.marketing_activity (
    activity_id text,
    idempotency_key text,
    request_hash text,
    workflow_id uuid,
    draft_version bigint,
    account_id text,
    created_by text,
    activity_name text,
    audience_rule jsonb,
    audience_snapshot_id text,
    audience_count bigint,
    template_id text,
    scheduled_at timestamptz,
    timezone text,
    estimated_cost_minor bigint,
    currency text,
    status text,
    created_at timestamptz,
    updated_at timestamptz
);

CREATE TABLE agent_workflow.workflow_node_execution (
    execution_id uuid,
    workflow_id uuid,
    event_id uuid,
    draft_version bigint,
    node_name text,
    attempt_no integer,
    tool_name text,
    model_name text,
    prompt_name text,
    prompt_version text,
    trace_id text,
    status text,
    input_json jsonb,
    output_json jsonb,
    error_code text,
    started_at timestamptz,
    finished_at timestamptz
);

CREATE INDEX aw_session_id_idx
    ON agent_workflow.workflow_session (session_id);

CREATE INDEX aw_workflow_id_idx
    ON agent_workflow.workflow_instance (workflow_id);
CREATE INDEX aw_workflow_session_updated_idx
    ON agent_workflow.workflow_instance (session_id, updated_at DESC);

CREATE INDEX aw_draft_workflow_version_idx
    ON agent_workflow.activity_draft (workflow_id, draft_version);

CREATE INDEX aw_event_id_idx
    ON agent_workflow.workflow_event (event_id);
CREATE INDEX aw_event_source_idx
    ON agent_workflow.workflow_event (source, source_event_id);
CREATE INDEX aw_event_session_created_idx
    ON agent_workflow.workflow_event (session_id, created_at, event_id);
CREATE INDEX aw_event_workflow_created_idx
    ON agent_workflow.workflow_event (workflow_id, created_at, event_id);
CREATE INDEX aw_event_due_idx
    ON agent_workflow.workflow_event (status, next_attempt_at);
CREATE INDEX aw_event_lease_idx
    ON agent_workflow.workflow_event (status, lease_expires_at);

CREATE INDEX aw_transition_workflow_revision_idx
    ON agent_workflow.workflow_transition (workflow_id, state_revision);

CREATE INDEX aw_approval_id_idx
    ON agent_workflow.approval_request (approval_id);
CREATE INDEX aw_approval_workflow_version_idx
    ON agent_workflow.approval_request (workflow_id, draft_version);
CREATE INDEX aw_approval_status_created_idx
    ON agent_workflow.approval_request (status, created_at);

CREATE INDEX aw_create_key_idx
    ON agent_workflow.activity_create_operation (idempotency_key);
CREATE INDEX aw_create_workflow_version_idx
    ON agent_workflow.activity_create_operation (workflow_id, draft_version);
CREATE INDEX aw_create_due_idx
    ON agent_workflow.activity_create_operation (status, next_attempt_at);
CREATE INDEX aw_create_lease_idx
    ON agent_workflow.activity_create_operation (status, lease_expires_at);

CREATE INDEX aw_activity_id_idx
    ON agent_workflow.marketing_activity (activity_id);
CREATE INDEX aw_activity_key_idx
    ON agent_workflow.marketing_activity (idempotency_key);

CREATE INDEX aw_node_execution_id_idx
    ON agent_workflow.workflow_node_execution (execution_id);
CREATE INDEX aw_node_workflow_started_idx
    ON agent_workflow.workflow_node_execution (workflow_id, started_at, execution_id);

COMMIT;
