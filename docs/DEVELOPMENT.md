# Development

Technical reference for running, extending and contributing to Motor Programming. Players should start at the [README](../README.md).

Stack: Go server (pure rules engine → session manager → one `GameService` facade → GraphQL via gqlgen, with subscriptions) and a SvelteKit UI. Maps are JSON files, each session is persisted to its own file, and spectators watch through a GraphQL subscription.

## Run

```sh
cp .env.example .env      # optional
make run                  # builds and serves http://localhost:8000
# or
go run . -port 9191 -maps-dir maps -sessions-dir sessions
```

- UI: `/` (landing page, map picker, recent sessions), `/sessions` (all sessions: filter by name/map/status, sort, "mine only"), `/play/<session>`, `/watch/<session>`, `/maps`, `/editor`, `/learn` (rules, how AI agents play, strategies, a copyable generic AI prompt)
- **Play with an AI**: every `/play/<id>` and `/watch/<id>` page has a card that copies a prompt (session id, map layout, tape lengths, recursion rule, ready-to-run GraphQL calls; never a solution) so an AI agent can play that session to win. Built by `frontend/src/lib/prompt.ts`.
- GraphQL: `/graphql` (playground at `/playground`), LLM guide at `/llms.txt`
- Rebuild the UI: `make build-frontend` (builds `frontend/` and copies it into `static/`, which is committed)

## Instructions

`TURN_LEFT`, `TURN_RIGHT`, `MOVE_FORWARD`, `CALL_SUB_1..3`, `EMPTY` (skipped, not counted). Moving into a rock or off the grid is a silent no-op that still counts as a step.

Coordinates: `x` = column, `y` = row, row 0 is the top of the layout, **y grows downward**. Facing: up (y-1), right, down, left.

## Maps (`maps/*.json`)

```json
{
  "id": "symmetric_paths", "name": "Level 4 - Symmetric Paths", "difficulty": "easy",
  "layout": ["..........", ".*.......*", ".....#....", "..........", "..........", "..#..^..#.", "..........", "..........", ".....#....", ".*.......*"],
  "main_tape_length": 13, "sub_tape_lengths": [6, 6, 6],
  "max_steps": 200, "max_call_depth": 16, "allow_recursion": false
}
```

Glyphs: `.` empty, `#` rock, `*` treat, `^ > v <` start + facing (exactly one). Layout 3–20 per side. Defaults when omitted: main 10, subs `[10]`, max_steps 200, max_call_depth 16, recursion off. Schema: `map-schema.json`.

Shipped maps (11): five introductory levels (`straight_line`, `turn_challenge`, `rock_obstacle`, `symmetric_paths`, `zigzag_path`), plus puzzles whose tapes are tight enough that the solution is *provably* forced (`make validate` and `validate_test.go` check each bound):

| map | idea | why it's forced |
|---|---|---|
| `long_corridor`, `staircase` | recursion | non-recursive programs execute fewer forward moves than the treat distance |
| `serpentine` | mirrored row + U-turn subs that share a row sub | 61 actions needed; main-only sub calls run at most 7 × 7 = 49 |
| `powers` | each sub triples the one below (S1=2, S2=6, S3=18 moves) | 19 moves needed; two nesting levels reach at most 18 |
| `pinwheel` | one out-and-back unit repeated per arm, each ending facing the next arm | 35 actions needed; one nesting level runs at most 4 × 5 = 20 |
| `bump_spiral` | recursion; walls absorb overshoot, so one pattern follows every side | 56 actions needed; without recursion at most 3 × 10 = 30 |

### Reference solutions

Reference solutions live in `solutions/<id>.json` (`{"main": [...], "subs": [[...]]}`, exact tape lengths). They are **not** in the public repo (`solutions/` is gitignored) and are never served by the API. Without them, `make validate` reports missing solutions and the solution-dependent tests in `validate/validate_test.go` are skipped.

### Adding a map

1. Add `maps/<id>.json` (id = filename, `^[a-z0-9_-]{1,64}$`).
2. Add `solutions/<id>.json`.
3. `make validate`. It checks structure, treat reachability, and that the reference solution WINs; for `allow_recursion` maps the solution must fail with recursion forced off.

Maps can also be created from the `/editor` UI or the admin mutations (`X-Admin-Key`).

## Engine semantics (`game/engine`)

- A **step** is one executed non-`EMPTY` instruction. `CALL_SUB_k` counts as a step.
- Call stack: `CallStack[0]` is main, the top frame is the active tape, `PC` = next instruction on that tape. Depth = `len(CallStack) - 1`; no tail-call elimination.
- Order after each step: all treats collected → **WON** (wins ties with the step cap); `steps >= max_steps` → **LOST/STEP_LIMIT**; call depth `> max_call_depth` → **LOST/CALL_DEPTH**; main tape exhausted → **LOST/PROGRAM_ENDED** (reported on the step that ran the last real instruction).
- `allow_recursion: false`: a call to a sub that is active or already anywhere on the call stack is a no-op (still a step). No self-call or mutual recursion is possible.

## Sessions

One run of one map. `setProgram` (rejected while playing) resets the VM; `run(speedMs 50..5000)` starts a server-side ticker; `step`, `pause`, `reset`. The map is snapshotted into the session at creation, so editing or deleting a map never breaks existing sessions. Every update carries a monotonically increasing `seq`; clients drop `seq <=` last seen. Subscribers get the current snapshot immediately, then every change, with latest-wins delivery so the final WON/LOST frame is never lost to a slow consumer.

Sessions are persisted to `sessions/<id>.json` on create, setProgram, pause, reset, terminal outcome and rename (not per tick), and on graceful shutdown. After a restart sessions come back with `playing=false`, program and VM intact.

Error codes (`extensions.code`): `NOT_FOUND`, `INVALID_PROGRAM`, `SESSION_PLAYING`, `SESSION_TERMINAL`, `INVALID_ARGUMENT`, `FORBIDDEN`, `LIMIT_REACHED`.

## Configuration (`.env`)

`GRAPHQL_INTROSPECTION`, `GRAPHQL_PLAYGROUND`, `ADMIN_API_KEY`, `ALLOW_UNAUTHENTICATED_ADMIN`, `ALLOWED_ORIGINS`, `MAX_SESSIONS`, `SESSION_TTL` (see `.env.example`).

## Deployment

Pushes to `main` run `.github/workflows/deploy.yml`: vet + test, build a linux/arm64 binary, ship it with `maps/` and `static/` to the EC2 host, restart the systemd unit (behind nginx + Let's Encrypt) and smoke-test GraphQL. Secrets: `EC2_HOST`, `EC2_SSH_KEY`, `EC2_KNOWN_HOSTS`.

## Development

```sh
make test       # go test -race ./...
make validate   # all maps + reference solutions
make generate   # regenerate gqlgen code after editing graph/schema.graphqls
make verify     # gofmt check, vet, lint, test
cd frontend && npm run check && npm test
scripts/smoke.sh 9191   # end-to-end HTTP smoke against a running server
```

See `CLAUDE.md` for the package map and conventions.
