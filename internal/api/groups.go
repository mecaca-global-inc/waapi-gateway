package api

import (
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"
)

// parseGroupJID: accepts "1234567890-1700000000@g.us" OR raw "1234567890-1700000000".
func parseGroupJID(in string) (types.JID, error) {
	in = strings.TrimSpace(in)
	if in == "" {
		return types.JID{}, fiber.NewError(fiber.StatusBadRequest, "group id required")
	}
	if !strings.Contains(in, "@") {
		in = in + "@g.us"
	}
	jid, err := types.ParseJID(in)
	if err != nil {
		return jid, fiber.NewError(fiber.StatusBadRequest, "invalid group jid")
	}
	if jid.Server != types.GroupServer {
		return jid, fiber.NewError(fiber.StatusBadRequest, "not a group jid (expected @g.us)")
	}
	return jid, nil
}

func parseParticipants(raw []string) ([]types.JID, error) {
	out := make([]types.JID, 0, len(raw))
	for _, p := range raw {
		j, err := parseJID(p)
		if err != nil {
			return nil, fiber.NewError(fiber.StatusBadRequest, "bad participant "+p)
		}
		out = append(out, j)
	}
	return out, nil
}

func groupInfoDTO(g *types.GroupInfo) fiber.Map {
	members := make([]fiber.Map, 0, len(g.Participants))
	for _, p := range g.Participants {
		members = append(members, fiber.Map{
			"jid":          p.JID.String(),
			"is_admin":     p.IsAdmin,
			"is_superadmin": p.IsSuperAdmin,
		})
	}
	return fiber.Map{
		"jid":         g.JID.String(),
		"name":        g.GroupName.Name,
		"topic":       g.GroupTopic.Topic,
		"locked":      g.GroupLocked.IsLocked,
		"announce":    g.GroupAnnounce.IsAnnounce,
		"owner":       g.OwnerJID.String(),
		"created_at":  g.GroupCreated.Unix(),
		"participants": members,
	}
}

// ---------- Create ----------

type createGroupReq struct {
	Name         string   `json:"name"`
	Participants []string `json:"participants"`
}

func (s *Server) createGroup(c *fiber.Ctx) error {
	sess, err := s.resolveSession(c, c.Params("session"))
	if err != nil {
		return err
	}
	var req createGroupReq
	if err := c.BodyParser(&req); err != nil || req.Name == "" {
		return fiber.NewError(fiber.StatusBadRequest, "name required")
	}
	parts, err := parseParticipants(req.Participants)
	if err != nil {
		return err
	}
	g, gerr := sess.Client.CreateGroup(c.Context(), whatsmeow.ReqCreateGroup{
		Name:         req.Name,
		Participants: parts,
	})
	if gerr != nil {
		return fiber.NewError(fiber.StatusInternalServerError, gerr.Error())
	}
	return c.Status(fiber.StatusCreated).JSON(groupInfoDTO(g))
}

// ---------- Group info ----------

func (s *Server) getGroupInfo(c *fiber.Ctx) error {
	sess, err := s.resolveSession(c, c.Params("session"))
	if err != nil {
		return err
	}
	jid, jerr := parseGroupJID(c.Params("gid"))
	if jerr != nil {
		return jerr
	}
	g, gerr := sess.Client.GetGroupInfo(c.Context(), jid)
	if gerr != nil {
		return fiber.NewError(fiber.StatusInternalServerError, gerr.Error())
	}
	return c.JSON(groupInfoDTO(g))
}

// ---------- Leave ----------

func (s *Server) leaveGroup(c *fiber.Ctx) error {
	sess, err := s.resolveSession(c, c.Params("session"))
	if err != nil {
		return err
	}
	jid, jerr := parseGroupJID(c.Params("gid"))
	if jerr != nil {
		return jerr
	}
	if err := sess.Client.LeaveGroup(c.Context(), jid); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(fiber.Map{"ok": true})
}

// ---------- Participants add/remove/promote/demote ----------

type participantsReq struct {
	Participants []string `json:"participants"`
}

