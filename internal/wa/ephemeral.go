package wa

import (
	"sync"

	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
)

// ChatExpirationCache tracks the disappearing-messages timer (in seconds) seen
// for each chat. Updated from incoming events.Message; consulted on outgoing
// sends so replies inherit the chat's ephemeral setting automatically.
type ChatExpirationCache struct {
	mu sync.RWMutex
	m  map[string]uint32
}

func NewChatExpirationCache() *ChatExpirationCache {
	return &ChatExpirationCache{m: make(map[string]uint32)}
}

func (c *ChatExpirationCache) Get(chat types.JID) uint32 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.m[chat.String()]
}

func (c *ChatExpirationCache) Set(chat types.JID, seconds uint32) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if seconds == 0 {
		delete(c.m, chat.String())
		return
	}
	c.m[chat.String()] = seconds
}

// ExtractExpiration walks a received WhatsApp message and returns the first
// non-zero `ContextInfo.Expiration` it finds across the common message types.
// Returns 0 when the chat is not ephemeral.
func ExtractExpiration(msg *waE2E.Message) uint32 {
	if msg == nil {
		return 0
	}
	if ci := msg.GetExtendedTextMessage().GetContextInfo(); ci != nil {
		if x := ci.GetExpiration(); x > 0 {
			return x
		}
	}
	if ci := msg.GetImageMessage().GetContextInfo(); ci != nil {
		if x := ci.GetExpiration(); x > 0 {
			return x
		}
	}
	if ci := msg.GetVideoMessage().GetContextInfo(); ci != nil {
		if x := ci.GetExpiration(); x > 0 {
			return x
		}
	}
	if ci := msg.GetAudioMessage().GetContextInfo(); ci != nil {
		if x := ci.GetExpiration(); x > 0 {
			return x
		}
	}
	if ci := msg.GetDocumentMessage().GetContextInfo(); ci != nil {
		if x := ci.GetExpiration(); x > 0 {
			return x
		}
	}
	if ci := msg.GetStickerMessage().GetContextInfo(); ci != nil {
		if x := ci.GetExpiration(); x > 0 {
			return x
		}
	}
	if ci := msg.GetContactMessage().GetContextInfo(); ci != nil {
		if x := ci.GetExpiration(); x > 0 {
			return x
		}
	}
	if ci := msg.GetLocationMessage().GetContextInfo(); ci != nil {
		if x := ci.GetExpiration(); x > 0 {
			return x
		}
	}
	// Protocol message may carry an ephemeral setting change.
	if x := msg.GetProtocolMessage().GetEphemeralExpiration(); x > 0 {
		return x
	}
	return 0
}
