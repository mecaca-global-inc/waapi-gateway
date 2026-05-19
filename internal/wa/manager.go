package wa

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/mecaca/waapi-gateway/internal/store"
	"github.com/mecaca/waapi-gateway/internal/webhook"
	"github.com/rs/zerolog/log"
	wastore "go.mau.fi/whatsmeow/store"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
)

var ErrSessionNotFound = errors.New("session not found")
var ErrSessionExists = errors.New("session already exists")

type Manager struct {
	container  *sqlstore.Container
	repo       *store.Repo
	dispatcher *webhook.Dispatcher

	mu       sync.RWMutex
	sessions map[string]*Session
	// jidToName maps WhatsApp JID -> session name (for matching devices to sessions).
	jidToName map[string]string
	// listeners for live event stream (dashboard WS).
	listeners []chan EventNotice
	// ephemeral disappearing-messages cache (per-session).
	expCache map[string]*ChatExpirationCache
}

type EventNotice struct {
	Session string `json:"session"`
	Kind    string `json:"kind"`
	Payload any    `json:"payload,omitempty"`
}

func NewManager(ctx context.Context, dialect, uri string, repo *store.Repo, disp *webhook.Dispatcher) (*Manager, error) {
	logger := waLog.Stdout("WAStore", "WARN", true)
	container, err := sqlstore.New(ctx, dialect, uri, logger)
	if err != nil {
		return nil, fmt.Errorf("sqlstore.New: %w", err)
	}
	return &Manager{
		container:  container,
		repo:       repo,
		dispatcher: disp,
		sessions:   make(map[string]*Session),
		jidToName:  make(map[string]string),
		expCache:   make(map[string]*ChatExpirationCache),
	}, nil
}

// ChatExpiration returns the disappearing-messages seconds last observed for a
// chat in the given session (0 if not ephemeral / unknown).
func (m *Manager) ChatExpiration(session string, chat types.JID) uint32 {
	m.mu.RLock()
	c := m.expCache[session]
	m.mu.RUnlock()
	if c == nil {
		return 0
	}
	return c.Get(chat)
}

func (m *Manager) recordExpiration(session string, chat types.JID, seconds uint32) {
	m.mu.Lock()
	c, ok := m.expCache[session]
	if !ok {
		c = NewChatExpirationCache()
		m.expCache[session] = c
	}
	m.mu.Unlock()
	c.Set(chat, seconds)
}

// LoadExisting restores sessions stored in app DB on startup.
func (m *Manager) LoadExisting(ctx context.Context) error {
	rows, err := m.repo.ListSessions(ctx)
	if err != nil {
		return err
	}
	for _, s := range rows {
		if _, err := m.Create(ctx, s.Name); err != nil {
			log.Error().Err(err).Str("session", s.Name).Msg("failed to load session")
		}
	}
	return nil
}

func (m *Manager) Create(ctx context.Context, name string) (*Session, error) {
	m.mu.Lock()
	if _, ok := m.sessions[name]; ok {
		m.mu.Unlock()
		return nil, ErrSessionExists
	}
	m.mu.Unlock()

	if _, err := m.repo.UpsertSession(ctx, name); err != nil {
		return nil, err
	}

	device, err := m.findOrCreateDevice(ctx, name)
	if err != nil {
		return nil, err
	}

	s := newSession(name, device, m)
	m.mu.Lock()
	m.sessions[name] = s
	if device.ID != nil {
		m.jidToName[device.ID.String()] = name
	}
	m.mu.Unlock()
	return s, nil
}

// findOrCreateDevice associates session name with a whatsmeow device row.
// Strategy: store the session name in app_sessions.jid once paired; on next boot,
// match by JID. New sessions get a fresh empty device.
func (m *Manager) findOrCreateDevice(ctx context.Context, name string) (*wastore.Device, error) {
	row, err := m.repo.GetSession(ctx, name)
	if err == nil && row.JID != "" {
		jid, parseErr := types.ParseJID(row.JID)
		if parseErr == nil {
			d, derr := m.container.GetDevice(ctx, jid)
			if derr == nil && d != nil {
				return d, nil
			}
		}
	}
	return m.container.NewDevice(), nil
}

