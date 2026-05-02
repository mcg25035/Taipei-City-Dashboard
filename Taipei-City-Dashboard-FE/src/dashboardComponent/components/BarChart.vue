<!-- Developed by Taipei Urban Intelligence Center 2023-2024-->
<script setup>
import { ref, computed } from "vue";
import VueApexCharts from "vue3-apexcharts";

const props = defineProps([
	"chart_config",
	"activeChart",
	"series",
	"map_config",
	"map_filter",
	"map_filter_on",
]);

const emits = defineEmits([
	"filterByParam",
	"filterByLayer",
	"clearByParamFilter",
	"clearByLayerFilter",
	"fly",
]);

function hexToRgb(hex) {
	const cleaned = String(hex).trim().replace(/^#/, "");
	if (cleaned.length !== 6) return null;
	const num = parseInt(cleaned, 16);
	if (Number.isNaN(num)) return null;
	return { r: (num >> 16) & 255, g: (num >> 8) & 255, b: num & 255 };
}

function rgbToHex(r, g, b) {
	const toHex = (v) => Math.max(0, Math.min(255, Math.round(v))).toString(16).padStart(2, "0");
	return `#${toHex(r)}${toHex(g)}${toHex(b)}`;
}

function interpolateColor(c1, c2, t) {
	const a = hexToRgb(c1);
	const b = hexToRgb(c2);
	if (!a || !b) return c1 ?? c2 ?? "#000000";
	return rgbToHex(a.r + (b.r - a.r) * t, a.g + (b.g - a.g) * t, a.b + (b.b - a.b) * t);
}

function isChromatic(hex) {
	const rgb = hexToRgb(hex);
	if (!rgb) return false;
	return !(rgb.r === rgb.g && rgb.g === rgb.b);
}

function gradientColor(palette, ratio) {
	if (palette.length === 0) return "#000000";
	if (palette.length === 1) return palette[0];
	const clamped = Math.max(0, Math.min(1, ratio));
	const segment = clamped * (palette.length - 1);
	const idx = Math.min(Math.floor(segment), palette.length - 2);
	const t = segment - idx;
	return interpolateColor(palette[idx], palette[idx + 1], t);
}

const proportionalColors = computed(() => {
	const rawPalette = props.chart_config.color ?? [];
	const data = props.series?.[0]?.data ?? [];
	const palette = rawPalette.filter(isChromatic);
	const effectivePalette = palette.length > 0 ? palette : rawPalette;
	if (effectivePalette.length === 0 || data.length === 0) {
		return [...effectivePalette];
	}
	const numericValues = data.map((item) =>
		typeof item === "object" && item !== null ? Number(item.y) : Number(item),
	);
	const min = Math.min(...numericValues);
	const max = Math.max(...numericValues);
	const range = max - min;
	if (!isFinite(min) || !isFinite(max) || range === 0) {
		return numericValues.map(() => effectivePalette[0]);
	}
	return numericValues.map((value) =>
		gradientColor(effectivePalette, (value - min) / range),
	);
});

const chartOptions = computed(() => ({
	chart: {
		offsetY: 15,
		stacked: true,
		toolbar: {
			show: false,
		},
	},
	colors: proportionalColors.value,
	dataLabels: {
		offsetX: 20,
		textAnchor: "start",
	},
	grid: {
		show: false,
	},
	legend: {
		show: false,
	},
	plotOptions: {
		bar: {
			borderRadius: 2,
			distributed: true,
			horizontal: true,
			dataLabels: {
				hideOverflowingLabels: false,
			},
		},
	},
	stroke: {
		colors: ["#282a2c"],
		show: true,
		width: 0,
	},
	// The class "chart-tooltip" could be edited in /assets/styles/chartStyles.css
	tooltip: {
		custom: function ({ series, seriesIndex, dataPointIndex, w }) {
			return (
				'<div class="chart-tooltip">' +
				"<h6>" +
				w.globals.labels[dataPointIndex] +
				"</h6>" +
				"<span>" +
				series[seriesIndex][dataPointIndex] +
				` ${props.chart_config.unit}` +
				"</span>" +
				"</div>"
			);
		},
		followCursor: true,
	},
	xaxis: {
		axisBorder: {
			show: false,
		},
		axisTicks: {
			show: false,
		},
		labels: {
			show: false,
		},
		type: "category",
	},
	yaxis: {
		labels: {
			formatter: function (value) {
				return value.length > 7 ? value.slice(0, 6) + "..." : value;
			},
		},
	},
}));

const chartHeight = computed(() => {
	return `${40 + props.series[0].data.length * 30}`;
});

const selectedIndex = ref(null);

function handleDataSelection(_e, _chartContext, config) {
	if (!props.map_filter || !props.map_filter_on) {
		return;
	}
	if (
		`${config.dataPointIndex}-${config.seriesIndex}` !== selectedIndex.value
	) {
		// Supports filtering by xAxis
		if (props.map_filter.mode === "byParam") {
			emits(
				"filterByParam",
				props.map_filter,
				props.map_config,
				config.w.globals.labels[config.dataPointIndex],
				null,
			);
		}
		// Supports filtering by xAxis
		else if (props.map_filter.mode === "byLayer") {
			emits(
				"filterByLayer",
				props.map_config,
				config.w.globals.labels[config.dataPointIndex],
			);
		}
		selectedIndex.value = `${config.dataPointIndex}-${config.seriesIndex}`;
	} else {
		if (props.map_filter.mode === "byParam") {
			emits("clearByParamFilter", props.map_config);
		} else if (props.map_filter.mode === "byLayer") {
			emits("clearByLayerFilter", props.map_config);
		}
		selectedIndex.value = null;
	}
}
</script>

<template>
	<div v-if="activeChart === 'BarChart'">
		<VueApexCharts
			width="100%"
			:height="chartHeight"
			type="bar"
			:options="chartOptions"
			:series="series"
			@data-point-selection="handleDataSelection"
		/>
	</div>
</template>
