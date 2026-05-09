package hub_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/mferree/agent-city/internal/hub"
	"github.com/mferree/agent-city/internal/model"
)

// TestHubRunExitsOnContextCancel verifies that hub.Run returns when its
// context is cancelled.
func TestHubRunExitsOnContextCancel(t *testing.T) {
	s := hub.NewState(model.CityState{})
	h := hub.New(s)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		h.Run(ctx)
		close(done)
	}()

	cancel()

	select {
	case <-done:
		// expected
	case <-time.After(time.Second):
		t.Fatal("hub.Run did not return within 1s of context cancellation")
	}
}

// TestHubSendsCloseMessageOnShutdown verifies that connected WebSocket clients
// receive a close frame when the hub's context is cancelled.
func TestHubSendsCloseMessageOnShutdown(t *testing.T) {
	s := hub.NewState(model.CityState{})
	h := hub.New(s)

	ctx, cancel := context.WithCancel(context.Background())
	go h.Run(ctx)

	srv := httptest.NewServer(http.HandlerFunc(h.ServeWS))
	t.Cleanup(srv.Close)

	wsURL := "ws" + srv.URL[4:]
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("websocket dial: %v", err)
	}

	// Read the initial state.full snapshot sent on connect.
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	if _, _, err = conn.ReadMessage(); err != nil {
		t.Fatalf("reading initial state: %v", err)
	}

	// Cancel the hub context — should close all client connections promptly.
	cancel()

	// The server should close the connection within a short window, not merely
	// wait for our read deadline to expire. Use a tight deadline so a missing
	// close frame causes a test timeout rather than a silent pass.
	conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
	_, _, err = conn.ReadMessage()
	if err == nil {
		t.Fatal("expected connection to be closed after hub shutdown, got a message instead")
	}
	if netErr, ok := err.(interface{ Timeout() bool }); ok && netErr.Timeout() {
		t.Fatal("read timed out — server did not close the connection after hub shutdown")
	}
	// websocket.CloseMessage, EOF, or connection-reset are all acceptable.
}
