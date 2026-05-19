package api

import (
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"google.golang.org/protobuf/proto"
)

type sendBase struct {
	Session string `json:"session"`
	ChatID  string `json:"chat_id"`
	// Expiration (seconds). When > 0 the outgoing message is marked as
	// disappearing with this timer. If omitted, the gateway auto-populates from
	// the last observed ephemeral setting for this chat (so replies in a
	// disappearing-messages thread inherit it). Pass `-1` (or any negative
	// value) to explicitly force a non-ephemeral message.
	Expiration *int32 `json:"expiration,omitempty"`
}

func (s *Server) effectiveExpiration(session string, chat types.JID, override *int32) uint32 {
	if override != nil {
		if *override <= 0 {
			return 0
		}
		return uint32(*override)
	}
	return s.mgr.ChatExpiration(session, chat)
}

type sendTextReq struct {
	sendBase
	Text string `json:"text"`
}

func (s *Server) sendText(c *fiber.Ctx) error {
	var req sendTextReq
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "bad json")
	}
	if req.Text == "" {
		return fiber.NewError(fiber.StatusBadRequest, "text required")
	}
	sess, err := s.resolveSession(c, req.Session)
	if err != nil {
		return err
	}
	jid, err := parseJID(req.ChatID)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}
	exp := s.effectiveExpiration(req.Session, jid, req.Expiration)
	var msg *waE2E.Message
	if exp > 0 {
		msg = &waE2E.Message{ExtendedTextMessage: &waE2E.ExtendedTextMessage{
			Text:        proto.String(req.Text),
			ContextInfo: &waE2E.ContextInfo{Expiration: proto.Uint32(exp)},
		}}
	} else {
		msg = &waE2E.Message{Conversation: proto.String(req.Text)}
	}
	resp, err := sess.Client.SendMessage(c.Context(), jid, msg)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(fiber.Map{"id": resp.ID, "timestamp": resp.Timestamp.Unix()})
}

type mediaReq struct {
	sendBase
	URL      string `json:"url"`
	Caption  string `json:"caption"`
	Mimetype string `json:"mimetype"`
	Filename string `json:"filename"`
}

func (s *Server) readMediaPayload(c *fiber.Ctx) (req mediaReq, data []byte, err error) {
	ct := c.Get("Content-Type")
	if strings.HasPrefix(ct, "multipart/form-data") {
		req.Session = c.FormValue("session")
		req.ChatID = c.FormValue("chat_id")
		req.Caption = c.FormValue("caption")
		req.Mimetype = c.FormValue("mimetype")
		req.Filename = c.FormValue("filename")
		if v := c.FormValue("expiration"); v != "" {
			if n, perr := strconv.Atoi(v); perr == nil {
				exp32 := int32(n)
				req.Expiration = &exp32
			}
		}
		file, herr := c.FormFile("file")
		if herr != nil {
			err = fiber.NewError(fiber.StatusBadRequest, "file required")
			return
		}
		fh, oerr := file.Open()
		if oerr != nil {
			err = oerr
			return
		}
		defer fh.Close()
		data, err = io.ReadAll(fh)
		if req.Filename == "" {
			req.Filename = file.Filename
		}
		if req.Mimetype == "" {
			req.Mimetype = file.Header.Get("Content-Type")
		}
		return
	}
	if perr := c.BodyParser(&req); perr != nil {
		err = fiber.NewError(fiber.StatusBadRequest, perr.Error())
		return
	}
	if req.URL == "" {
		err = fiber.NewError(fiber.StatusBadRequest, "url or multipart file required")
		return
	}
	resp, herr := http.Get(req.URL)
	if herr != nil {
		err = fiber.NewError(fiber.StatusBadRequest, "fetch url: "+herr.Error())
		return
	}
	defer resp.Body.Close()
	data, err = io.ReadAll(resp.Body)
	if req.Mimetype == "" {
		req.Mimetype = resp.Header.Get("Content-Type")
	}
	return
}

func ctxInfo(exp uint32) *waE2E.ContextInfo {
	if exp == 0 {
		return nil
	}
	return &waE2E.ContextInfo{Expiration: proto.Uint32(exp)}
}

