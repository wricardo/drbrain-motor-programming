// Package service is the single facade (GameService) used by every transport.
//
// It owns the session lifecycle on top of three collaborators: a MapStore
// (maps), a SessionStore (live sessions + persistence) and a Broadcaster
// (subscriber fan-out). New wires them together.
//
// Sessions embed a private snapshot of their map, so editing or deleting a map
// never invalidates stored programs or VM state. A Playing session is driven
// by one runner goroutine (see runner.go) that ticks the VM at the requested
// speed; Pause, Reset, DeleteSession and Shutdown all stop it through the same
// detach path. Every state change bumps Session.Seq and is broadcast as a full
// immutable snapshot, outside the session lock.
//
// All values returned to callers are clones; live sessions never leave the
// service.
package service
