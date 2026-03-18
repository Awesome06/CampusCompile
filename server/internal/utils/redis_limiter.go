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

		// Lock failed; fetch the remaining TTL
		pttl, err := rdb.PTTL(ctx, key).Result()
		if err != nil {
			return false, 0, err
		}

		// go-redis maps Redis's -2 response to -2 * time.Nanosecond
		if pttl == -2*time.Nanosecond {
			// Key expired right after our SetArgs failed. Loop around and retry.
			continue
		}

		// go-redis maps Redis's -1 response to -1 * time.Nanosecond
		if pttl == -1*time.Nanosecond {
			// Infinite lock bug: Key exists but has no expiration.
			// Repair the state and catch potential errors.
			if err := rdb.Expire(ctx, key, duration).Err(); err != nil {
				return false, 0, fmt.Errorf("failed to repair infinite lock: %w", err)
			}
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
