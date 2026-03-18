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
	// 1. Guard against zero/negative durations causing permanent locks
	if duration <= 0 {
		return true, 0, nil
	}

	key := fmt.Sprintf("cooldown:%s:%s", action, userID)

	// Modern go-redis v9 approach: Use SetArgs to pass the NX mode.
	err := rdb.SetArgs(ctx, key, "locked", redis.SetArgs{
		Mode: "NX",
		TTL:  duration,
	}).Err()

	if err == nil {
		// Lock acquired successfully; user was not on cooldown.
		return true, 0, nil
	}

	// redis.Nil means the NX condition failed (the key already exists)
	if err != redis.Nil {
		return false, 0, err
	}

	// Lock failed; user is currently on cooldown. Fetch the remaining TTL.
	ttl, err := rdb.TTL(ctx, key).Result()
	if err != nil {
		return false, 0, err
	}

	// Handle race conditions:
	// -2 means the key expired between the SET and TTL check.
	// -1 means the key exists but has no expiry (shouldn't happen with our SetArgs, but good defense).
	if ttl <= 0 {
		return true, 0, nil
	}

	return false, ttl, nil
}
