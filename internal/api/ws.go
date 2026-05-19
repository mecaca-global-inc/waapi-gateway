package api

import (
	"encoding/json"
	"time"

	"github.com/gofiber/contrib/websocket"
	"github.com/rs/zerolog/log"
)

func (s *Server) handleWS(c *websocket.Conn) {
	defer c.Close()
	ch := s.mgr.Subscribe()
	defer s.mgr.Unsubscribe(ch)

	// ping loop
	pingTicker := time.NewTicker(30 * time.Second)
	defer pingTicker.Stop()

	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			if _, _, err := c.ReadMessage(); err != nil {
				return
			}
		}
	}()

	for {
		select {
		case <-done:
			return
		case <-pingTicker.C:
			if err := c.WriteControl(websocket.PingMessage, nil, time.Now().Add(5*time.Second)); err != nil {
				return
			}
		case ev, ok := <-ch:
			if !ok {
				return
			}
			b, err := json.Marshal(ev)
			if err != nil {
				log.Error().Err(err).Msg("ws marshal failed")
				continue
			}
			if err := c.WriteMessage(websocket.TextMessage, b); err != nil {
				return
			}
		}
	}
}
