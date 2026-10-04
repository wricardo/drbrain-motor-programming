# Dr Brain - Motor Programming (`drbrain-motor-programming`) — agent guide

Brand: **Dr Brain** is the family of reimplemented brain games and **Motor Programming** is this game. Every game lives in its own repo; this repo, its Go module path (`github.com/wricardo/drbrain-motor-programming`) and the server binary share the name `drbrain-motor-programming`. `localStorage` keys use that prefix too (the home page still reads the old `drbrain3.displayName` key once as a fallback). User-facing names live in `frontend/src/lib/brand.ts`, `llms.txt.tmpl` and the README.

Robot programming puzzle server + SvelteKit UI. Architecture copied from `../tesla-road-trip-game`; rules from `../drbrain2` (read-only references, never modify them). Read `README.md` for rules, map format and semantics.

## Package map (dependency direction →)

```
game/engine      pure rules, no deps/IO: MapConfig/NewMap, Program, VMState (Settle/Step), Simulate, Run
game/service     Session type, GameService facade + impl + Runner (ticker goroutine per playing session);
                 interfaces MapStore, SessionStore, Broadcaster; coded errors
game/config      MapStore impl: maps/*.json, RWMutex, atomic writes
game/session     SessionStore impl: in-memory Manager + FilePersistence (sessions/<id>.json)
transport/websocket  Hub: SubscribeSession fanout, latest-wins (no raw WS clients)
validate         map checks, reachability, reference-solution check (+ cmd/validate)
graph            gqlgen schema/resolvers, converters, subscription forwarder, admin gate
api              HTTP mux, /graphql, /llms.txt, SPA serving (frontend → static/)
main.go          flags, .env, wiring, graceful shutdown
frontend/        SvelteKit (Svelte 5) + Tailwind v4, adapter-static → static/ (committed)
maps/ solutions/ map-schema.json   data; solutions are never exposed by the API
```

`service` defines `Session`; `session` and `websocket` import `service`, never the reverse (the service talks to them through interfaces).

## Invariants — do not break

- **Engine**: after `NewVMState` and after any program change call `vm.Settle(&program)`. `Step` mutates in place; use `Clone()` for anything leaving the lock. `Run`/internal `step(…, nil)` allocate no events.
- **Locking**: live `*service.Session` is guarded by `Lock/Unlock`. Never broadcast or do I/O while holding it. Hub/resolvers only see `Clone()` snapshots. Don't hold the lock when calling `SessionStore.Persist`.
- **Runner lifecycle**: `Playing` ⇔ a runner is registered. Stop paths (Pause, Reset, Delete, terminal tick, Shutdown) all go through one `detach`; never wait on a runner while holding the session lock.
- **Delivery**: every broadcast carries a full snapshot + monotonically increasing `Seq`; hub drops the *oldest* queued update on a full buffer so terminal states are never lost. Subscription resolver emits the current snapshot first.
- **Map snapshot**: sessions embed their own copy of the map; map edits/deletes must never touch existing sessions.
- **Security**: map ids match `^[a-z0-9_-]{1,64}$` (they are filenames); session ids are validated before touching disk; admin mutations need `X-Admin-Key` == `ADMIN_API_KEY` (constant-time) unless `ALLOW_UNAUTHENTICATED_ADMIN=true`; `max_steps ≤ 10000`, `max_call_depth ≤ 64` bound `simulate`. Never add solutions to the GraphQL schema.
- **Recursion rule** is intentionally stricter than drbrain2 (see README "Engine semantics").
- A zero-sub map (`sub_tape_lengths: []`) is valid and must survive JSON round trips (no `omitempty` on that field).

## Workflows

- Edit `graph/schema.graphqls` → `make generate` → update `graph/convert.go` / resolvers → mirror in `frontend/src/lib/{queries,types}.ts`.
- New map: see README "Adding a map"; `make validate` must pass.
- Engine change: update golden step counts in `game/engine/engine_test.go` and `validate/validate_test.go` (7, 17, 19, 46, 23, 30, 44, 80, 32, 48, 83 for the shipped solutions; see the golden table in `validate/validate_test.go`) deliberately, never to make a failure disappear.
- UI change: `cd frontend && npm run check && npm test && npm run build:static` (or `make build-frontend`), commit `static/`.
- Don't run `go mod tidy` casually — `gqlgen generate` rewrites go.mod; re-`go get` any dropped deps.

## Verification checklist

`make verify` (or `go build ./... && go vet ./... && go test -race ./... && make validate`), then run `go run . -port 9191` and `scripts/smoke.sh 9191`; for UI changes load `/play/<id>` and `/watch/<id>` in a browser.
