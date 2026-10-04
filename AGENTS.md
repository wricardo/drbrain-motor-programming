# Repository Guidelines

## Project Overview
drbrain-motor-programming (product name: **Dr Brain - Motor Programming**; each Dr Brain game is its own repo): robot-programming puzzle server (Go 1.25) + SvelteKit (Svelte 5) UI. A player programs a robot on a grid (main tape + 0–3 sub tapes) to collect treats. Exposes GraphQL (queries, mutations, subscriptions over graphql-ws), `/llms.txt` (agent-facing rules, rendered from `llms.txt.tmpl`), and serves the built SPA. Architecture mirrors `../tesla-road-trip-game`, rules come from `../drbrain2` (read-only references; never modify). Root `CLAUDE.md` holds the invariants; `README.md` documents rules, map format and semantics; brand constants live in `frontend/src/lib/brand.ts`.

## Architecture & Data Flow
```
HTTP /graphql (api/server.go) → gqlgen resolver (graph/) → service.GameService (game/service)
   → engine (pure rules) ; Run → per-session ticker Runner → advance → Seq++ → Clone → unlock → Persist → Broadcast
   → websocket.Hub (8-slot per-subscriber queue, drop-oldest) → graph/subscription_forward.go → graphql-ws → frontend store
```
- **Composition root**: `main.go:run` wires `config.Manager` (MapStore) + `session.Manager`/`FilePersistence` (SessionStore) + `websocket.Hub` (Broadcaster) into `service.New(...)`, then `api.NewServer`. Shutdown: stop HTTP, then service runners, persisting sessions.
- **DI via interfaces** defined in `game/service` (`MapStore`, `SessionStore`, `Broadcaster`); implementations assert them at compile time. `service` never imports `session`/`websocket`.
- **Engine** (`game/engine`): no IO/deps. `NewMap`, `ValidateProgram`, `NewVMState`, `Settle`, `Step`, `Simulate`, `Run`. Always `vm.Settle(&program)` after `NewVMState` and any program change. `Step` mutates in place; `Clone()` before leaving a lock.
- **Locking**: live `*service.Session` guarded by `Lock/Unlock`; lock order session → runner mutex. Never broadcast, do I/O, `Persist`, or wait on a runner while holding the session lock. Hub/resolvers only see clones.
- **Runner lifecycle**: `Playing` ⇔ runner registered; all stop paths (Pause, Reset, Delete, terminal tick, Shutdown) go through one `detach`.
- **Delivery**: every broadcast = full snapshot + monotonic `Seq`. Subscription resolver subscribes first, then emits current snapshot, then only increasing `Seq`. Frontend `lib/stores/session.ts` also merges by `seq`.
- **Map snapshot**: sessions embed their own map copy; map edit/delete never touches sessions.
- **Errors**: `*service.Error{Code,Msg}` (`NOT_FOUND`, `INVALID_PROGRAM`, `INVALID_ARGUMENT`, `SESSION_PLAYING`, `SESSION_TERMINAL`, `FORBIDDEN`, `LIMIT_REACHED`); resolvers surface as `extensions.code`; unknown errors log and return `INTERNAL`. Missing map/session query → GraphQL `null`. Check with `errors.As`.
- **Security**: map ids `^[a-z0-9_-]{1,64}$` (they are filenames); session ids = 16 lowercase hex, validated before touching disk; admin mutations need `X-Admin-Key` == `ADMIN_API_KEY` unless `ALLOW_UNAUTHENTICATED_ADMIN=true` (only honored when key unset); `max_steps ≤ 10000`, `max_call_depth ≤ 64`; solutions never appear in the schema or static output.

