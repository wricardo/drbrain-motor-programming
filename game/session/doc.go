// Package session holds the live-session registry (Manager) and its optional
// durable storage (FilePersistence: one atomic JSON file per session).
//
// service defines the Session type; this package manages and persists
// sessions and therefore imports service, never the other way round.
//
// Sessions are identified by 16 lower-case hex characters (8 random bytes);
// any other id is rejected before it can reach the file system.
//
// Manager is goroutine-safe. Persist takes the session lock and writes a
// clone, so callers must not hold the lock. Sessions loaded at startup always
// have Playing=false; a restored VM resumes only when the user presses
// Run/Step.
package session
