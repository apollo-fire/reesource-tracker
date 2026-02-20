package sync

import (
	"io"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Event struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

var SyncClients = make(map[string]map[string]chan Event)
var SyncClientsMu sync.Mutex

func BroadcastEvent(evtType string, data interface{}) {
	SyncClientsMu.Lock()
	defer SyncClientsMu.Unlock()
	for _, clientMap := range SyncClients {
		for _, ch := range clientMap {
			select {
			case ch <- Event{Type: evtType, Data: data}:
			default:
				// drop if client is slow
			}
		}
	}
}

func EventStream(c *gin.Context) {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Transfer-Encoding", "chunked")
	c.Header("X-Accel-Buffering", "no")

	// For this app, we use a single global channel key
	model_id := "global"
	clientID := c.ClientIP() + uuid.New().String()
	eventChan := make(chan Event, 10)

	SyncClientsMu.Lock()
	if _, ok := SyncClients[model_id]; !ok {
		SyncClients[model_id] = make(map[string]chan Event)
	}
	SyncClients[model_id][clientID] = eventChan
	SyncClientsMu.Unlock()

	defer func() {
		SyncClientsMu.Lock()
		delete(SyncClients[model_id], clientID)
		if len(SyncClients[model_id]) == 0 {
			delete(SyncClients, model_id)
		}
		SyncClientsMu.Unlock()
		close(eventChan)
	}()

	c.SSEvent("info", "Connected")
	c.Writer.Flush()

	c.Stream(func(w io.Writer) bool {
		select {
		case evt := <-eventChan:
			c.SSEvent(evt.Type, evt.Data)
			c.Writer.Flush()
			return true
		case <-c.Request.Context().Done():
			return false
		}
	})
}

// RegisterSyncRoutes adds the /eventstream endpoint to the router group
func Routes(route *gin.RouterGroup) {
	route.GET("/sync", EventStream)
}
