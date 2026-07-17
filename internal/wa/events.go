package wa

import (
	"go.mau.fi/whatsmeow/types/events"
)

func MessagePayload(e *events.Message) map[string]any {
	body := ""
	if e.Message != nil {
		if t := e.Message.GetConversation(); t != "" {
			body = t
		} else if e.Message.ExtendedTextMessage != nil {
			body = e.Message.ExtendedTextMessage.GetText()
		}
	}
	payload := map[string]any{
		"id":              e.Info.ID,
		"chat":            e.Info.Chat.String(),
		"sender":          e.Info.Sender.String(),
		"addressing_mode": string(e.Info.AddressingMode),
		"from_me":         e.Info.IsFromMe,
		"timestamp":       e.Info.Timestamp.Unix(),
		"push_name":       e.Info.PushName,
		"body":            body,
		"has_media":       hasMedia(e),
	}
	// sender_alt carries the phone-number JID when the sender is LID-addressed (and vice versa).
	if !e.Info.SenderAlt.IsEmpty() {
		payload["sender_alt"] = e.Info.SenderAlt.String()
	}
	return payload
}

func hasMedia(e *events.Message) bool {
	if e.Message == nil {
		return false
	}
	return e.Message.ImageMessage != nil ||
		e.Message.VideoMessage != nil ||
		e.Message.AudioMessage != nil ||
		e.Message.DocumentMessage != nil ||
		e.Message.StickerMessage != nil
}

func ReceiptPayload(e *events.Receipt) map[string]any {
	payload := map[string]any{
		"chat":            e.Chat.String(),
		"sender":          e.Sender.String(),
		"addressing_mode": string(e.AddressingMode),
		"type":            string(e.Type),
		"message_ids":     e.MessageIDs,
		"timestamp":       e.Timestamp.Unix(),
	}
	if !e.SenderAlt.IsEmpty() {
		payload["sender_alt"] = e.SenderAlt.String()
	}
	return payload
}
