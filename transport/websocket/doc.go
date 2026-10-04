// Package websocket holds the session fan-out Hub used by GraphQL
// subscriptions (graphql-ws is served by the graph/api layer; this package has
// no raw WebSocket clients and no lobby).
//
// The Hub implements service.Broadcaster. The game service publishes an
// immutable *service.Session snapshot after every state change and each
// subscriber of that session receives it on a buffered channel (8) with
// latest-wins semantics: when a subscriber is slow the oldest queued snapshot
// is dropped, never the newest, so the terminal WON/LOST update always
// arrives. Snapshots carry a monotonically increasing Seq; because publishers
// do not hold the session lock while broadcasting, two updates may arrive out
// of order and consumers should discard Seq <= the last Seq they applied.
package websocket
