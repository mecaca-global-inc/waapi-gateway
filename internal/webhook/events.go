package webhook

const (
	EventSessionStatus  = "session.status"
	EventMessage        = "message"
	EventMessageAck     = "message.ack"
	EventMessageReact   = "message.reaction"
	EventStateQR        = "state.qr"
	EventStatePair      = "state.pair"
	EventStateLoggedOut = "state.loggedout"
)

type Envelope struct {
	Event   string `json:"event"`
	Session string `json:"session"`
	Time    int64  `json:"timestamp"`
	Payload any    `json:"payload"`
}
