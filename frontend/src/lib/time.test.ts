import { describe, expect, it } from 'vitest';
import { relativeTime } from './time';

describe('relativeTime', () => {
	const now = new Date(2026, 9, 6, 12, 0, 0);
	const ago = (ms: number) => new Date(now.getTime() - ms).toISOString();

	it('formats recent times relatively and older ones as dates', () => {
		expect(relativeTime(ago(10_000), now)).toBe('just now');
		expect(relativeTime(ago(-5_000), now)).toBe('just now');
		expect(relativeTime(ago(2 * 60_000), now)).toBe('2m ago');
		expect(relativeTime(ago(3 * 3_600_000), now)).toBe('3h ago');
		expect(relativeTime(ago(2 * 86_400_000), now)).toBe('2d ago');
		expect(relativeTime(new Date(2026, 9, 4 - 7, 9).toISOString(), now)).toBe('Sep 27');
		expect(relativeTime(new Date(2025, 0, 2).toISOString(), now)).toBe('Jan 2, 2025');
	});

	it('returns invalid input unchanged', () => {
		expect(relativeTime('nope', now)).toBe('nope');
	});
});