func (s *Server) sendMedia(c *fiber.Ctx, mediaType whatsmeow.MediaType, build func(req mediaReq, up whatsmeow.UploadResponse, data []byte, exp uint32) *waE2E.Message) error {
	req, data, err := s.readMediaPayload(c)
	if err != nil {
		return err
	}
	sess, err := s.resolveSession(c, req.Session)
	if err != nil {
		return err
	}
	jid, jerr := parseJID(req.ChatID)
	if jerr != nil {
		return fiber.NewError(fiber.StatusBadRequest, jerr.Error())
	}
	up, uerr := sess.Client.Upload(c.Context(), data, mediaType)
	if uerr != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "upload: "+uerr.Error())
	}
	exp := s.effectiveExpiration(req.Session, jid, req.Expiration)
	msg := build(req, up, data, exp)
	resp, serr := sess.Client.SendMessage(c.Context(), jid, msg)
	if serr != nil {
		return fiber.NewError(fiber.StatusInternalServerError, serr.Error())
	}
	return c.JSON(fiber.Map{"id": resp.ID, "timestamp": resp.Timestamp.Unix()})
}

func (s *Server) sendImage(c *fiber.Ctx) error {
	return s.sendMedia(c, whatsmeow.MediaImage, func(req mediaReq, up whatsmeow.UploadResponse, data []byte, exp uint32) *waE2E.Message {
		mime := req.Mimetype
		if mime == "" {
			mime = "image/jpeg"
		}
		return &waE2E.Message{ImageMessage: &waE2E.ImageMessage{
			Caption:       proto.String(req.Caption),
			Mimetype:      proto.String(mime),
			URL:           proto.String(up.URL),
			DirectPath:    proto.String(up.DirectPath),
			MediaKey:      up.MediaKey,
			FileEncSHA256: up.FileEncSHA256,
			FileSHA256:    up.FileSHA256,
			FileLength:    proto.Uint64(uint64(len(data))),
			ContextInfo:   ctxInfo(exp),
		}}
	})
}

func (s *Server) sendVideo(c *fiber.Ctx) error {
	return s.sendMedia(c, whatsmeow.MediaVideo, func(req mediaReq, up whatsmeow.UploadResponse, data []byte, exp uint32) *waE2E.Message {
		mime := req.Mimetype
		if mime == "" {
			mime = "video/mp4"
		}
		return &waE2E.Message{VideoMessage: &waE2E.VideoMessage{
			Caption:       proto.String(req.Caption),
			Mimetype:      proto.String(mime),
			URL:           proto.String(up.URL),
			DirectPath:    proto.String(up.DirectPath),
			MediaKey:      up.MediaKey,
			FileEncSHA256: up.FileEncSHA256,
			FileSHA256:    up.FileSHA256,
			FileLength:    proto.Uint64(uint64(len(data))),
			ContextInfo:   ctxInfo(exp),
		}}
	})
}

func (s *Server) sendVoice(c *fiber.Ctx) error {
	return s.sendMedia(c, whatsmeow.MediaAudio, func(req mediaReq, up whatsmeow.UploadResponse, data []byte, exp uint32) *waE2E.Message {
		mime := req.Mimetype
		if mime == "" {
			mime = "audio/ogg; codecs=opus"
		}
		return &waE2E.Message{AudioMessage: &waE2E.AudioMessage{
			Mimetype:      proto.String(mime),
			URL:           proto.String(up.URL),
			DirectPath:    proto.String(up.DirectPath),
			MediaKey:      up.MediaKey,
			FileEncSHA256: up.FileEncSHA256,
			FileSHA256:    up.FileSHA256,
			FileLength:    proto.Uint64(uint64(len(data))),
			PTT:           proto.Bool(true),
			ContextInfo:   ctxInfo(exp),
		}}
	})
}

func (s *Server) sendFile(c *fiber.Ctx) error {
	return s.sendMedia(c, whatsmeow.MediaDocument, func(req mediaReq, up whatsmeow.UploadResponse, data []byte, exp uint32) *waE2E.Message {
		mime := req.Mimetype
		if mime == "" {
			mime = "application/octet-stream"
		}
		name := req.Filename
		if name == "" {
			name = "file"
		}
		return &waE2E.Message{DocumentMessage: &waE2E.DocumentMessage{
			Caption:       proto.String(req.Caption),
			Mimetype:      proto.String(mime),
			FileName:      proto.String(name),
			URL:           proto.String(up.URL),
			DirectPath:    proto.String(up.DirectPath),
			MediaKey:      up.MediaKey,
			FileEncSHA256: up.FileEncSHA256,
			FileSHA256:    up.FileSHA256,
			FileLength:    proto.Uint64(uint64(len(data))),
			ContextInfo:   ctxInfo(exp),
		}}
	})
}

