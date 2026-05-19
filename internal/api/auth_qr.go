package api

import (
	"bytes"

	"github.com/gofiber/fiber/v2"
	qrcode "github.com/skip2/go-qrcode"
)

func (s *Server) getQR(c *fiber.Ctx) error {
	sess, err := s.mgr.Get(c.Params("session"))
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, err.Error())
	}
	code := sess.LastQR()
	if code == "" {
		return c.JSON(fiber.Map{"status": string(sess.Status()), "code": ""})
	}
	return c.JSON(fiber.Map{"status": string(sess.Status()), "code": code})
}

func (s *Server) getQRPNG(c *fiber.Ctx) error {
	sess, err := s.mgr.Get(c.Params("session"))
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, err.Error())
	}
	code := sess.LastQR()
	if code == "" {
		return fiber.NewError(fiber.StatusNotFound, "no qr available")
	}
	var buf bytes.Buffer
	q, qerr := qrcode.New(code, qrcode.Medium)
	if qerr != nil {
		return fiber.NewError(fiber.StatusInternalServerError, qerr.Error())
	}
	if err := q.Write(256, &buf); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	c.Set("Content-Type", "image/png")
	return c.Send(buf.Bytes())
}

type pairCodeReq struct {
	Phone string `json:"phone"`
}

func (s *Server) requestPairCode(c *fiber.Ctx) error {
	sess, err := s.mgr.Get(c.Params("session"))
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, err.Error())
	}
	var req pairCodeReq
	if err := c.BodyParser(&req); err != nil || req.Phone == "" {
		return fiber.NewError(fiber.StatusBadRequest, "phone required (E.164, no +)")
	}
	code, err := sess.PairPhone(c.Context(), req.Phone)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(fiber.Map{"code": code})
}