func (m *Manager) Get(name string) (*Session, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	s, ok := m.sessions[name]
	if !ok {
		return nil, ErrSessionNotFound
	}
	return s, nil
}

func (m *Manager) List() []*Session {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]*Session, 0, len(m.sessions))
	for _, s := range m.sessions {
		out = append(out, s)
	}
	return out
}

func (m *Manager) Delete(ctx context.Context, name string) error {
	m.mu.Lock()
	s, ok := m.sessions[name]
	delete(m.sessions, name)
	m.mu.Unlock()
	if ok {
		if s.Client.IsLoggedIn() {
			_ = s.Logout(ctx)
		} else {
			s.Stop()
		}
	}
	return m.repo.DeleteSession(ctx, name)
}

func (m *Manager) Subscribe() <-chan EventNotice {
	ch := make(chan EventNotice, 64)
	m.mu.Lock()
	m.listeners = append(m.listeners, ch)
	m.mu.Unlock()
	return ch
}

func (m *Manager) Unsubscribe(ch <-chan EventNotice) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i, l := range m.listeners {
		if l == ch {
			m.listeners = append(m.listeners[:i], m.listeners[i+1:]...)
			close(l)
			return
		}
	}
}

func (m *Manager) broadcast(n EventNotice) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, ch := range m.listeners {
		select {
		case ch <- n:
		default:
		}
	}
}

func (m *Manager) notifyStatus(name string, st Status, err error) {
	statusStr := string(st)
	go func() {
		ctx := context.Background()
		jid := ""
		if s, ok := m.sessions[name]; ok && s.Client.Store.ID != nil {
			jid = s.Client.Store.ID.String()
			m.mu.Lock()
			m.jidToName[jid] = name
			m.mu.Unlock()
		}
		if uerr := m.repo.UpdateSessionStatus(ctx, name, statusStr, jid); uerr != nil {
			log.Error().Err(uerr).Msg("update status failed")
		}
	}()
	payload := map[string]any{"status": statusStr}
	if err != nil {
		payload["error"] = err.Error()
	}
	m.broadcast(EventNotice{Session: name, Kind: "status", Payload: payload})
	if m.dispatcher != nil {
		m.dispatcher.Emit(webhook.Envelope{
			Event:   webhook.EventSessionStatus,
			Session: name,
			Payload: payload,
		})
	}
}

func (m *Manager) notifyQR(name, code string) {
	m.broadcast(EventNotice{Session: name, Kind: "qr", Payload: map[string]string{"code": code}})
	if m.dispatcher != nil && code != "" {
		m.dispatcher.Emit(webhook.Envelope{
			Event:   webhook.EventStateQR,
			Session: name,
			Payload: map[string]string{"code": code},
		})
	}
}

func (m *Manager) dispatchEvent(name string, evt any) {
	if m.dispatcher == nil {
		return
	}
	switch e := evt.(type) {
	case *events.Message:
		if exp := ExtractExpiration(e.Message); exp > 0 {
			m.recordExpiration(name, e.Info.Chat, exp)
		}
		m.dispatcher.Emit(webhook.Envelope{
			Event:   webhook.EventMessage,
			Session: name,
			Payload: MessagePayload(e),
		})
	case *events.Receipt:
		m.dispatcher.Emit(webhook.Envelope{
			Event:   webhook.EventMessageAck,
			Session: name,
			Payload: ReceiptPayload(e),
		})
	case *events.LoggedOut:
		m.dispatcher.Emit(webhook.Envelope{
			Event:   webhook.EventStateLoggedOut,
			Session: name,
			Payload: map[string]any{"reason": int(e.Reason), "on_connect": e.OnConnect},
		})
	case *events.PairSuccess:
		m.dispatcher.Emit(webhook.Envelope{
			Event:   webhook.EventStatePair,
			Session: name,
			Payload: map[string]any{"jid": e.ID.String()},
		})
	}
}

// Close releases all sessions and store.
func (m *Manager) Close() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, s := range m.sessions {
		s.Stop()
	}
	for _, ch := range m.listeners {
		close(ch)
	}
	m.listeners = nil
}
