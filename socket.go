package live

import (
	"context"
	"fmt"

	"golang.org/x/net/html"
	"nhooyr.io/websocket"
)

const (
	MaxMessageBufferSize = 16
)

// Socket describes a socket from the outside.
type Socket struct {
	Session Session
	Data    interface{}

	currentRender *html.Node
	msgs          chan Event
	closeSlow     func()
}

func (s *Socket) mount(ctx context.Context, view *View, params map[string]string, connected bool) error {
	// Mount view.
	if err := view.Mount(ctx, view, params, s, connected); err != nil {
		return fmt.Errorf("mount error: %w", err)
	}
	return nil
}

// handleView takes a view and runs a mount and render.
func (s *Socket) handleView(ctx context.Context, view *View, params map[string]string) error {
	// Render view.
	output, err := view.Render(ctx, view.t, s)
	if err != nil {
		return fmt.Errorf("render error: %w", err)
	}
	node, err := html.Parse(output)
	if err != nil {
		return fmt.Errorf("html parse error: %w", err)
	}

	// Get diff
	if s.currentRender != nil {
		patches, err := Diff(s.currentRender, node)
		if err != nil {
			return fmt.Errorf("diff error: %w", err)
		}
		for _, p := range patches {
			msg := Event{
				T:    ETPatch,
				Data: p,
			}
			s.msgs <- msg
		}
	}
	s.currentRender = node

	return nil
}

// NewSocket creates a new socket.
func NewSocket(s Session) *Socket {
	return &Socket{
		Session: s,
		msgs:    make(chan Event, MaxMessageBufferSize),
	}
}

// AssignSocket to a socket.
func (c *Socket) AssignWS(ws *websocket.Conn) {
	c.closeSlow = func() {
		ws.Close(websocket.StatusPolicyViolation, "socket too slow to keep up with messages")
	}
}
