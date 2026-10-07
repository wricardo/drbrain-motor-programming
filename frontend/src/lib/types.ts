// Hand-written types mirroring graph/schema.graphqls. Keep in sync with the schema.

export type Instruction =
	| 'EMPTY'
	| 'TURN_LEFT'
	| 'TURN_RIGHT'
	| 'MOVE_FORWARD'
	| 'CALL_SUB_1'
	| 'CALL_SUB_2'
	| 'CALL_SUB_3';

export type Facing = 'UP' | 'RIGHT' | 'DOWN' | 'LEFT';
export type Status = 'READY' | 'RUNNING' | 'WON' | 'LOST';
export type LossReason = 'STEP_LIMIT' | 'CALL_DEPTH' | 'PROGRAM_ENDED';

/** Coordinates: x = column, y = row; y grows downward. */
export interface Position {
	x: number;
	y: number;
}

/** One entry of the VM call stack. tape -1 is the main tape; pc is the next instruction index. */
export interface Frame {
	tape: number;
	pc: number;
}

export interface GameMap {
	id: string;
	name: string;
	description: string;
	difficulty: string;
	width: number;
	height: number;
	layout: string[];
	start: Position;
	startFacing: Facing;
	rocks: Position[];
	treats: Position[];
	mainTapeLength: number;
	subTapeLengths: number[];
	maxSteps: number;
	maxCallDepth: number;
	allowRecursion: boolean;
}

export interface Program {
	main: Instruction[];
	subs: Instruction[][];
}

export interface VMState {
	pos: Position;
	facing: Facing;
	treatsRemaining: Position[];
	callStack: Frame[];
	steps: number;
	status: Status;
	lossReason: LossReason | null;
	visited: Position[];
}

export interface StepEvent {
	step: number;
	instruction: Instruction;
	tapeIndex: number;
	slotIndex: number;
	from: Position;
	to: Position;
	facingBefore: Facing;
	facingAfter: Facing;
	blocked: boolean;
	collected: Position | null;
	callStack: Frame[];
	status: Status;
	lossReason: LossReason | null;
}

export interface Session {
	id: string;
	displayName: string;
	mapId: string;
	map: GameMap;
	program: Program;
	vm: VMState;
	playing: boolean;
	speedMs: number;
	seq: number;
	createdAt: string;
	lastActionAt: string;
	attempts: number;
	bestSteps: number | null;
	lastEvent: StepEvent | null;
}

export interface SessionUpdate {
	seq: number;
	session: Session;
	event: StepEvent | null;
}

/** Lightweight session row used by the recent-sessions list. */
export interface SessionSummary {
	id: string;
	displayName: string;
	mapId: string;
	playing: boolean;
	createdAt: string;
	lastActionAt: string;
	map: { name: string };
	vm: { status: Status; steps: number; treatsRemaining: Position[] };
}

export interface MapValidationResult {
	valid: boolean;
	issues: { message: string }[];
	unreachableTreats: Position[];
}

export interface MapInput {
	id: string;
	name: string;
	description?: string;
	difficulty?: string;
	layout: string[];
	mainTapeLength?: number;
	subTapeLengths?: number[];
	maxSteps?: number;
	maxCallDepth?: number;
	allowRecursion?: boolean;
}

/** WebSocket connection indicator for spectator/play views. */
export type ConnectionState = 'connecting' | 'live' | 'reconnecting' | 'offline';
