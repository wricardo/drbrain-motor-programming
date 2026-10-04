import type { Facing } from './types';

/** Clockwise rotation (degrees) of an upward-pointing arrow for each facing. */
export function facingDegrees(f: Facing): number {
	return { UP: 0, RIGHT: 90, DOWN: 180, LEFT: 270 }[f];
}

/**
 * Returns an angle equivalent to the facing that is reachable from prev by the
 * shortest turn, so CSS transitions never spin the long way round (e.g. LEFT→UP).
 */
export function nextAngle(prev: number, facing: Facing): number {
	const target = facingDegrees(facing);
	let delta = (target - prev) % 360;
	if (delta > 180) delta -= 360;
	if (delta < -180) delta += 360;
	return prev + delta;
}
