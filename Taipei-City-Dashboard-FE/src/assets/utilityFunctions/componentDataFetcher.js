import http from "../../router/axios";
import { getComponentDataTimeframe } from "./dataTimeframe";

const STATIC_TIME_VALUES = ["static", "current", "demo"];

export async function fetchChartData(component) {
	const response = await http.get(`/component/${component.id}/chart`, {
		params: {
			city: component.city,
			...(!STATIC_TIME_VALUES.includes(component.time_from)
				? getComponentDataTimeframe(component.time_from, component.time_to, true)
				: {}),
		},
	});
	component.chart_data = response.data.data;
	if (response.data.categories) {
		component.chart_config.categories = response.data.categories;
	}
}

export async function fetchHistoryData(component) {
	if (!component.history_config?.range) return;
	for (let i in component.history_config.range) {
		const response = await http.get(`/component/${component.id}/history`, {
			params: {
				city: component.city,
				...getComponentDataTimeframe(
					component.history_config.range[i],
					"now",
					true,
				),
			},
		});
		if (i === "0") component.history_data = [];
		component.history_data.push(response.data.data);
	}
}

export async function fetchComponentData(component) {
	try {
		await fetchChartData(component);
	} catch (error) {
		console.error(`Failed to fetch chart data for AI component ${component.id}:`, error);
		component.chart_data = [];
	}

	try {
		await fetchHistoryData(component);
	} catch (error) {
		console.error(`Failed to fetch history data for AI component ${component.id}:`, error);
		if (!component.history_data) {
			component.history_data = [];
		}
		component.history_data.push([]);
	}

	return component;
}
