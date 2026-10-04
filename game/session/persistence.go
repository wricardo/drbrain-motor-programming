package session

import (
	"errors"
	"regexp"

	"github.com/wricardo/drbrain-motor-programming/game/service"
)

// Errors returned by persistence implementations.
var (
	// ErrNotFound means no persisted session exists for the id.
	ErrNotFound = errors.New("session not found")
	// ErrInvalidID means the id is not a legal session id (and therefore not a
	// safe file name).
	ErrInvalidID = errors.New("invalid session id")
	// ErrExists means a live session with the same id is already registered.
	ErrExists = errors.New("session already exists")
)

var idRe = regexp.MustCompile(`^[0-9a-f]{16}$`)

// ValidID reports whether id is a legal session id: 16 lower-case hex digits
// (8 random bytes). The restriction doubles as path-traversal protection for
// file persistence.
func ValidID(id string) bool { return idRe.MatchString(id) }

// SessionPersistence stores sessions durably. Implementations receive
// immutable snapshots (clones) from the Manager and never see live sessions.
type SessionPersistence interface {
	// Save writes the snapshot, replacing any previous version atomically.
	Save(s *service.Session) error
	// Load reads one session. It returns ErrNotFound if absent.
	Load(id string) (*service.Session, error)
	// Delete removes one session. It returns ErrNotFound if absent.
	Delete(id string) error
	// ListAll returns the ids of all persisted sessions.
	ListAll() ([]string, error)
}