type sendLocationReq struct {
	sendBase
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Name      string  `json:"name"`
	Address   string  `json:"address"`
}

func (s *Server) sendLocation(c *fiber.Ctx) error {
	var req sendLocationReq
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "bad json")
	}
	sess, err := s.resolveSession(c, req.Session)
	if err != nil {
		return err
	}
	jid, jerr := parseJID(req.ChatID)
	if jerr != nil {
		return fiber.NewError(fiber.StatusBadRequest, jerr.Error())
	}
	exp := s.effectiveExpiration(req.Session, jid, req.Expiration)
	msg := &waE2E.Message{LocationMessage: &waE2E.LocationMessage{
		DegreesLatitude:  proto.Float64(req.Latitude),
		DegreesLongitude: proto.Float64(req.Longitude),
		Name:             proto.String(req.Name),
		Address:          proto.String(req.Address),
		ContextInfo:      ctxInfo(exp),
	}}
	resp, serr := sess.Client.SendMessage(c.Context(), jid, msg)
	if serr != nil {
		return fiber.NewError(fiber.StatusInternalServerError, serr.Error())
	}
	return c.JSON(fiber.Map{"id": resp.ID, "timestamp": resp.Timestamp.Unix()})
}

type sendContactReq struct {
	sendBase
	DisplayName string `json:"display_name"`
	VCard       string `json:"vcard"`
}

func (s *Server) sendContact(c *fiber.Ctx) error {
	var req sendContactReq
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "bad json")
	}
	if req.VCard == "" {
		return fiber.NewError(fiber.StatusBadRequest, "vcard required")
	}
	sess, err := s.resolveSession(c, req.Session)
	if err != nil {
		return err
	}
	jid, jerr := parseJID(req.ChatID)
	if jerr != nil {
		return fiber.NewError(fiber.StatusBadRequest, jerr.Error())
	}
	exp := s.effectiveExpiration(req.Session, jid, req.Expiration)
	msg := &waE2E.Message{ContactMessage: &waE2E.ContactMessage{
		DisplayName: proto.String(req.DisplayName),
		Vcard:       proto.String(req.VCard),
		ContextInfo: ctxInfo(exp),
	}}
	resp, serr := sess.Client.SendMessage(c.Context(), jid, msg)
	if serr != nil {
		return fiber.NewError(fiber.StatusInternalServerError, serr.Error())
	}
	return c.JSON(fiber.Map{"id": resp.ID, "timestamp": resp.Timestamp.Unix()})
}

type seenReq struct {
	sendBase
	MessageIDs []string `json:"message_ids"`
	Sender     string   `json:"sender"`
}

func (s *Server) sendSeen(c *fiber.Ctx) error {
	var req seenReq
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "bad json")
	}
	sess, err := s.resolveSession(c, req.Session)
	if err != nil {
		return err
	}
	chat, jerr := parseJID(req.ChatID)
	if jerr != nil {
		return fiber.NewError(fiber.StatusBadRequest, jerr.Error())
	}
	var sender types.JID
	if req.Sender != "" {
		sender, _ = parseJID(req.Sender)
	}
	ids := make([]types.MessageID, 0, len(req.MessageIDs))
	for _, id := range req.MessageIDs {
		ids = append(ids, types.MessageID(id))
	}
	if err := sess.Client.MarkRead(c.Context(), ids, c.Context().Time(), chat, sender); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(fiber.Map{"ok": true})
}

type presenceReq struct {
	sendBase
}

func (s *Server) startTyping(c *fiber.Ctx) error { return s.setTyping(c, types.ChatPresenceComposing) }
func (s *Server) stopTyping(c *fiber.Ctx) error  { return s.setTyping(c, types.ChatPresencePaused) }

func (s *Server) setTyping(c *fiber.Ctx, state types.ChatPresence) error {
	var req presenceReq
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "bad json")
	}
	sess, err := s.resolveSession(c, req.Session)
	if err != nil {
		return err
	}
	jid, jerr := parseJID(req.ChatID)
	if jerr != nil {
		return fiber.NewError(fiber.StatusBadRequest, jerr.Error())
	}
	if err := sess.Client.SendChatPresence(c.Context(), jid, state, types.ChatPresenceMediaText); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(fiber.Map{"ok": true})
}
