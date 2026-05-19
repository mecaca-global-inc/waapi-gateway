package api

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

func subtleEqual(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

type loginReq struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type loginResp struct {
	APIKey string `json:"api_key"`
}

func hashKey(plain string) string {
	sum := sha256.Sum256([]byte(plain))
	return hex.EncodeToString(sum[:])
}

const adminSessionKey = "admin-session"

func (s *Server) handleLogin(c *fiber.Ctx) error {
	var req loginReq
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "bad json")
	}
	if subtleEqual(req.Username, s.cfg.AdminUser) == false || subtleEqual(req.Password, s.cfg.AdminPass) == false {
		return fiber.NewError(fiber.StatusUnauthorized, "invalid credentials")
	}
	// Rotate the admin-session key: delete any prior row then issue a fresh one.
	if err := s.repo.DeleteAPIKeysByName(c.Context(), adminSessionKey); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	plain := uuid.NewString()
	if _, err := s.repo.AddAPIKey(c.Context(), adminSessionKey, hashKey(plain)); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(loginResp{APIKey: plain})
}

// checkAPIKey returns nil if the given plaintext key is valid.
func (s *Server) checkAPIKey(c *fiber.Ctx, key string) error {
	if key == "" {
		return fiber.NewError(fiber.StatusUnauthorized, "missing api key")
	}
	hash := hashKey(key)
	keys, err := s.repo.ListAPIKeys(c.Context())
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	for _, k := range keys {
		if subtle.ConstantTimeCompare([]byte(k.KeyHash), []byte(hash)) == 1 {
			_ = s.repo.TouchAPIKey(c.Context(), hash)
			return nil
		}
	}
	return fiber.NewError(fiber.StatusUnauthorized, "invalid api key")
}

func (s *Server) authMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		auth := c.Get("Authorization")
		key := c.Get("X-Api-Key")
		if key == "" && strings.HasPrefix(auth, "Bearer ") {
			key = strings.TrimPrefix(auth, "Bearer ")
		}
		if err := s.checkAPIKey(c, key); err != nil {
			return err
		}
		return c.Next()
	}
}

func (s *Server) listKeys(c *fiber.Ctx) error {
	keys, err := s.repo.ListAPIKeys(c.Context())
	if err != nil {
		return err
	}
	return c.JSON(keys)
}

type createKeyReq struct {
	Name string `json:"name"`
}

func (s *Server) createKey(c *fiber.Ctx) error {
	var req createKeyReq
	if err := c.BodyParser(&req); err != nil || req.Name == "" {
		return fiber.NewError(fiber.StatusBadRequest, "name required")
	}
	plain := uuid.NewString()
	k, err := s.repo.AddAPIKey(c.Context(), req.Name, hashKey(plain))
	if err != nil {
		return err
	}
	return c.JSON(fiber.Map{"id": k.ID, "name": k.Name, "api_key": plain})
}

func (s *Server) deleteKey(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "bad id")
	}
	if err := s.repo.DeleteAPIKey(c.Context(), int64(id)); err != nil {
		return err
	}
	return c.JSON(fiber.Map{"ok": true})
}
