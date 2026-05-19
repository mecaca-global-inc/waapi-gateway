package wa

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
)

type Status string

const (
	StatusStopped  Status = "STOPPED"
	StatusStarting Status = "STARTING"
	StatusScanQR   Status = "SCAN_QR"
	StatusWorking  Status = "WORKING"
	StatusFailed   Status = "FAILED"
)

type Session struct {
	Name    string
	Client  *whatsmeow.Client
	device  *store.Device
	manager *Manager

	mu       sync.RWMutex
	status   Status
	lastErr  error
	lastQR   string
	qrCancel context.CancelFunc
}

func newSession(name string, device *store.Device, mgr *Manager) *Session {
	logger := waLog.Stdout("Client/"+name, "WARN", true)
	client := whatsmeow.NewClient(device, logger)
	s := &Session{
		Name:    name,
		Client:  client,
		device:  device,
		manager: mgr,
		status:  StatusStopped,
	}
	client.AddEventHandler(s.handleEvent)
	return s
}

func (s *Session) Status() Status {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.status
}

func (s *Session) LastQR() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.lastQR
}

func (s *Session) setStatus(st Status, err error) {
	s.mu.Lock()
	s.status = st
	s.lastErr = err
	s.mu.Unlock()
	if s.manager != nil {
		s.manager.notifyStatus(s.Name, st, err)
	}
}

func (s *Session) setQR(code string) {
	s.mu.Lock()
	s.lastQR = code
	s.mu.Unlock()
	if s.manager != nil {
		s.manager.notifyQR(s.Name, code)
	}
}

func (s *Session) Start(ctx context.Context) error {
	if s.Client.IsConnected() {
		return nil
	}
	s.setStatus(StatusStarting, nil)
	if s.Client.Store.ID == nil {
		qrCtx, cancel := context.WithCancel(context.Background())
		s.mu.Lock()
		s.qrCancel = cancel
		s.mu.Unlock()
		ch, err := s.Client.GetQRChannel(qrCtx)
		if err != nil {
			cancel()
			s.setStatus(StatusFailed, err)
			return fmt.Errorf("get qr channel: %w", err)
		}
		if err := s.Client.Connect(); err != nil {
			cancel()
			s.setStatus(StatusFailed, err)
			return fmt.Errorf("connect: %w", err)
		}
		go s.consumeQR(ch)
	} else {
		if err := s.Client.Connect(); err != nil {
			s.setStatus(StatusFailed, err)
			return fmt.Errorf("connect: %w", err)
		}
	}
	return nil
}

func (s *Session) consumeQR(ch <-chan whatsmeow.QRChannelItem) {
	for ev := range ch {
		switch ev.Event {
		case whatsmeow.QRChannelEventCode:
			s.setStatus(StatusScanQR, nil)
			s.setQR(ev.Code)
		case "success":
			s.setQR("")
		case "timeout", "err-client-outdated", "err-scanned-without-multidevice":
			s.setStatus(StatusFailed, errors.New(ev.Event))
			return
		case whatsmeow.QRChannelEventError:
			s.setStatus(StatusFailed, ev.Error)
			return
		}
	}
}

func (s *Session) Stop() {
	s.mu.Lock()
	if s.qrCancel != nil {
		s.qrCancel()
		s.qrCancel = nil
	}
	s.mu.Unlock()
	s.Client.Disconnect()
	s.setStatus(StatusStopped, nil)
}

func (s *Session) Logout(ctx context.Context) error {
	if err := s.Client.Logout(ctx); err != nil {
		return err
	}
	s.setStatus(StatusStopped, nil)
	s.setQR("")
	return nil
}

func (s *Session) PairPhone(ctx context.Context, phone string) (string, error) {
	if !s.Client.IsConnected() {
		if err := s.Start(ctx); err != nil {
			return "", err
		}
	}
	return s.Client.PairPhone(ctx, phone, true, whatsmeow.PairClientChrome, "WAAPI Gateway")
}

func (s *Session) handleEvent(evt any) {
	switch e := evt.(type) {
	case *events.Connected:
		s.setStatus(StatusWorking, nil)
		s.setQR("")
	case *events.PairSuccess:
		log.Info().Str("session", s.Name).Str("jid", e.ID.String()).Msg("pair success")
		s.setStatus(StatusWorking, nil)
	case *events.LoggedOut:
		s.setStatus(StatusStopped, nil)
	case *events.Disconnected:
		if s.Status() != StatusStopped {
			s.setStatus(StatusStarting, nil)
		}
	}
	if s.manager != nil {
		s.manager.dispatchEvent(s.Name, evt)
	}
}

// Helper to wait until the client is logged in or context cancelled.
func (s *Session) WaitLogin(ctx context.Context) error {
	t := time.NewTicker(500 * time.Millisecond)
	defer t.Stop()
	for {
		if s.Client.IsLoggedIn() {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-t.C:
		}
	}
}
