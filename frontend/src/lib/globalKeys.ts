/** Page-level shortcuts (outside the tape editor). */
export type GlobalKeyAction = 'reset' | 'shortcuts';

/**
 * Maps a window keydown to a page shortcut: R resets, ? opens the shortcuts
 * dialog. Ignored with modifiers, while typing in a form field, and while a
 * program slot or the slot menu has focus (there R means "turn right").
 */
export function globalKeyAction(e: Pick<KeyboardEvent, 'key' | 'ctrlKey' | 'metaKey' | 'altKey' | 'target'>): GlobalKeyAction | null {
	if (e.ctrlKey || e.metaKey || e.altKey) return null;
	const el = e.target as HTMLElement | null;
	if (el && typeof el.closest === 'function') {
		if (el.closest('input, textarea, select, [contenteditable="true"], [data-slot], [data-testid="slot-picker"]')) return null;
	}
	if (e.key === '?') return 'shortcuts';
	if (e.key === 'r' || e.key === 'R') return 'reset';
	return null;
}
