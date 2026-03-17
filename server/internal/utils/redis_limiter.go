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
		// An actual Redis execution or connection error occurred
		return false, 0, err
	}

	// Lock failed; user is currently on cooldown. Fetch the remaining TTL.
	ttl, err := rdb.TTL(ctx, key).Result()
	if err != nil {
		return false, 0, err
	}

	return false, ttl, nil
}
