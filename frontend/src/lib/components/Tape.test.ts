import { cleanup, fireEvent, render, screen } from '@testing-library/svelte';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { emptyProgram, setSlot } from '../program';
import Tape from './Tape.svelte';

afterEach(cleanup);

const cfg = { mainTapeLength: 3, subTapeLengths: [2] };

describe('Tape', () => {
	it('places the selected palette instruction when a slot is clicked', async () => {
		const onchange = vi.fn();
		render(Tape, { program: emptyProgram(cfg), selected: 'TURN_LEFT', onchange });
		await fireEvent.click(screen.getByTestId('slot--1-1'));
		expect(onchange.mock.calls[0][0].main).toEqual(['EMPTY', 'TURN_LEFT', 'EMPTY']);
	});

	it('keyboard: F fills the focused slot, Backspace clears it, arrows move focus', async () => {
		const onchange = vi.fn();
		const { rerender } = render(Tape, { program: emptyProgram(cfg), onchange });
		const slot0 = screen.getByTestId('slot--1-0');
		slot0.focus();
		await fireEvent.keyDown(slot0, { key: 'f' });
		expect(onchange.mock.calls[0][0].main[0]).toBe('MOVE_FORWARD');
		expect(document.activeElement).toBe(screen.getByTestId('slot--1-1')); // advances after placing

		await fireEvent.keyDown(document.activeElement!, { key: 'ArrowDown' });
		expect(document.activeElement).toBe(screen.getByTestId('slot-0-1'));

		await rerender({ program: setSlot(emptyProgram(cfg), -1, 2, 'TURN_RIGHT'), onchange });
		const slot2 = screen.getByTestId('slot--1-2');
		await fireEvent.keyDown(slot2, { key: 'Backspace' });
		expect(onchange.mock.calls.at(-1)![0].main[2]).toBe('EMPTY');
	});

	it('call shortcut for a missing sub is ignored', async () => {
		const onchange = vi.fn();
		render(Tape, { program: emptyProgram(cfg), onchange });
		await fireEvent.keyDown(screen.getByTestId('slot--1-0'), { key: '2' });
		expect(onchange).not.toHaveBeenCalled();
	});

	it('read-only tapes ignore edits but still allow navigation, and expose highlights to AT', async () => {
		const onchange = vi.fn();
		render(Tape, {
			program: emptyProgram(cfg),
			disabled: true,
			selected: 'MOVE_FORWARD',
			callStack: [
				{ tape: -1, pc: 2 },
				{ tape: 0, pc: 1 }
			],
			onchange
		});
		await fireEvent.click(screen.getByTestId('slot--1-0'));
		await fireEvent.keyDown(screen.getByTestId('slot--1-0'), { key: 'f' });
		expect(onchange).not.toHaveBeenCalled();
		expect(screen.getByTestId('slot--1-1').getAttribute('aria-label')).toMatch(/waiting for subroutine/);
		expect(screen.getByTestId('slot-0-1').getAttribute('aria-label')).toMatch(/next to run/);
	});
});
