package integration_test

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"os"
	"testing"
	"time"

	"Agent-Workflow/internal/store/postgres"
	"Agent-Workflow/internal/store/postgres/model"
	"github.com/jackc/pgx/v5/pgtype"
	pgdriver "gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func postgresStore(t *testing.T) (*postgres.Store, *gorm.DB, context.Context) {
	t.Helper()
	dsn := os.Getenv("AGENT_WORKFLOW_TEST_DSN")
	if dsn == "" {
		t.Skip("AGENT_WORKFLOW_TEST_DSN must point to an initialized disposable PostgreSQL database")
	}
	db, err := gorm.Open(pgdriver.Open(dsn))
	if err != nil {
		t.Fatal(err)
	}
	pool, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = pool.Close() })
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	t.Cleanup(cancel)
	return postgres.NewStore(db), db, ctx
}

func testUUID(t *testing.T) pgtype.UUID {
	t.Helper()
	id := pgtype.UUID{Valid: true}
	if _, err := rand.Read(id.Bytes[:]); err != nil {
		t.Fatal(err)
	}
	return id
}

func TestPostgresEventDedupAndLease(t *testing.T) {
	store, _, ctx := postgresStore(t)
	now := time.Now().UTC().Add(-time.Second)
	key := postgres.EventKey{Source: "integration", SourceEventID: testUUID(t).String()}
	event := model.WorkflowEvent{
		EventID: testUUID(t), SessionID: testUUID(t), Source: &key.Source, SourceEventID: &key.SourceEventID,
		RequestHash: new("request"), Payload: json.RawMessage(`{"message":"hi"}`), CreatedAt: &now,
	}
	type delivery struct {
		event   model.WorkflowEvent
		created bool
		err     error
	}
	results := make(chan delivery, 2)
	ready := make(chan struct{})
	for range 2 {
		go func() {
			<-ready
			var result delivery
			result.err = store.Transaction(ctx, func(tx *postgres.Tx) error {
				var err error
				result.event, result.created, err = tx.ReceiveEvent(ctx, event)
				return err
			})
			results <- result
		}()
	}
	close(ready)
	first, second := <-results, <-results
	if first.err != nil || second.err != nil || first.created == second.created || first.event.EventID != second.event.EventID {
		t.Fatalf("unexpected duplicate deliveries: errors %v, %v, inserted %v, %v", first.err, second.err, first.created, second.created)
	}
	lease := postgres.Lease{Token: testUUID(t), ExpiresAt: now.Add(time.Minute)}
	if err := store.Transaction(ctx, func(tx *postgres.Tx) error {
		claimed, err := tx.ClaimEvent(ctx, key, time.Now(), lease)
		if err != nil {
			return err
		}
		if claimed.AttemptCount == nil || *claimed.AttemptCount != 1 || claimed.LeaseToken != lease.Token {
			return errors.New("event was not claimed")
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	result := postgres.EventResult{Status: "APPLIED", ResultJSON: json.RawMessage(`{"done":true}`)}
	wrongToken := testUUID(t)
	if err := store.Transaction(ctx, func(tx *postgres.Tx) error {
		return tx.SaveEventResult(ctx, key, wrongToken, time.Now(), result)
	}); !errors.Is(err, postgres.ErrConflict) {
		t.Fatalf("wrong lease should conflict, got %v", err)
	}
	if err := store.Transaction(ctx, func(tx *postgres.Tx) error {
		return tx.SaveEventResult(ctx, key, lease.Token, time.Now(), result)
	}); err != nil {
		t.Fatal(err)
	}
	finished, err := store.GetEventBySource(ctx, key)
	if err != nil || finished.Status == nil || *finished.Status != "APPLIED" || finished.LeaseToken.Valid || finished.ProcessedAt == nil {
		t.Fatalf("event result did not commit: %+v, %v", finished, err)
	}
	if !json.Valid(finished.Payload) || !json.Valid(finished.ResultJSON) {
		t.Fatal("JSONB payload or result did not round-trip")
	}
}

func TestPostgresWorkflowCASRollback(t *testing.T) {
	store, _, ctx := postgresStore(t)
	now := time.Now().UTC()
	workflow := model.WorkflowInstance{
		WorkflowID: testUUID(t), SessionID: testUUID(t), StateRevision: new(int64(1)),
		Status: new("RUNNING"), CreatedAt: &now, UpdatedAt: &now,
	}
	if err := store.Transaction(ctx, func(tx *postgres.Tx) error {
		return tx.CreateWorkflow(ctx, workflow)
	}); err != nil {
		t.Fatal(err)
	}
	next := workflow
	next.Status = new("WAITING_INPUT")
	if err := store.Transaction(ctx, func(tx *postgres.Tx) error {
		return tx.UpdateWorkflow(ctx, next, 1, "RUNNING")
	}); err != nil {
		t.Fatal(err)
	}
	if err := store.Transaction(ctx, func(tx *postgres.Tx) error {
		if err := tx.AppendTransition(ctx, model.WorkflowTransition{
			WorkflowID: workflow.WorkflowID, StateRevision: new(int64(2)),
		}); err != nil {
			return err
		}
		return tx.UpdateWorkflow(ctx, next, 1, "RUNNING")
	}); !errors.Is(err, postgres.ErrConflict) {
		t.Fatalf("stale revision should conflict, got %v", err)
	}
	persisted, err := store.GetWorkflow(ctx, workflow.WorkflowID)
	if err != nil || persisted.StateRevision == nil || *persisted.StateRevision != 2 {
		t.Fatalf("unexpected workflow revision: %+v, %v", persisted, err)
	}
	transitions, err := store.ListWorkflowTransitions(ctx, workflow.WorkflowID, 0, 10)
	if err != nil || len(transitions) != 0 {
		t.Fatalf("transition should be rolled back: %+v, %v", transitions, err)
	}
}

func TestPostgresCreationRecoversToReconciliation(t *testing.T) {
	store, db, ctx := postgresStore(t)
	now := time.Now().UTC().Add(-time.Second)
	op := model.ActivityCreateOperation{
		WorkflowID: testUUID(t), DraftVersion: new(int64(1)), ConfirmationEventID: testUUID(t), AuthorizationEventID: testUUID(t),
		RequestHash: new("request"), RequestJSON: json.RawMessage(`{"command":"create"}`), CreatedAt: &now,
	}
	var key string
	lease := postgres.Lease{Token: testUUID(t), ExpiresAt: now.Add(time.Minute)}
	if err := store.Transaction(ctx, func(tx *postgres.Tx) error {
		operation, inserted, err := tx.EnqueueActivityCreation(ctx, op)
		if err != nil {
			return err
		}
		if !inserted || operation.IdempotencyKey == nil {
			return errors.New("creation was not queued")
		}
		key = *operation.IdempotencyKey
		_, err = tx.ClaimActivityCreation(ctx, key, time.Now(), lease)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	activityID := testUUID(t).String()
	activity := model.MarketingActivity{
		ActivityID: &activityID, IdempotencyKey: &key, RequestHash: op.RequestHash,
		WorkflowID: op.WorkflowID, DraftVersion: op.DraftVersion, CreatedAt: &now,
	}
	if err := store.Transaction(ctx, func(tx *postgres.Tx) error {
		_, inserted, err := tx.RecordMarketingActivity(ctx, activity, lease.Token)
		if err == nil && !inserted {
			return errors.New("marketing activity was not inserted")
		}
		return err
	}); err != nil {
		t.Fatal(err)
	}
	if err := db.WithContext(ctx).Model(&model.ActivityCreateOperation{}).Where("idempotency_key = ?", key).
		Update("lease_expires_at", now).Error; err != nil {
		t.Fatal(err)
	}
	if err := store.Transaction(ctx, func(tx *postgres.Tx) error {
		return tx.RescheduleExpiredActivityCreation(ctx, key, time.Now())
	}); err != nil {
		t.Fatal(err)
	}
	newLease := postgres.Lease{Token: testUUID(t), ExpiresAt: now.Add(time.Minute)}
	if err := store.Transaction(ctx, func(tx *postgres.Tx) error {
		operation, err := tx.ClaimActivityCreation(ctx, key, time.Now(), newLease)
		if err != nil {
			return err
		}
		if operation.Status == nil || *operation.Status != "RECONCILING" || operation.AttemptCount == nil ||
			*operation.AttemptCount != 1 || operation.ReconcileCount == nil || *operation.ReconcileCount != 1 {
			return errors.New("expired creation was not recovered as reconciliation")
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	result := postgres.ActivityCreationResult{Status: "SUCCEEDED", ActivityID: &activityID}
	if err := store.Transaction(ctx, func(tx *postgres.Tx) error {
		return tx.SaveActivityCreationResult(ctx, key, lease.Token, time.Now(), result)
	}); !errors.Is(err, postgres.ErrConflict) {
		t.Fatalf("expired creation lease should conflict, got %v", err)
	}
	if err := store.Transaction(ctx, func(tx *postgres.Tx) error {
		persisted, inserted, err := tx.RecordMarketingActivity(ctx, activity, lease.Token)
		if err != nil {
			return err
		}
		if inserted || persisted.ActivityID == nil || *persisted.ActivityID != activityID {
			return errors.New("duplicate request did not reuse the original activity")
		}
		return tx.SaveActivityCreationResult(ctx, key, newLease.Token, time.Now(), result)
	}); err != nil {
		t.Fatal(err)
	}
}
