# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository. `AGENTS.md` is a symlink to this file, so other coding agents read the same guide.

## Dr Brain - Motor Programming

Robot-programming puzzle game for humans and AI agents. Go server (pure rules engine → `GameService` → gqlgen GraphQL with subscriptions) serving a SvelteKit SPA. Repo, Go module (`github.com/wricardo/drbrain-motor-programming`) and binary share the name `drbrain-motor-programming`.

**Full reference: [`docs/DEVELOPMENT.md`](docs/DEVELOPMENT.md)** — architecture, rules and engine semantics, map format and design, API, frontend, UI look and feel, configuration, deployment, maintenance workflows. Read the relevant section before non-trivial changes and keep it up to date when behavior changes. `README.md` is player-facing only.

## Commands

```sh
make dev                                   # go run . -port 8000 -debug (API + UI)
make test                                  # go test -race ./...
go test -race -run TestName ./game/engine  # single Go test
make validate                              # maps + reference solutions (needs local solutions/)
make verify                                # gofmt, vet, lint (golangci-lint), test
make generate                              # gqlgen after editing graph/schema.graphqls
make build-frontend                        # build UI into static/ (committed)
cd frontend && npm run check && npm test
cd frontend && npx vitest run src/lib/grid.test.ts   # single frontend test
cd frontend && npm run dev:local           # Vite UI against a local server on :8000
scripts/smoke.sh 9191                      # end-to-end HTTP smoke (starts its own server)
```

## Invariants — do not break

- **Engine** (`game/engine`, no I/O): call `vm.Settle(&program)` after `NewVMState` and any program change. `Step` mutates in place; `Clone()` anything leaving a lock. `Run`/`step(…, nil)` allocate no events.
- **Dependencies**: `game/service` defines `MapStore`, `SessionStore`, `Broadcaster`; implementations import `service`, never the reverse.
- **Locking**: mutate pattern is lock → mutate → `touch` (seq++) → `Clone` → unlock → persist → broadcast. Never broadcast, do I/O, `Persist` or wait on a runner while holding a session lock. Lock order session → runner mutex.
- **Runner lifecycle**: `Playing` ⇔ runner registered; every stop path goes through one `detach`.
- **Delivery**: full snapshot + monotonic `Seq` on every broadcast; hub drops the *oldest* queued update and ignores `Seq` ≤ last published; subscription resolver emits the current snapshot first.
- **Map snapshot**: sessions embed their own map copy; map edits/deletes never touch sessions. Map creation is atomic via `MapStore.Create`, never check-then-save.
- **Security**: map ids `^[a-z0-9_-]{1,64}$` (filenames); session ids 16 lowercase hex, validated before disk; admin mutations need `X-Admin-Key` (constant-time) unless `ALLOW_UNAUTHENTICATED_ADMIN=true` with no key set; `max_steps ≤ 10000`, `max_call_depth ≤ 64`.
- **Recursion off**: calling a sub already on the call stack is a no-op (still a step).
- **Zero subs**: `sub_tape_lengths: []` ≠ omitted. No `omitempty`; never turn `[]` into `nil` (incl. `graph/convert.go`).

## Repo rules

- **Public repo.** Never mention other local/private repos in tracked files. README stays player-facing.
- **Solutions are private.** `solutions/` is gitignored and local only. Never commit it or put solution content in the schema, API, prompts, `/llms.txt` or `static/`. Solution-dependent tests skip via `requireSolutions` when it is absent.
- **Push to `main` deploys** to https://motor-programming.wricardo.net (`.github/workflows/deploy.yml`). CI does not build the UI: rebuild and commit `static/` with every UI change.
- `graph/generated`, `graph/model` are gqlgen output: don't hand-edit. Don't run `go mod tidy` casually (gqlgen rewrites `go.mod`; re-`go get` dropped deps).

## Change checklists

- **Schema**: `graph/schema.graphqls` → `make generate` → `graph/convert.go` + resolvers → `frontend/src/lib/{queries,types}.ts` → `llms.txt.tmpl` / `lib/prompt.ts` if agents are affected → query copies in `api/hardening_test.go`.
- **Engine**: update golden step counts in `game/engine/engine_test.go` and `validate/validate_test.go` (7, 17, 19, 46, 23, 30, 44, 80, 32, 48, 83, 65, 77, 57, 40, 30, 42) deliberately, never to hide a failure.
- **New map**: `maps/<id>.json` + local `solutions/<id>.json` → `make validate` → golden step count in `validate/validate_test.go`.
- **New page**: route under `frontend/src/routes/` + `spaRoutes` in `api/server.go` (+ nav in `+layout.svelte`).
- **UI**: Svelte 5 runes only; follow "UI look and feel" in `docs/DEVELOPMENT.md` (instruction colors/glyphs/keys live in `lib/instructions.ts`, brand names in `lib/brand.ts`). Gotcha: explicit `onchange`/`oninput` run before `bind:value` updates — read `event.currentTarget.value`.

## Testing

Go: table tests + `t.Run`, no `t.Parallel()` (uses `t.Setenv`), always `-race`; service tests use `newEnv` (`game/service/helpers_test.go`) and `waitFor`, never fixed sleeps. Frontend: Vitest + jsdom + Testing Library; logic lives in tested `lib/*.ts` modules.

**Before pushing**: `make verify && make validate`, `scripts/smoke.sh 9191` (starts its own server; port must be free); for UI changes open `/play/<id>` and `/watch/<id>` and run a program to the end.
