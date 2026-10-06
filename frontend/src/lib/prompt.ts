import type { GameMap } from './types';

/** The minimal slice of a session the prompt needs. */
import { TITLE } from './brand';

export interface PromptSession {
	id: string;
	map: GameMap;
}

const instructionList = (subCount: number): string =>
	['EMPTY', 'TURN_LEFT', 'TURN_RIGHT', 'MOVE_FORWARD', ...Array.from({ length: subCount }, (_, i) => `CALL_SUB_${i + 1}`)].join(', ');

const emptyTape = (n: number): string => `[${Array(n).fill('EMPTY').join(', ')}]`;

/** Renders the layout with an x ruler and a y prefix so an agent can read coordinates off it directly. */
export function layoutBlock(layout: string[]): string {
	const width = layout.reduce((w, r) => Math.max(w, r.length), 0);
	const pad = String(Math.max(0, layout.length - 1)).length;
	const ruler = ' '.repeat(pad + 3) + Array.from({ length: width }, (_, x) => String(x % 10)).join('');
	const rows = layout.map((row, y) => `y${String(y).padStart(pad, ' ')}  ${row}`);
	return [ruler, ...rows].join('\n');
}

const FACING_TEXT: Record<string, string> = { UP: 'up (y-1)', RIGHT: 'right (x+1)', DOWN: 'down (y+1)', LEFT: 'left (x-1)' };

/**
 * Prompt that makes an AI agent play (and win) one specific, existing session through the GraphQL API.
 * Built only from the map, so it stays stable while the session changes. It never contains a solution.
 */
export function buildSessionPrompt(origin: string, s: PromptSession): string {
	const m = s.map;
	const subs = m.subTapeLengths;
	const tapes = [`main = ${m.mainTapeLength} slots`, ...subs.map((n, i) => `sub${i + 1} (subs[${i}]) = ${n} slots`)].join('; ');
	const skeleton = `main: ${emptyTape(m.mainTapeLength)}\n    subs: [${subs.map(emptyTape).join(', ')}]`;
	const treats = m.treats.map((t) => `(${t.x},${t.y})`).join(' ');
	const recursion = m.allowRecursion
		? `Recursion is ALLOWED on this map: a sub may call itself (or any sub already running). The call stack is bounded by maxCallDepth = ${m.maxCallDepth} and by the step limit, and there is no tail-call optimisation (the stack stays deep until a sub finishes).`
		: `Recursion is NOT allowed on this map: a CALL_SUB_k aimed at the sub that is running, or at any sub already waiting on the call stack, does nothing (the step is still spent). A sub may call a different sub that is not on the stack, so you can nest subs; maxCallDepth = ${m.maxCallDepth}.`;

	return `Use this GraphQL API to play an existing ${TITLE} session and WIN it.

${TITLE} is a robot programming puzzle. You do not steer the robot: you write it a program (a main tape plus ${subs.length} subroutine ${subs.length === 1 ? 'tape' : 'tapes'} of instructions), the server runs it one instruction per step, and you win the moment every treat has been collected.

Session ID: ${s.id}
GraphQL endpoint: ${origin}/graphql
Playground: ${origin}/playground
Full rules and every field: ${origin}/llms.txt
Spectators can watch live at: ${origin}/watch/${s.id}

GraphQL introspection is usually enabled; use the Playground or __schema/__type if you need to discover fields.

## This map: ${m.name} (${m.width}x${m.height}, ${m.difficulty})
${m.description}

Layout (x = column, y = row; row 0 is the TOP and y grows DOWNWARD):
${layoutBlock(m.layout)}

'.' empty, '#' rock (blocks), '*' treat, '^ > v <' = robot start cell and initial facing.
Robot starts at (${m.start.x},${m.start.y}) facing ${FACING_TEXT[m.startFacing] ?? m.startFacing}.
Treats (${m.treats.length}): ${treats}

Tapes (setProgram needs EXACTLY these lengths; pad unused slots with EMPTY): ${tapes}.
Limits: at most ${m.maxSteps} steps; ${recursion}

## Rules that matter
- Instructions: ${instructionList(subs.length)}.
- Each executed instruction is one step (a blocked move and a CALL both count). EMPTY slots are skipped for free.
- MOVE_FORWARD into a rock or off the map is a harmless no-op (still a step, "blocked: true"). Treats are collected on arrival.
- When a sub tape ends, control returns to the caller; when the main tape ends with treats left, you lose (PROGRAM_ENDED). Other losses: STEP_LIMIT, CALL_DEPTH.
- Win beats the step limit on the same step. setProgram and reset put the robot back at the start. setProgram is rejected while the session is playing (SESSION_PLAYING); finished sessions refuse run/step until reset (SESSION_TERMINAL).

## How to win
1. Plan on paper first: write the route as straight runs and turns using the coordinates above. Tapes are tiny, so look for repeated patterns and put them in subroutines (a sub can be a "row", a "U-turn", a "spiral side"...). Remember a blocked move costs a step but does no harm, so an over-long run of MOVE_FORWARD is safe when a wall will stop it.
2. Store the program in the session:
mutation {
  setProgram(sessionID: "${s.id}", program: {
    ${skeleton}
  }) { id vm { status steps } }
}
   (Replace the EMPTY slots with real instructions.)
3. Run it on the server (50-5000 ms per step) or step manually, then read the result:
mutation { run(sessionID: "${s.id}", speedMs: 100) { playing vm { status steps } } }
mutation { step(sessionID: "${s.id}") { vm { status steps pos { x y } } lastEvent { instruction blocked } } }
query { session(id: "${s.id}") { playing attempts bestSteps vm { status lossReason steps pos { x y } facing treatsRemaining { x y } callStack { tape pc } } } }
4. If the status is LOST, read lossReason and treatsRemaining, fix the program, and repeat from step 2 (call reset first if you want to run again without changing the program). Do not stop until vm.status is WON, then report the winning program and its step count.

Other mutations: pause, reset, renameSession, deleteSession. Do not try to guess other sessions' ids and do not delete this one.`;
}

/**
 * Generic prompt (no session yet): the agent reads llms.txt, creates its own session on a map it picks and wins it.
 */
export function buildGenericPrompt(origin: string): string {
	return `Play ${TITLE}, a robot programming puzzle, through its GraphQL API and win it.

You write the robot a program (a main tape plus up to 3 subroutine tapes); the server runs it one instruction per step; you win when every treat has been collected. Tapes are tiny, so the trick is spotting repeated patterns and reusing them with subroutines.

1. Read ${origin}/llms.txt first: it has the complete rules, the coordinate convention (row 0 is the top, y grows downward) and working examples for every step.
2. GraphQL endpoint: ${origin}/graphql (Playground: ${origin}/playground).
3. List the maps: query { maps { id name difficulty layout mainTapeLength subTapeLengths maxSteps maxCallDepth allowRecursion } } and pick one (start with an easy one).
4. Create your own session: mutation { createSession(mapID: "MAP_ID", displayName: "my agent") { id } }
5. Plan the route from the layout, then setProgram, run, and read the session until vm.status is WON. If it is LOST, read lossReason, fix the program and try again.
6. Tell me the session id so I can watch it live at ${origin}/watch/SESSION_ID, and report the winning program and its step count.`;
}
