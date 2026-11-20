package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"gorm.io/gorm"
)

var (
	ErrInvalidArgument = errors.New("invalid repository argument")
	ErrAlreadyExists   = errors.New("record already exists")
	ErrConflict        = errors.New("record no longer matches the expected state")
	ErrDataIntegrity   = errors.New("multiple records for a logical key")
	ErrLockOrder       = errors.New("advisory locks acquired out of order")
)

// Store 提供 PostgreSQL 查询及事务入口。
type Store struct {
	*queries
}

type queries struct {
	db *gorm.DB
}

func NewStore(db *gorm.DB) *Store {
	return &Store{queries: &queries{db: db}}
}

// Tx 在同一事务中提供查询和写入。跨表操作应先按顺序获取所需的逻辑锁。
type Tx struct {
	*queries
	locks    map[lockKey]struct{}
	lastLock lockKey
}

// Transaction 使用 READ COMMITTED。回调返回错误时回滚，Tx 仅在回调期间有效。
func (s *Store) Transaction(ctx context.Context, fn func(*Tx) error) error {
	return s.db.WithContext(ctx).Transaction(func(db *gorm.DB) error {
		return fn(&Tx{queries: &queries{db: db}, locks: make(map[lockKey]struct{})})
	}, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
}

func findOne[T any](query *gorm.DB) (T, error) {
	var zero T
	var rows []T
	if err := query.Limit(2).Find(&rows).Error; err != nil {
		return zero, err
	}
	switch len(rows) {
	case 0:
		return zero, gorm.ErrRecordNotFound
	case 1:
		return rows[0], nil
	default:
		return zero, fmt.Errorf("%w: %T", ErrDataIntegrity, zero)
	}
}

func insertUnique[T any](db *gorm.DB, keyQuery *gorm.DB, row *T) error {
	_, err := findOne[T](keyQuery)
	switch {
	case err == nil:
		return ErrAlreadyExists
	case errors.Is(err, gorm.ErrRecordNotFound):
		return db.Create(row).Error
	default:
		return err
	}
}

func findList[T any](query *gorm.DB, limit int) ([]T, error) {
	if limit <= 0 {
		return nil, fmt.Errorf("%w: limit must be positive", ErrInvalidArgument)
	}
	var rows []T
	err := query.Limit(limit).Find(&rows).Error
	return rows, err
}

func updateOne(result *gorm.DB) error {
	if result.Error != nil {
		return result.Error
	}
	switch result.RowsAffected {
	case 0:
		return ErrConflict
	case 1:
		return nil
	default:
		return ErrDataIntegrity
	}
}
