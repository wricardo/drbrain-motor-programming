<script lang="ts">
	/** A treat, drawn as a chocolate-chip cookie inside an <svg>; centred at (cx, cy). */
	let { cx, cy, r = 15 }: { cx: number; cy: number; r?: number } = $props();
</script>

<g transform="translate({cx} {cy}) scale({r / 15})" aria-hidden="true">
	<circle r="15" fill="#d9a066" stroke="#8a5a2b" stroke-width="1.5" />
	<circle cx="-5" cy="-6" r="2.4" fill="#4a2c12" />
	<circle cx="6" cy="-4" r="2.2" fill="#4a2c12" />
	<circle cx="-7" cy="4" r="2" fill="#4a2c12" />
	<circle cx="3" cy="7" r="2.5" fill="#4a2c12" />
	<circle cx="9" cy="5" r="1.6" fill="#4a2c12" />
	<circle cx="-1" cy="-1" r="1.6" fill="#4a2c12" />
</g>
