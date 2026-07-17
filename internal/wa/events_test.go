package wa

import (
	"testing"

	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

func TestMessagePayloadLIDSender(t *testing.T) {
	lid := types.NewJID("112537404182586", types.HiddenUserServer)
	pn := types.NewJID("6281234567890", types.DefaultUserServer)

	e := &events.Message{}
	e.Info.Chat = lid
	e.Info.Sender = lid
	e.Info.SenderAlt = pn
	e.Info.AddressingMode = types.AddressingModeLID

	p := MessagePayload(e)
	if got := p["addressing_mode"]; got != "lid" {
		t.Fatalf("addressing_mode = %v, want lid", got)
	}
	if got := p["sender_alt"]; got != pn.String() {
		t.Fatalf("sender_alt = %v, want %s", got, pn.String())
	}
}

func TestMessagePayloadPNOnlyOmitsSenderAlt(t *testing.T) {
	pn := types.NewJID("6281234567890", types.DefaultUserServer)

	e := &events.Message{}
	e.Info.Chat = pn
	e.Info.Sender = pn
	// no SenderAlt, no AddressingMode

	p := MessagePayload(e)
	if _, ok := p["sender_alt"]; ok {
		t.Fatalf("sender_alt present for PN-only message, want absent")
	}
	if got := p["addressing_mode"]; got != "" {
		t.Fatalf("addressing_mode = %v, want empty", got)
	}
}
