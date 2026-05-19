package api

import (
	"time"

	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	fiberlog "github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"

	"github.com/mecaca/waapi-gateway/internal/config"
	"github.com/mecaca/waapi-gateway/internal/store"
	"github.com/mecaca/waapi-gateway/internal/wa"
)

type Server struct {
	app  *fiber.App
	cfg  *config.Config
	repo *store.Repo
	mgr  *wa.Manager
}

func NewServer(cfg *config.Config, repo *store.Repo, mgr *wa.Manager) *Server {
	app := fiber.New(fiber.Config{
		AppName:               "waapi-gateway",
		DisableStartupMessage: true,
		BodyLimit:             64 * 1024 * 1024, // 64 MB upload cap
	})
	app.Use(recover.New())
	app.Use(fiberlog.New())
	app.Use(cors.New(cors.Config{
		AllowOrigins:     joinOrigins(cfg.CORSOrigins),
		AllowHeaders:     "Authorization,Content-Type,X-Api-Key",
		AllowMethods:     "GET,POST,PUT,DELETE,OPTIONS",
		AllowCredentials: false,
	}))
	s := &Server{app: app, cfg: cfg, repo: repo, mgr: mgr}
	s.routes()
	return s
}

func (s *Server) App() *fiber.App { return s.app }

func (s *Server) Listen(addr string) error { return s.app.Listen(addr) }

func (s *Server) Shutdown() error { return s.app.Shutdown() }

func (s *Server) routes() {
	s.app.Get("/healthz", func(c *fiber.Ctx) error { return c.JSON(fiber.Map{"ok": true}) })
	s.app.Get("/readyz", func(c *fiber.Ctx) error { return c.JSON(fiber.Map{"ok": true}) })

	// OpenAPI spec + Swagger UI
	s.docsRoutes()

	// Login endpoint (basic auth -> issues an API key in response). Rate-limited.
	loginLimiter := limiter.New(limiter.Config{
		Max:        10,
		Expiration: time.Minute,
		KeyGenerator: func(c *fiber.Ctx) string { return c.IP() },
		LimitReached: func(c *fiber.Ctx) error {
			return fiber.NewError(fiber.StatusTooManyRequests, "too many login attempts")
		},
	})
	s.app.Post("/api/login", loginLimiter, s.handleLogin)

	api := s.app.Group("/api", s.authMiddleware())

	// Sessions
	api.Get("/sessions", s.listSessions)
	api.Post("/sessions", s.createSession)
	api.Get("/sessions/:name", s.getSession)
	api.Delete("/sessions/:name", s.deleteSession)
	api.Post("/sessions/:name/start", s.startSession)
	api.Post("/sessions/:name/stop", s.stopSession)
	api.Post("/sessions/:name/logout", s.logoutSession)

	// Auth (QR / pairing)
	api.Get("/:session/auth/qr", s.getQR)
	api.Get("/:session/auth/qr.png", s.getQRPNG)
	api.Post("/:session/auth/request-code", s.requestPairCode)

	// Messaging
	api.Post("/sendText", s.sendText)
	api.Post("/sendImage", s.sendImage)
	api.Post("/sendFile", s.sendFile)
	api.Post("/sendVoice", s.sendVoice)
	api.Post("/sendVideo", s.sendVideo)
	api.Post("/sendLocation", s.sendLocation)
	api.Post("/sendContact", s.sendContact)
	api.Post("/sendSeen", s.sendSeen)
	api.Post("/startTyping", s.startTyping)
	api.Post("/stopTyping", s.stopTyping)

	// Read
	api.Get("/:session/me", s.getMe)
	api.Get("/:session/contacts", s.getContacts)
	api.Get("/:session/chats", s.getChats)
	api.Get("/:session/groups", s.getGroups)

	// Group admin
	api.Post("/:session/groups", s.createGroup)
	api.Post("/:session/groups/join", s.joinGroup)
	api.Get("/:session/groups/:gid", s.getGroupInfo)
	api.Post("/:session/groups/:gid/leave", s.leaveGroup)
	api.Post("/:session/groups/:gid/participants/add", s.addParticipants)
	api.Post("/:session/groups/:gid/participants/remove", s.removeParticipants)
	api.Post("/:session/groups/:gid/participants/promote", s.promoteParticipants)
	api.Post("/:session/groups/:gid/participants/demote", s.demoteParticipants)
	api.Put("/:session/groups/:gid/name", s.setGroupName)
	api.Put("/:session/groups/:gid/topic", s.setGroupTopic)
	api.Put("/:session/groups/:gid/locked", s.setGroupLocked)
	api.Put("/:session/groups/:gid/announce", s.setGroupAnnounce)
	api.Put("/:session/groups/:gid/photo", s.setGroupPhoto)
	api.Put("/:session/groups/:gid/disappearing", s.setGroupDisappearing)
	api.Get("/:session/groups/:gid/invite-link", s.getInviteLink)
	api.Post("/:session/groups/:gid/invite-link/revoke", s.revokeInviteLink)

	// Webhooks
	api.Get("/:session/webhooks", s.listWebhooks)
	api.Post("/:session/webhooks", s.addWebhook)
	api.Delete("/webhooks/:id", s.deleteWebhook)

	// API keys
	api.Get("/keys", s.listKeys)
	api.Post("/keys", s.createKey)
	api.Delete("/keys/:id", s.deleteKey)

	// WebSocket — auth via ?key= query param (browsers can't set headers on WS upgrade).
	s.app.Use("/ws", func(c *fiber.Ctx) error {
		if !websocket.IsWebSocketUpgrade(c) {
			return fiber.ErrUpgradeRequired
		}
		if err := s.checkAPIKey(c, c.Query("key")); err != nil {
			return err
		}
		return c.Next()
	})
	s.app.Get("/ws", websocket.New(s.handleWS))
}

func joinOrigins(o []string) string {
	if len(o) == 0 {
		return "*"
	}
	out := o[0]
	for _, x := range o[1:] {
		out += "," + x
	}
	return out
}
