package middleware

import (
	"testing"
	"time"
)

func TestLockSessionRefreshSerializesSameSession(t *testing.T) {
	unlock := lockSessionRefresh("shared-session")

	acquired := make(chan struct{})
	go func() {
		release := lockSessionRefresh("shared-session")
		close(acquired)
		release()
	}()

	select {
	case <-acquired:
		t.Fatal("second refresh lock acquisition should block for the same session")
	case <-time.After(50 * time.Millisecond):
	}

	unlock()

	select {
	case <-acquired:
	case <-time.After(time.Second):
		t.Fatal("second refresh lock acquisition should resume after the first unlocks")
	}
}

func TestLockSessionRefreshAllowsDifferentSessions(t *testing.T) {
	unlock := lockSessionRefresh("session-a")
	defer unlock()

	acquired := make(chan struct{})
	go func() {
		release := lockSessionRefresh("session-b")
		close(acquired)
		release()
	}()

	select {
	case <-acquired:
	case <-time.After(time.Second):
		t.Fatal("different sessions should not block each other while refreshing")
	}
}
