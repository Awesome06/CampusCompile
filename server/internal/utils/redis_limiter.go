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
	// Guard against zero/negative durations
	if duration <= 0 {
		return true, 0, nil
	}

	key := fmt.Sprintf("cooldown:%s:%s", action, userID)

	// Retry loop to handle the edge case where the key expires
	// exactly between the SetArgs and PTTL calls (-2 race condition).
	for attempts := 0; attempts < 2; attempts++ {
		err := rdb.SetArgs(ctx, key, "locked", redis.SetArgs{
			Mode: "NX",
			TTL:  duration,
		}).Err()

		if err == nil {
			// Lock acquired successfully
			return true, 0, nil
		}

		if err != redis.Nil {
			// Actual Redis execution/connection error
			return false, 0, err
		}

		// Lock failed; fetch the remaining TTL in milliseconds
		pttl, err := rdb.PTTL(ctx, key).Result()
		if err != nil {
			return false, 0, err
		}

		if pttl == -2 {
			// Key expired right after our SetArgs failed.
			// Loop around and try to acquire the lock again.
			continue
		}

		if pttl == -1 {
			// Infinite lock bug: Key exists but has no expiration.
			// Repair the state by enforcing the intended duration.
			rdb.Expire(ctx, key, duration)
			return false, duration, nil
		}

		if pttl > 0 {
			// User is genuinely on cooldown
			return false, pttl, nil
		}
	}

	// Fallback if the race condition persists beyond retry attempts
	return false, duration, nil
}
