package api

import (
	"github.com/gofiber/fiber/v2"
	"github.com/mecaca/waapi-gateway/internal/store"
	"github.com/mecaca/waapi-gateway/internal/wa"
)

type sessionDTO struct {
	Name   string `json:"name"`
	JID    string `json:"jid"`
	Status string `json:"status"`
}

type createSessionReq struct {
	Name string `json:"name"`
}

func (s *Server) listSessions(c *fiber.Ctx) error {
	rows, err := s.repo.ListSessions(c.Context())
	if err != nil {
		return err
	}
	out := make([]sessionDTO, 0, len(rows))
	for _, r := range rows {
		live := r.Status
		if sess, gerr := s.mgr.Get(r.Name); gerr == nil {
			live = string(sess.Status())
		}
		out = append(out, sessionDTO{Name: r.Name, JID: r.JID, Status: live})
	}
	return c.JSON(out)
}

func (s *Server) createSession(c *fiber.Ctx) error {
	var req createSessionReq
	if err := c.BodyParser(&req); err != nil || req.Name == "" {
		return fiber.NewError(fiber.StatusBadRequest, "name required")
	}
	sess, err := s.mgr.Create(c.Context(), req.Name)
	if err != nil {
		if err == wa.ErrSessionExists {
			return fiber.NewError(fiber.StatusConflict, err.Error())
		}
		return err
	}
	return c.Status(fiber.StatusCreated).JSON(sessionDTO{
		Name:   sess.Name,
		Status: string(sess.Status()),
	})
}

func (s *Server) getSession(c *fiber.Ctx) error {
	name := c.Params("name")
	row, err := s.repo.GetSession(c.Context(), name)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "not found")
	}
	live := row.Status
	if sess, gerr := s.mgr.Get(name); gerr == nil {
		live = string(sess.Status())
	}
	return c.JSON(sessionDTO{Name: row.Name, JID: row.JID, Status: live})
}

func (s *Server) deleteSession(c *fiber.Ctx) error {
	name := c.Params("name")
	if err := s.mgr.Delete(c.Context(), name); err != nil {
		return err
	}
	return c.JSON(fiber.Map{"ok": true})
}

func (s *Server) startSession(c *fiber.Ctx) error {
	sess, err := s.mgr.Get(c.Params("name"))
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, err.Error())
	}
	if err := sess.Start(c.Context()); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(sessionDTO{Name: sess.Name, Status: string(sess.Status())})
}

func (s *Server) stopSession(c *fiber.Ctx) error {
	sess, err := s.mgr.Get(c.Params("name"))
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, err.Error())
	}
	sess.Stop()
	return c.JSON(sessionDTO{Name: sess.Name, Status: string(sess.Status())})
}

func (s *Server) logoutSession(c *fiber.Ctx) error {
	sess, err := s.mgr.Get(c.Params("name"))
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, err.Error())
	}
	if err := sess.Logout(c.Context()); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(sessionDTO{Name: sess.Name, Status: string(sess.Status())})
}

// Webhooks per session

type addWebhookReq struct {
	URL     string   `json:"url"`
	Secret  string   `json:"secret"`
	Events  []string `json:"events"`
	Enabled *bool    `json:"enabled"`
}

func (s *Server) listWebhooks(c *fiber.Ctx) error {
	session := c.Params("session")
	hooks, err := s.repo.ListWebhooksBySession(c.Context(), session)
	if err != nil {
		return err
	}
	return c.JSON(hooks)
}

func (s *Server) addWebhook(c *fiber.Ctx) error {
	session := c.Params("session")
	var req addWebhookReq
	if err := c.BodyParser(&req); err != nil || req.URL == "" {
		return fiber.NewError(fiber.StatusBadRequest, "url required")
	}
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	w := &store.Webhook{
		SessionName: session,
		URL:         req.URL,
		Secret:      req.Secret,
		Events:      req.Events,
		Enabled:     enabled,
	}
	if err := s.repo.AddWebhook(c.Context(), w); err != nil {
		return err
	}
	return c.Status(fiber.StatusCreated).JSON(w)
}

func (s *Server) deleteWebhook(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "bad id")
	}
	if err := s.repo.DeleteWebhook(c.Context(), int64(id)); err != nil {
		return err
	}
	return c.JSON(fiber.Map{"ok": true})
}
