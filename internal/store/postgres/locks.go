package postgres

import (
	"context"
	"fmt"
	"hash/fnv"
	"slices"
	"strconv"

	"github.com/jackc/pgx/v5/pgtype"
)

// 命名空间和顺序在所有连接间固定。哈希冲突只会使不同业务键串行。
const (
	lockSession int32 = 0x41570001 + iota
	lockEvent
	lockWorkflow
	lockCreation
	lockActivity
)

// EventKey 是入站事件的来源及上游标识，消费者和生产者使用同一组键。
type EventKey struct {
	Source        string
	SourceEventID string
}

func (k EventKey) value() string {
	return strconv.Itoa(len(k.Source)) + ":" + k.Source + k.SourceEventID
}

type lockKey struct {
	namespace int32
	hash      int32
}

// LockSessions 锁定会话。跨表操作依次获取会话、事件、工作流、创建操作和活动锁。
func (tx *Tx) LockSessions(ctx context.Context, ids ...pgtype.UUID) error {
	return tx.lockUUIDs(ctx, lockSession, ids)
}

// LockEvents 按稳定顺序锁定一组来源事件。
func (tx *Tx) LockEvents(ctx context.Context, events ...EventKey) error {
	keys := make([]string, 0, len(events))
	for _, event := range events {
		if event.Source == "" || event.SourceEventID == "" {
			return fmt.Errorf("%w: event source and source event ID are required", ErrInvalidArgument)
		}
		keys = append(keys, event.value())
	}
	return tx.lock(ctx, lockEvent, keys)
}

// LockWorkflows 锁定工作流及其草稿、审批和状态审计。
func (tx *Tx) LockWorkflows(ctx context.Context, ids ...pgtype.UUID) error {
	return tx.lockUUIDs(ctx, lockWorkflow, ids)
}

// LockActivityCreations 按幂等键锁定创建及核查操作。
func (tx *Tx) LockActivityCreations(ctx context.Context, keys ...string) error {
	return tx.lock(ctx, lockCreation, keys)
}

// LockActivities 按幂等键锁定模拟活动服务的写入结果。
func (tx *Tx) LockActivities(ctx context.Context, keys ...string) error {
	return tx.lock(ctx, lockActivity, keys)
}

func (tx *Tx) lockUUIDs(ctx context.Context, namespace int32, ids []pgtype.UUID) error {
	keys := make([]string, 0, len(ids))
	for _, id := range ids {
		if !id.Valid {
			return fmt.Errorf("%w: lock requires a UUID", ErrInvalidArgument)
		}
		keys = append(keys, string(id.Bytes[:]))
	}
	return tx.lock(ctx, namespace, keys)
}

func (tx *Tx) lock(ctx context.Context, namespace int32, keys []string) error {
	hashes := make([]int32, 0, len(keys))
	for _, key := range keys {
		if key == "" {
			return fmt.Errorf("%w: lock key is empty", ErrInvalidArgument)
		}
		hash := fnv.New32a()
		_, _ = hash.Write([]byte(key))
		hashes = append(hashes, int32(hash.Sum32()))
	}
	slices.Sort(hashes)
	for _, hash := range hashes {
		key := lockKey{namespace: namespace, hash: hash}
		if _, held := tx.locks[key]; held {
			continue
		}
		if key.namespace < tx.lastLock.namespace || (key.namespace == tx.lastLock.namespace && key.hash < tx.lastLock.hash) {
			return ErrLockOrder
		}
		if err := tx.db.WithContext(ctx).Exec("SELECT pg_advisory_xact_lock(?::integer, ?::integer)", key.namespace, key.hash).Error; err != nil {
			return err
		}
		tx.locks[key] = struct{}{}
		tx.lastLock = key
	}
	return nil
}
