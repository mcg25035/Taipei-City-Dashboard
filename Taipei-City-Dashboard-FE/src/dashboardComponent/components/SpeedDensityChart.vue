<script setup>
import { computed } from "vue";
import { useMapStore } from "../../store/mapStore";

const props = defineProps([
	"chart_config",
	"series",
	"map_config",
	"map_filter",
	"map_filter_on",
]);

const mapStore = useMapStore();

const STOPS = [
	{ speed: 10, color: "#c5d4fa" },
	{ speed: 30, color: "#829cf5" },
	{ speed: 50, color: "#660080" },
];
const SPEED_MIN = 0;
const SPEED_MAX = 60;
const BIN_WIDTH = 1;
const BIN_COUNT = Math.round((SPEED_MAX - SPEED_MIN) / BIN_WIDTH);
const KERNEL_BANDWIDTH = 2.5;

const sourceIndex = computed(() => {
	const cfg = Array.isArray(props.map_config)
		? props.map_config[0]
		: props.map_config;
	return cfg?.index || null;
});

const speeds = computed(() => {
	const idx = sourceIndex.value;
	const features = idx ? mapStore.featureCache?.[idx] : null;
	if (!features?.length) return [];
	const out = [];
	for (const f of features) {
		const raw = f?.properties?.travel_speed;
		const v = typeof raw === "number" ? raw : parseFloat(raw);
		if (Number.isFinite(v)) out.push(v);
	}
	return out;
});

const density = computed(() => {
	const xs = [];
	const ys = [];
	for (let i = 0; i <= BIN_COUNT; i++) {
		xs.push(SPEED_MIN + i * BIN_WIDTH);
	}
	if (!speeds.value.length) {
		return { xs, ys: xs.map(() => 0), max: 0 };
	}
	const h = KERNEL_BANDWIDTH;
	const norm = 1 / (Math.sqrt(2 * Math.PI) * h);
	for (const x of xs) {
		let sum = 0;
		for (const v of speeds.value) {
			const z = (x - v) / h;
			sum += Math.exp(-0.5 * z * z);
		}
		ys.push((sum * norm) / speeds.value.length);
	}
	const max = ys.reduce((m, v) => (v > m ? v : m), 0);
	return { xs, ys, max };
});

const VIEW_W = 320;
const VIEW_H = 140;
const PAD_X = 6;
const PAD_TOP = 8;
const CURVE_H = 96;
const BAR_TOP = CURVE_H + 14;
const BAR_H = 18;

const curvePath = computed(() => {
	const { xs, ys, max } = density.value;
	if (!xs.length || max <= 0) return "";
	const innerW = VIEW_W - PAD_X * 2;
	const sx = (x) =>
		PAD_X + (1 - (x - SPEED_MIN) / (SPEED_MAX - SPEED_MIN)) * innerW;
	const sy = (y) => PAD_TOP + CURVE_H - (y / max) * CURVE_H;
	let d = `M ${sx(xs[0]).toFixed(2)} ${(PAD_TOP + CURVE_H).toFixed(2)}`;
	for (let i = 0; i < xs.length; i++) {
		d += ` L ${sx(xs[i]).toFixed(2)} ${sy(ys[i]).toFixed(2)}`;
	}
	d += ` L ${sx(xs[xs.length - 1]).toFixed(2)} ${(PAD_TOP + CURVE_H).toFixed(2)} Z`;
	return d;
});

const gradientStops = computed(() =>
	STOPS.map((s) => ({
		offset: (1 - (s.speed - SPEED_MIN) / (SPEED_MAX - SPEED_MIN)) * 100,
		color: s.color,
	})).sort((a, b) => a.offset - b.offset),
);

const sampleCount = computed(() => speeds.value.length);
const meanSpeed = computed(() => {
	if (!speeds.value.length) return null;
	return speeds.value.reduce((a, b) => a + b, 0) / speeds.value.length;
});
</script>

<template>
	<div class="speeddensity">
		<svg
			:viewBox="`0 0 ${VIEW_W} ${VIEW_H}`"
			preserveAspectRatio="none"
			class="speeddensity-svg"
		>
			<defs>
				<linearGradient
					id="speeddensity-grad"
					x1="0"
					y1="0"
					x2="1"
					y2="0"
				>
					<stop
						v-for="s in gradientStops"
						:key="s.offset"
						:offset="`${s.offset}%`"
						:stop-color="s.color"
					/>
				</linearGradient>
			</defs>
			<path
				v-if="curvePath"
				:d="curvePath"
				fill="url(#speeddensity-grad)"
				fill-opacity="0.85"
				stroke="#a98ce0"
				stroke-width="1.2"
			/>
			<rect
				:x="PAD_X"
				:y="BAR_TOP"
				:width="VIEW_W - PAD_X * 2"
				:height="BAR_H"
				rx="3"
				ry="3"
				fill="url(#speeddensity-grad)"
			/>
		</svg>
		<div class="speeddensity-meta">
			<span>樣本 {{ sampleCount }} 路段</span>
			<span v-if="meanSpeed !== null">
				平均 {{ meanSpeed.toFixed(1) }} km/h
			</span>
			<span v-else>等候資料…</span>
		</div>
	</div>
</template>

<style scoped lang="scss">
.speeddensity {
	width: 100%;
	display: flex;
	flex-direction: column;
	gap: 4px;
	padding: 4px 0 2px;

	&-svg {
		width: 100%;
		height: auto;
		display: block;
	}

	&-meta {
		display: flex;
		justify-content: space-between;
		font-size: 0.7rem;
		color: var(--color-complement-text);
		padding: 0 6px;
	}
}
</style>
