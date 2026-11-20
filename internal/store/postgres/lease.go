package postgres

import (
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"gorm.io/gorm"
)

// Lease 标识一次任务认领及租约到期时间。
type Lease struct {
	Token     pgtype.UUID
	ExpiresAt time.Time
}

func (l Lease) validate(at time.Time) error {
	if !l.Token.Valid || at.IsZero() || !l.ExpiresAt.After(at) {
		return fmt.Errorf("%w: lease requires a token and a future expiry", ErrInvalidArgument)
	}
	return nil
}

func withActiveLease(query *gorm.DB, token pgtype.UUID, at time.Time) *gorm.DB {
	return query.Where("lease_token = ? AND lease_expires_at > GREATEST(?, clock_timestamp())", token, at)
}
