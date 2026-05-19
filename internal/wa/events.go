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
	return map[string]any{
		"id":         e.Info.ID,
		"chat":       e.Info.Chat.String(),
		"sender":     e.Info.Sender.String(),
		"from_me":    e.Info.IsFromMe,
		"timestamp":  e.Info.Timestamp.Unix(),
		"push_name":  e.Info.PushName,
		"body":       body,
		"has_media":  hasMedia(e),
	}
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
	return map[string]any{
		"chat":        e.Chat.String(),
		"sender":      e.Sender.String(),
		"type":        string(e.Type),
		"message_ids": e.MessageIDs,
		"timestamp":   e.Timestamp.Unix(),
	}
}
