package api

import (
	"github.com/gofiber/fiber/v2"
)

func (s *Server) getMe(c *fiber.Ctx) error {
	sess, err := s.mgr.Get(c.Params("session"))
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, err.Error())
	}
	if sess.Client.Store.ID == nil {
		return c.JSON(fiber.Map{"logged_in": false})
	}
	return c.JSON(fiber.Map{
		"logged_in": true,
		"jid":       sess.Client.Store.ID.String(),
		"push_name": sess.Client.Store.PushName,
		"platform":  sess.Client.Store.Platform,
	})
}

func (s *Server) getContacts(c *fiber.Ctx) error {
	sess, err := s.resolveSession(c, c.Params("session"))
	if err != nil {
		return err
	}
	all, cerr := sess.Client.Store.Contacts.GetAllContacts(c.Context())
	if cerr != nil {
		return fiber.NewError(fiber.StatusInternalServerError, cerr.Error())
	}
	out := make([]fiber.Map, 0, len(all))
	for jid, ci := range all {
		out = append(out, fiber.Map{
			"jid":       jid.String(),
			"push_name": ci.PushName,
			"full_name": ci.FullName,
			"first":     ci.FirstName,
			"business":  ci.BusinessName,
		})
	}
	return c.JSON(out)
}

func (s *Server) getChats(c *fiber.Ctx) error {
	// whatsmeow does not maintain a chat list cache; expose contacts as a proxy.
	return s.getContacts(c)
}

func (s *Server) getGroups(c *fiber.Ctx) error {
	sess, err := s.resolveSession(c, c.Params("session"))
	if err != nil {
		return err
	}
	groups, gerr := sess.Client.GetJoinedGroups(c.Context())
	if gerr != nil {
		return fiber.NewError(fiber.StatusInternalServerError, gerr.Error())
	}
	out := make([]fiber.Map, 0, len(groups))
	for _, g := range groups {
		out = append(out, fiber.Map{
			"jid":     g.JID.String(),
			"name":    g.GroupName.Name,
			"topic":   g.GroupTopic.Topic,
			"members": len(g.Participants),
		})
	}
	return c.JSON(out)
}