func (s *Server) updateParticipants(c *fiber.Ctx, action whatsmeow.ParticipantChange) error {
	sess, err := s.resolveSession(c, c.Params("session"))
	if err != nil {
		return err
	}
	jid, jerr := parseGroupJID(c.Params("gid"))
	if jerr != nil {
		return jerr
	}
	var req participantsReq
	if err := c.BodyParser(&req); err != nil || len(req.Participants) == 0 {
		return fiber.NewError(fiber.StatusBadRequest, "participants required")
	}
	parts, err := parseParticipants(req.Participants)
	if err != nil {
		return err
	}
	res, perr := sess.Client.UpdateGroupParticipants(c.Context(), jid, parts, action)
	if perr != nil {
		return fiber.NewError(fiber.StatusInternalServerError, perr.Error())
	}
	out := make([]fiber.Map, 0, len(res))
	for _, p := range res {
		out = append(out, fiber.Map{"jid": p.JID.String(), "error": p.Error})
	}
	return c.JSON(out)
}

func (s *Server) addParticipants(c *fiber.Ctx) error {
	return s.updateParticipants(c, whatsmeow.ParticipantChangeAdd)
}
func (s *Server) removeParticipants(c *fiber.Ctx) error {
	return s.updateParticipants(c, whatsmeow.ParticipantChangeRemove)
}
func (s *Server) promoteParticipants(c *fiber.Ctx) error {
	return s.updateParticipants(c, whatsmeow.ParticipantChangePromote)
}
func (s *Server) demoteParticipants(c *fiber.Ctx) error {
	return s.updateParticipants(c, whatsmeow.ParticipantChangeDemote)
}

// ---------- Name / Topic / Locked / Announce ----------

type nameReq struct {
	Name string `json:"name"`
}

func (s *Server) setGroupName(c *fiber.Ctx) error {
	sess, err := s.resolveSession(c, c.Params("session"))
	if err != nil {
		return err
	}
	jid, jerr := parseGroupJID(c.Params("gid"))
	if jerr != nil {
		return jerr
	}
	var req nameReq
	if err := c.BodyParser(&req); err != nil || req.Name == "" {
		return fiber.NewError(fiber.StatusBadRequest, "name required")
	}
	if err := sess.Client.SetGroupName(c.Context(), jid, req.Name); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(fiber.Map{"ok": true})
}

type topicReq struct {
	Topic string `json:"topic"`
}

func (s *Server) setGroupTopic(c *fiber.Ctx) error {
	sess, err := s.resolveSession(c, c.Params("session"))
	if err != nil {
		return err
	}
	jid, jerr := parseGroupJID(c.Params("gid"))
	if jerr != nil {
		return jerr
	}
	var req topicReq
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "bad json")
	}
	if err := sess.Client.SetGroupTopic(c.Context(), jid, "", "", req.Topic); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(fiber.Map{"ok": true})
}

type boolReq struct {
	Value bool `json:"value"`
}

func (s *Server) setGroupLocked(c *fiber.Ctx) error {
	sess, err := s.resolveSession(c, c.Params("session"))
	if err != nil {
		return err
	}
	jid, jerr := parseGroupJID(c.Params("gid"))
	if jerr != nil {
		return jerr
	}
	var req boolReq
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "bad json")
	}
	if err := sess.Client.SetGroupLocked(c.Context(), jid, req.Value); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(fiber.Map{"ok": true})
}

func (s *Server) setGroupAnnounce(c *fiber.Ctx) error {
	sess, err := s.resolveSession(c, c.Params("session"))
	if err != nil {
		return err
	}
	jid, jerr := parseGroupJID(c.Params("gid"))
	if jerr != nil {
		return jerr
	}
	var req boolReq
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "bad json")
	}
	if err := sess.Client.SetGroupAnnounce(c.Context(), jid, req.Value); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(fiber.Map{"ok": true})
}

// ---------- Photo ----------

