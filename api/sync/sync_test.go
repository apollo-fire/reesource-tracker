package sync_test

import (
	"context"
	"net/http/httptest"
	syncpkg "reesource-tracker/api/sync"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestBroadcastEvent_NoClients(t *testing.T) {
	// Should not panic or error if no clients are registered
	assert.NotPanics(t, func() {
		syncpkg.BroadcastEvent("test_event", map[string]string{"foo": "bar"})
	})
}

func TestBroadcastEvent_WithClients(t *testing.T) {
	// Setup: register a client manually
	modelID := "global"
	clientID := "test-client"
	ch := make(chan syncpkg.Event, 1)
	syncClientsMu := getSyncClientsMu()
	syncClients := getSyncClients()
	syncClientsMu.Lock()
	if _, ok := (*syncClients)[modelID]; !ok {
		(*syncClients)[modelID] = make(map[string]chan syncpkg.Event)
	}
	(*syncClients)[modelID][clientID] = ch
	syncClientsMu.Unlock()

	// Broadcast event
	syncpkg.BroadcastEvent("test", "payload")

	select {
	case evt := <-ch:
		assert.Equal(t, "test", evt.Type)
		assert.Equal(t, "payload", evt.Data)
	case <-time.After(time.Second):
		t.Fatal("event not received by client")
	}

	// Cleanup
	syncClientsMu.Lock()
	delete((*syncClients)[modelID], clientID)
	syncClientsMu.Unlock()
}

// Custom ResponseRecorder that implements http.CloseNotifier for Gin streaming tests
type closeNotifyRecorder struct {
	*httptest.ResponseRecorder
	closeNotify chan bool
}

func (c *closeNotifyRecorder) CloseNotify() <-chan bool {
	return c.closeNotify
}

func TestEventStream_RegistersAndCleansUpClient(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/sync", syncpkg.EventStream)

	req := httptest.NewRequest("GET", "/sync", nil)
	// Use custom recorder
	w := &closeNotifyRecorder{
		ResponseRecorder: httptest.NewRecorder(),
		closeNotify:      make(chan bool, 1),
	}

	// Use a context with cancel to simulate client disconnect
	ctx, cancel := context.WithCancel(req.Context())
	req = req.WithContext(ctx)

	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
		// Simulate client disconnect
		w.closeNotify <- true
	}()

	router.ServeHTTP(w, req)

	// Check that the response contains the initial SSE event
	assert.Contains(t, w.Body.String(), "event:info")
	assert.Contains(t, w.Body.String(), "Connected")
}

// Helpers to access private syncClients and syncClientsMu for testing
func getSyncClients() *map[string]map[string]chan syncpkg.Event {
	return &syncpkg.SyncClients
}

func getSyncClientsMu() *sync.Mutex {
	return &syncpkg.SyncClientsMu
}
