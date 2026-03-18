package utils

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// EnforceCooldown atomically checks and sets a cooldown for a specific user action.
// Returns true if the action is allowed, false if the user is on cooldown.
func EnforceCooldown(ctx context.Context, rdb *redis.Client, userID string, action string, duration time.Duration) (bool, time.Duration, error) {
	if duration <= 0 {
		return true, 0, nil
	}

	key := fmt.Sprintf("cooldown:%s:%s", action, userID)

	for attempts := 0; attempts < 2; attempts++ {
		err := rdb.SetArgs(ctx, key, "locked", redis.SetArgs{
			Mode: "NX",
			TTL:  duration,
		}).Err()

		if err == nil {
			return true, 0, nil
		}

		if err != redis.Nil {
			return false, 0, err
		}

		pttl, err := rdb.PTTL(ctx, key).Result()
		if err != nil {
			return false, 0, err
		}

		if pttl == -2*time.Millisecond {
			continue
		}

		if pttl == -1*time.Millisecond {
			if err := rdb.Expire(ctx, key, duration).Err(); err != nil {
				return false, 0, fmt.Errorf("failed to repair infinite lock: %w", err)
			}
			return false, duration, nil
		}

		if pttl <= 0 {
			continue
		}

		return false, pttl, nil
	}

	// Fail-open. If the race condition persists beyond retry attempts
	// (e.g., bouncing rapidly between 0 and -2), we cannot definitively prove
	// the lock is active. We allow the request rather than punishing the user.
	return true, 0, nil
}
