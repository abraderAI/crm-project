package email

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/mail"
	"sync"

	imap "github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"

	"github.com/abraderAI/crm-project/api/internal/channel"
)

// LiveIMAPProvider implements IMAPProvider using a real IMAP connection
// via the go-imap/v2 library. It connects with implicit TLS and authenticates
// with username/password (App Password or similar).
type LiveIMAPProvider struct {
	mu       sync.Mutex
	client   *imapclient.Client
	notifyCh chan struct{}
	logger   *slog.Logger
}

// NewLiveIMAPProvider creates a new live IMAP provider.
func NewLiveIMAPProvider(logger *slog.Logger) *LiveIMAPProvider {
	if logger == nil {
		logger = slog.Default()
	}
	return &LiveIMAPProvider{
		notifyCh: make(chan struct{}, 16),
		logger:   logger,
	}
}

// Connect establishes a TLS connection to the IMAP server and logs in.
func (p *LiveIMAPProvider) Connect(cfg channel.EmailConfig) error {
	addr := fmt.Sprintf("%s:%d", cfg.IMAPHost, cfg.IMAPPort)

	notifyCh := p.notifyCh
	options := &imapclient.Options{
		UnilateralDataHandler: &imapclient.UnilateralDataHandler{
			Mailbox: func(data *imapclient.UnilateralDataMailbox) {
				// Fired when the server reports a change in mailbox state.
				// NumMessages being set indicates new mail has arrived.
				if data.NumMessages != nil {
					select {
					case notifyCh <- struct{}{}:
					default:
					}
				}
			},
		},
	}

	c, err := imapclient.DialTLS(addr, options)
	if err != nil {
		return fmt.Errorf("dialing IMAP server %s: %w", addr, err)
	}

	if err := c.Login(cfg.Username, cfg.Password).Wait(); err != nil {
		_ = c.Close()
		return fmt.Errorf("IMAP login for %s: %w", cfg.Username, err)
	}

	p.mu.Lock()
	p.client = c
	p.mu.Unlock()

	p.logger.Info("IMAP connected", "host", cfg.IMAPHost, "user", cfg.Username)
	return nil
}

// StartIDLE selects the given mailbox, processes any existing unread messages,
// then enters an IMAP IDLE loop. The handler is called once per new message UID.
// StartIDLE blocks until the connection is lost or Close is called.
func (p *LiveIMAPProvider) StartIDLE(mailbox string, handler func(uid uint32)) error {
	c := p.getClient()
	if c == nil {
		return fmt.Errorf("not connected")
	}

	// Select the mailbox and capture the UIDNEXT baseline.
	selected, err := c.Select(mailbox, nil).Wait()
	if err != nil {
		return fmt.Errorf("selecting mailbox %q: %w", mailbox, err)
	}

	// Process any unread messages that arrived while the watcher was offline.
	// After this, lastUID is set so subsequent notifications only fetch new mail.
	var lastUID imap.UID
	if selected.UIDNext > 1 {
		var uidSet imap.UIDSet
		uidSet.AddRange(1, selected.UIDNext-1)
		unseen, err := c.UIDSearch(&imap.SearchCriteria{
			UID:     []imap.UIDSet{uidSet},
			NotFlag: []imap.Flag{imap.FlagSeen},
		}, nil).Wait()
		if err == nil {
			for _, uid := range unseen.AllUIDs() {
				handler(uint32(uid))
			}
		}
		lastUID = selected.UIDNext - 1
	}

	// Drain stale notifications before entering IDLE.
	p.drainNotify()

	// IDLE loop: re-enter IDLE after each new-mail notification.
	for {
		idleCmd, err := c.Idle()
		if err != nil {
			return fmt.Errorf("starting IDLE: %w", err)
		}

		idleDone := make(chan error, 1)
		go func() { idleDone <- idleCmd.Wait() }()

		select {
		case <-p.notifyCh:
			// Server signalled new mail — exit IDLE then search for new UIDs.
			if err := idleCmd.Close(); err != nil {
				<-idleDone
				return fmt.Errorf("closing IDLE command: %w", err)
			}
			if err := <-idleDone; err != nil {
				return err
			}
			if err := p.fetchNewUIDs(c, &lastUID, handler); err != nil {
				return err
			}

		case err := <-idleDone:
			// Server closed IDLE (e.g. 30-min keepalive timeout) — re-enter.
			if err != nil {
				return err
			}
		}

		// Drain stale notifications before re-entering IDLE.
		p.drainNotify()
	}
}

// fetchNewUIDs searches for messages with UID > lastUID and calls handler for each.
func (p *LiveIMAPProvider) fetchNewUIDs(c *imapclient.Client, lastUID *imap.UID, handler func(uid uint32)) error {
	var uidSet imap.UIDSet
	uidSet.AddRange(*lastUID+1, 0) // 0 encodes as "*" (all UIDs >= lastUID+1)

	result, err := c.UIDSearch(&imap.SearchCriteria{
		UID: []imap.UIDSet{uidSet},
	}, nil).Wait()
	if err != nil {
		return fmt.Errorf("searching for new UIDs after %d: %w", *lastUID, err)
	}

	for _, uid := range result.AllUIDs() {
		handler(uint32(uid))
		if uid > *lastUID {
			*lastUID = uid
		}
	}
	return nil
}

// drainNotify discards pending notifications from the buffer.
func (p *LiveIMAPProvider) drainNotify() {
	for {
		select {
		case <-p.notifyCh:
		default:
			return
		}
	}
}

// FetchMessage retrieves the full RFC 5322 message for the given UID.
// It must be called while the connection is NOT in IDLE state (i.e. from
// within the StartIDLE handler callback, after IDLE has been exited).
func (p *LiveIMAPProvider) FetchMessage(_ context.Context, uid uint32) (*mail.Message, error) {
	c := p.getClient()
	if c == nil {
		return nil, fmt.Errorf("not connected")
	}

	bodySection := &imap.FetchItemBodySection{}
	fetchCmd := c.Fetch(imap.UIDSetNum(imap.UID(uid)), &imap.FetchOptions{
		BodySection: []*imap.FetchItemBodySection{bodySection},
	})
	defer fetchCmd.Close() //nolint:errcheck

	msgData := fetchCmd.Next()
	if msgData == nil {
		return nil, fmt.Errorf("message UID %d not found", uid)
	}

	for {
		item := msgData.Next()
		if item == nil {
			break
		}
		bsData, ok := item.(imapclient.FetchItemDataBodySection)
		if !ok {
			continue
		}
		raw, err := io.ReadAll(bsData.Literal)
		if err != nil {
			return nil, fmt.Errorf("reading body for UID %d: %w", uid, err)
		}
		return mail.ReadMessage(bytes.NewReader(raw))
	}

	return nil, fmt.Errorf("body section not found in response for UID %d", uid)
}

// Close terminates the IMAP connection.
func (p *LiveIMAPProvider) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.client != nil {
		err := p.client.Close()
		p.client = nil
		return err
	}
	return nil
}

func (p *LiveIMAPProvider) getClient() *imapclient.Client {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.client
}
