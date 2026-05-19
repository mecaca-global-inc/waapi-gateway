package api

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/mecaca/waapi-gateway/internal/wa"
	"go.mau.fi/whatsmeow/types"
)

func (s *Server) resolveSession(c *fiber.Ctx, name string) (*wa.Session, error) {
	if name == "" {
		return nil, fiber.NewError(fiber.StatusBadRequest, "session is required")
	}
	sess, err := s.mgr.Get(name)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusNotFound, err.Error())
	}
	if !sess.Client.IsConnected() {
		return nil, fiber.NewError(fiber.StatusServiceUnavailable, "session not connected")
	}
	return sess, nil
}

// parseJID accepts "628123456789", "628123456789@s.whatsapp.net" or full JID strings.
func parseJID(in string) (types.JID, error) {
	in = strings.TrimSpace(in)
	if in == "" {
		return types.JID{}, fiber.NewError(fiber.StatusBadRequest, "chat id required")
	}
	if !strings.Contains(in, "@") {
		// default to user JID if a phone number is supplied
		in = in + "@s.whatsapp.net"
	}
	return types.ParseJID(in)
}
