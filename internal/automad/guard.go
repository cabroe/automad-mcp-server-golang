package automad

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"
)

// action describes a bridge operation for the write guard.
type action struct {
	name        string
	readOnly    bool
	destructive bool
}

// permit is the outcome of a guard check.
type permit struct {
	// allowed is true when the action may proceed immediately.
	allowed bool
	// pending is true when the action is destructive and awaits confirmation.
	pending bool
	// forbidden carries the rejection reason when the action is not allowed.
	forbidden string

	confirmToken string
	target       string
	action       string
	expiresAt    time.Time
}

const confirmTTL = 5 * time.Minute

// writeGuard gates destructive operations according to the configured WriteMode.
type writeGuard struct {
	mode WriteMode

	mu      sync.Mutex
	pending map[string]pendingEntry
}

type pendingEntry struct {
	action    string
	target    string
	expiresAt time.Time
}

func newWriteGuard(mode WriteMode) *writeGuard {
	return &writeGuard{mode: mode, pending: map[string]pendingEntry{}}
}

// check evaluates whether act on target may proceed. In confirm-destructive
// mode a destructive action first returns a pending permit carrying a token;
// re-calling with that token (matching action+target) allows it.
func (g *writeGuard) check(act action, target, confirmToken string) permit {
	if act.readOnly {
		return permit{allowed: true}
	}
	switch g.mode {
	case WriteUnrestricted:
		return permit{allowed: true}
	case WriteReadOnly:
		return permit{forbidden: "server is in read-only mode; set AUTOMAD_WRITE_MODE=confirm-destructive or unrestricted to enable writes"}
	}
	// confirm-destructive: non-destructive writes proceed; destructive ones
	// require a confirmation token.
	if !act.destructive {
		return permit{allowed: true}
	}
	if confirmToken != "" {
		if g.consume(confirmToken, act.name, target) {
			return permit{allowed: true}
		}
		return permit{forbidden: "invalid or expired confirm_token; request a fresh one by calling again without confirm_token"}
	}
	return g.issue(act.name, target)
}

func (g *writeGuard) issue(actName, target string) permit {
	token := randomToken()
	expires := time.Now().Add(confirmTTL)
	g.mu.Lock()
	g.pending[token] = pendingEntry{action: actName, target: target, expiresAt: expires}
	g.mu.Unlock()
	return permit{
		pending:      true,
		confirmToken: token,
		target:       target,
		action:       actName,
		expiresAt:    expires,
	}
}

func (g *writeGuard) consume(token, actName, target string) bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	entry, ok := g.pending[token]
	if !ok {
		return false
	}
	delete(g.pending, token)
	if time.Now().After(entry.expiresAt) {
		return false
	}
	return entry.action == actName && entry.target == target
}

func randomToken() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return hex.EncodeToString([]byte(time.Now().String()))
	}
	return hex.EncodeToString(b)
}