func (s *Server) setGroupPhoto(c *fiber.Ctx) error {
	sess, err := s.resolveSession(c, c.Params("session"))
	if err != nil {
		return err
	}
	jid, jerr := parseGroupJID(c.Params("gid"))
	if jerr != nil {
		return jerr
	}
	var data []byte

	ct := c.Get("Content-Type")
	if strings.HasPrefix(ct, "multipart/form-data") {
		file, herr := c.FormFile("file")
		if herr != nil {
			return fiber.NewError(fiber.StatusBadRequest, "file required")
		}
		fh, oerr := file.Open()
		if oerr != nil {
			return fiber.NewError(fiber.StatusInternalServerError, oerr.Error())
		}
		defer fh.Close()
		data, _ = io.ReadAll(fh)
	} else {
		var body struct {
			URL string `json:"url"`
		}
		if err := c.BodyParser(&body); err != nil || body.URL == "" {
			return fiber.NewError(fiber.StatusBadRequest, "url or multipart file required")
		}
		resp, herr := http.Get(body.URL)
		if herr != nil {
			return fiber.NewError(fiber.StatusBadRequest, herr.Error())
		}
		defer resp.Body.Close()
		data, _ = io.ReadAll(resp.Body)
	}
	if len(data) == 0 {
		return fiber.NewError(fiber.StatusBadRequest, "empty photo")
	}
	pictureID, err := sess.Client.SetGroupPhoto(c.Context(), jid, data)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(fiber.Map{"picture_id": pictureID})
}

// ---------- Invite link ----------

func (s *Server) getInviteLink(c *fiber.Ctx) error {
	sess, err := s.resolveSession(c, c.Params("session"))
	if err != nil {
		return err
	}
	jid, jerr := parseGroupJID(c.Params("gid"))
	if jerr != nil {
		return jerr
	}
	link, err := sess.Client.GetGroupInviteLink(c.Context(), jid, false)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(fiber.Map{"link": "https://chat.whatsapp.com/" + link, "code": link})
}

func (s *Server) revokeInviteLink(c *fiber.Ctx) error {
	sess, err := s.resolveSession(c, c.Params("session"))
	if err != nil {
		return err
	}
	jid, jerr := parseGroupJID(c.Params("gid"))
	if jerr != nil {
		return jerr
	}
	link, err := sess.Client.GetGroupInviteLink(c.Context(), jid, true)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(fiber.Map{"link": "https://chat.whatsapp.com/" + link, "code": link})
}

// ---------- Join by link ----------

type joinReq struct {
	Link string `json:"link"`
}

// extractInviteCode pulls the invite code out of either a full chat.whatsapp.com URL or a bare code.
func extractInviteCode(in string) string {
	in = strings.TrimSpace(in)
	if i := strings.Index(in, "chat.whatsapp.com/"); i >= 0 {
		in = in[i+len("chat.whatsapp.com/"):]
	}
	if i := strings.Index(in, "?"); i >= 0 {
		in = in[:i]
	}
	return strings.Trim(in, "/")
}

func (s *Server) joinGroup(c *fiber.Ctx) error {
	sess, err := s.resolveSession(c, c.Params("session"))
	if err != nil {
		return err
	}
	var req joinReq
	if err := c.BodyParser(&req); err != nil || req.Link == "" {
		return fiber.NewError(fiber.StatusBadRequest, "link required")
	}
	code := extractInviteCode(req.Link)
	jid, jerr := sess.Client.JoinGroupWithLink(c.Context(), code)
	if jerr != nil {
		return fiber.NewError(fiber.StatusInternalServerError, jerr.Error())
	}
	return c.JSON(fiber.Map{"jid": jid.String()})
}

// ---------- Disappearing timer ----------

type disappearingReq struct {
	Seconds int64 `json:"seconds"`
}

func (s *Server) setGroupDisappearing(c *fiber.Ctx) error {
	sess, err := s.resolveSession(c, c.Params("session"))
	if err != nil {
		return err
	}
	jid, jerr := parseGroupJID(c.Params("gid"))
	if jerr != nil {
		return jerr
	}
	var req disappearingReq
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "bad json")
	}
	dur := time.Duration(req.Seconds) * time.Second
	if err := sess.Client.SetDisappearingTimer(c.Context(), jid, dur, time.Now()); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(fiber.Map{"ok": true})
}