## Key Directories
| Path | Purpose |
|---|---|
| `game/engine` | rules, VM, simulate (`types.go`, `engine.go`) |
| `game/service` | `Session`, `GameService` (`impl.go`), `Runner` (`runner.go`), `errors.go` |
| `game/config` | map store: `maps/*.json`, RWMutex, atomic temp+rename writes |
| `game/session` | in-memory registry + `sessions/<id>.json` versioned persistence |
| `transport/websocket` | `Hub` (Broadcaster impl; no raw WS clients) |
| `graph/` | `schema.graphqls`, resolvers, `convert.go`, admin gate in `resolver.go`; `generated/`, `model/` are gqlgen output (don't hand-edit) |
| `api/` | mux, body cap (1 MiB), complexity cap (1000), WS origin check, SPA fallback (`spaRoutes`) |
| `validate/`, `cmd/validate` | map + reachability + reference-solution checks |
| `maps/`, `solutions/`, `map-schema.json` | data; `sessions/` is runtime, gitignored |
| `frontend/` | SvelteKit + Tailwind v4; builds into `static/` (committed) |

## Development Commands
```bash
make build | run PORT=8000 | dev        # dev = go run . -port 8000 -debug
make test                                # go test -race ./...
go test ./game/engine -run '^TestGoldenSolutions$'   # single test
make vet | fmt | lint                    # lint needs golangci-lint (govet + ineffassign only)
make validate                            # all maps + reference solutions
make generate                            # gqlgen, after editing graph/schema.graphqls
make verify                              # gofmt check, vet, lint, test (needs golangci-lint; CLAUDE.md's full check also runs make validate)
make build-frontend                      # npm ci if needed + build:static → static/
cd frontend && npm run check && npm test && npm run build:static
go run . -port 9191 && scripts/smoke.sh 9191   # smoke.sh starts its own temp server; covers create/set-program/simulate/run/llms.txt, not UI or subscriptions
```
Flags: `-port 8000 -host 0.0.0.0 -maps-dir -sessions-dir -solutions-dir -debug -public-url`. Env (`.env` via godotenv): `ADMIN_API_KEY`, `ALLOW_UNAUTHENTICATED_ADMIN`, `ALLOWED_ORIGINS` (WS upgrades + `/graphql` CORS; empty/`*` = all), `GRAPHQL_INTROSPECTION`, `GRAPHQL_PLAYGROUND`, `MAX_SESSIONS=1000`, `SESSION_TTL=168h`. Frontend env: `PUBLIC_GRAPHQL_URL`, `PUBLIC_WS_URL`.

## Code Conventions & Common Patterns
- Standard `gofmt`; domain-noun types (`GameService`, `MapStore`), concrete impls usually `Manager`; GraphQL camelCase, enums/codes `UPPER_SNAKE`.
- Mutating service method pattern: lock session → mutate → `touch` (Seq++, LastActionAt) → `Clone` → unlock → persist → broadcast the clone.
- Persistence: temp file + rename (`game/session/file_persistence.go`, `game/config/manager.go`); load-time validation; restored sessions are never `Playing`.
- JSON: `sub_tape_lengths: []` (zero subs) ≠ omitted (default `[10]`). No `omitempty`, and never turn `[]` into `nil`.
- Schema change flow: edit `graph/schema.graphqls` → `make generate` → update `graph/convert.go` + resolvers → mirror by hand in `frontend/src/lib/{queries,types}.ts`.
- Frontend: Svelte 5 runes only (`$state/$derived/$effect/$props`), `onclick` attributes, keyed `{#each}`; urql (network-only, POST) + lazy shared `graphql-ws`; `unwrap()` turns urql failures into exceptions. **Gotcha**: explicit `onchange/oninput` handlers run *before* `bind:value` updates — read `event.currentTarget.value`, not the bound var.
- New top-level route → also add to `spaRoutes` in `api/server.go`.
- Don't run `go mod tidy` casually (gqlgen generate rewrites go.mod; re-`go get` dropped deps).

## Important Files
`main.go`, `api/server.go`, `graph/schema.graphqls`, `graph/resolver.go` (admin gate), `graph/schema.resolvers.go`, `graph/convert.go`, `game/engine/{types,engine}.go`, `game/service/{impl,runner,errors}.go`, `transport/websocket/hub.go`, `llms.txt.tmpl`, `map-schema.json`, `gqlgen.yml`, `Makefile`, `frontend/src/lib/{queries,types,graphql}.ts`, `frontend/src/lib/stores/{session,source}.ts`, `frontend/src/routes/{play,watch,editor,sessions,maps}`.

Adding a map: create `maps/<id>.json` (grid 3–20 × 3–20; glyphs `. # *` and one start `^ > v <`; ≥1 treat; main tape 1–32, 0–3 subs each 1–16) + `solutions/<id>.json` (`main`, `subs`), then `make validate`.

## Runtime/Tooling Preferences
Go 1.25; gqlgen 0.17.73, gorilla/mux, gorilla/websocket, godotenv. Frontend: **npm** (lockfile v3, `.npmrc engine-strict=true`), Node `^20.19 || >=22.12`, Vite, adapter-static (`fallback: index.html`). Go serves `frontend/build` first, then `static/`. `static/` is committed — rebuild and commit after UI changes. `graph/generated` is committed.

## Testing & QA
- Go: stdlib `testing`, `t.Helper/TempDir/Cleanup`, table + `t.Run`, no `t.Parallel()` (tests use `t.Setenv`). Service tests use `newEnv` in `game/service/helpers_test.go` (fake MapStore, spy persistence, broadcaster recorder); poll with `waitFor`, never fixed sleeps. API tests use `httptest.Server`. Always run with `-race`.
- Golden step counts (deliberate changes only): straight_line 7, turn_challenge 17, rock_obstacle 19, symmetric_paths 46, zigzag_path 23, long_corridor 30, staircase 44 (`validate/validate_test.go`; engine tests cover 7, 46, 23, 30, 44).
- Frontend: Vitest + jsdom + Testing Library (`cd frontend && npm test`, single file `npm test -- src/lib/grid.test.ts`). No coverage thresholds configured.
- UI changes: load `/play/<id>` and `/watch/<id>` in a browser.

## Hardening notes (fixed; keep invariants)
- Hub drops broadcasts with `Seq` ≤ last published per session; don't bypass it.
- `api/server.go`: WS `InitTimeout` 10s + frame-size guard (`api/wslimit.go`, gqlgen has no read-limit hook); complexity weighted via `complexityRoot()` (`sessions` by `limit`, `simulate`/`includeEvents` surcharge) — `api/hardening_test.go` holds a copy of frontend queries, keep in sync with `frontend/src/lib/queries.ts`; CORS on `/graphql` reuses `ALLOWED_ORIGINS`.
- Map creation goes through atomic `GameService.CreateMap` → `MapStore.Create` (conflict = `INVALID_ARGUMENT`); never check-then-save.
- `NewMap` bounds-checks before allocating; `SubIndex` accepts only canonical `CALL_SUB_1..3`; admin key compared via sha256 digests; atomic writes fsync file + dir.
- `graph/convert.go` keeps `subTapeLengths: []` non-nil.

## Known Issues
- None tracked. Add items here only when verified.
