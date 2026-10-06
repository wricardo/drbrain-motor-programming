/** Minimal rectangle (a DOMRect satisfies it). */
export interface Rect {
	left: number;
	top: number;
	width: number;
	height: number;
}

/**
 * Places a popover of size `size` next to `anchor` inside a viewport of
 * `viewport` size: below the anchor, left-aligned, flipping above when it does
 * not fit below, and clamped so it stays `margin` px inside the viewport.
 */
export function placePopover(
	anchor: Rect,
	size: { width: number; height: number },
	viewport: { width: number; height: number },
	gap = 6,
	margin = 8
): { left: number; top: number } {
	const below = anchor.top + anchor.height + gap;
	const above = anchor.top - gap - size.height;
	const top = below + size.height <= viewport.height - margin || above < margin ? below : above;
	const maxLeft = viewport.width - margin - size.width;
	const left = Math.max(margin, Math.min(anchor.left, maxLeft));
	return { left, top: Math.max(margin, top) };
}
