package store

import (
	"context"
	"testing"
	"time"
)

func TestRunRetentionRejectsUnsafePolicyBeforeDatabaseAccess(t *testing.T) {
	database := &Mongo{}
	if _, err := database.RunRetention(context.Background(), time.Now(), RetentionPolicy{}); err == nil {
		t.Fatal("expected zero-day retention policy to fail closed")
	}
}
