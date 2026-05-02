<!-- Developed by Taipei Urban Intelligence Center 2023-2024-->

<script setup>
/* global gtag */
import { onMounted, ref, computed, watch, markRaw } from "vue";
import { useRoute } from "vue-router";
import mapboxGl from "mapbox-gl";
import { useAuthStore } from "../../store/authStore";
import { useContentStore } from "../../store/contentStore";
import { useDialogStore } from "../../store/dialogStore";
import { useMapStore } from "../../store/mapStore";
import { useChatStore } from "../../store/chatStore";

import MobileLayers from "../dialogs/MobileLayers.vue";
import IncidentReport from "../dialogs/IncidentReport.vue";
import FindClosestPoint from "../dialogs/FindClosestPoint.vue";

const authStore = useAuthStore();
const mapStore = useMapStore();
const dialogStore = useDialogStore();
const contentStore = useContentStore();
const chatStore = useChatStore();
const route = useRoute();

const districtLayer = ref(false);
const villageLayer = ref(false);

const canUseFindClosestPoint = computed(() => {
	let pointLayerCount = 0;

	mapStore.currentVisibleLayers.forEach((layer) => {
		if (["circle", "symbol"].includes(layer.split("-")[1])) {
			pointLayerCount++;
		}
	});

	return pointLayerCount === 1;
});

function toggleDistrictLayer() {
	districtLayer.value = !districtLayer.value;
	mapStore.toggleDistrictBoundaries(districtLayer.value);
	// 載入區界時觸發GA自訂事件
	gtag("event", "map_actions", {
		action_type: "載入區界",
		time: Date.now(),
	});
}

function toggleVillageLayer() {
	villageLayer.value = !villageLayer.value;
	mapStore.toggleVillageBoundaries(villageLayer.value);
	// 載入里界時觸發GA自訂事件
	gtag("event", "map_actions", {
		action_type: "載入里界",
		time: Date.now(),
	});
}

// 尋找最近點時觸發GA自訂事件
function findClosestPointGA() {
	gtag("event", "map_actions", {
		action_type: "尋找最近點",
		time: Date.now(),
	});
}

watch(
	() => route.query?.city,
	(newValue) => {
		newValue
			? mapStore.updateMapViewForCity(newValue)
			: mapStore.updateMapViewForCity("taipei");
	},
);

onMounted(() => {
	mapStore.initializeMapBox();
	mapStore.setCurrentLocation();
	route.query.city
		? mapStore.updateMapViewForCity(route.query.city)
		: mapStore.updateMapViewForCity("taipei");
	mapStore.map.on("dblclick", (event) => {
		if (route.name === "ai-tour") {
			const locationCount = chatStore.attachments.filter(
				(a) => a.type === "location",
			).length;
			if (locationCount >= 6) return;
			const el = document.createElement("div");
			el.className = "ai-tour-marker";
			el.textContent = String(locationCount + 1);
			const m = new mapboxGl.Marker({ element: el, anchor: "bottom" })
				.setLngLat(event.lngLat)
				.addTo(mapStore.map);
			chatStore.attachments.push({
				type: "location",
				lng: event.lngLat.lng,
				lat: event.lngLat.lat,
				marker: markRaw(m),
			});
		}
	});

	watch(
		() => chatStore.attachments.length,
		() => {
			let n = 1;
			chatStore.attachments.forEach((a) => {
				if (a.type === "location" && a.marker) {
					a.marker.getElement().textContent = String(n++);
				}
			});
		},
	);
});
</script>

<template>
	<div class="mapcontainer">
		<div class="mapcontainer-map">
			<!-- #mapboxBox needs to be empty to ensure Mapbox performance -->
			<div id="mapboxBox" />
			<div class="mapcontainer-layers">
				<button
					:style="{
						color: districtLayer
							? 'var(--color-highlight)'
							: 'var(--color-component-background)',
					}"
					@click="toggleDistrictLayer"
				>
					區
				</button>
				<button
					:style="{
						color: villageLayer
							? 'var(--color-highlight)'
							: 'var(--color-component-background)',
					}"
					@click="toggleVillageLayer"
				>
					里
				</button>

				<button
					v-if="canUseFindClosestPoint"
					:style="{
						color: villageLayer
							? 'var(--color-highlight)'
							: 'var(--color-component-background)',
					}"
					class="hide-if-mobile"
					type="button"
					@click="
						dialogStore.showDialog('findClosestPoint');
						findClosestPointGA();
					"
				>
					近
				</button>
				<button
					class="show-if-mobile"
					@click="dialogStore.showDialog('mobileLayers')"
				>
					<span>layers</span>
				</button>
				<div
					v-if="mapStore.loadingLayers.length > 0"
					class="mapcontainer-layers-loading"
				>
					<div />
				</div>
			</div>

			<button
				v-if="authStore.user.is_admin"
				class="mapcontainer-layers-incident"
				title="通報災害"
				@click="dialogStore.showDialog('incidentReport')"
			>
				!</button
			><!-- The key prop informs vue that the component should be updated when switching dashboards -->
			<MobileLayers :key="contentStore.currentDashboard.index" />
			<IncidentReport />
			<FindClosestPoint />
		</div>

	</div>
</template>

<style scoped lang="scss">
.mapcontainer {
	position: relative;
	width: 100%;
	height: 100%;
	flex: 1;

	&-map {
		height: 100%;
	}

	&-layers {
		position: absolute;
		right: 10px;
		top: 150px;
		z-index: 1;
		display: flex;
		flex-direction: column;
		row-gap: 4px;

		button {
			width: 1.75rem;
			height: 1.75rem;
			display: flex;
			align-items: center;
			justify-content: center;
			border-radius: 50%;
			background-color: white;
			transition: color 0.2s;
		}

		span {
			color: var(--color-component-background);
			font-size: 1.2rem;
			font-family: var(--font-icon);
		}

		&-loading {
			height: 2rem;
			display: flex;
			align-items: center;
			justify-content: center;
			z-index: 20;

			@media (max-width: 1000px) {
				top: 145px;
			}

			div {
				width: 1.3rem;
				height: 1.3rem;
				border-radius: 50%;
				border: solid 4px var(--color-border);
				border-top: solid 4px var(--color-highlight);
				animation: spin 0.7s ease-in-out infinite;
			}
		}

		&-incident {
			position: absolute;
			right: 10px;
			bottom: 60px;
			width: 50px;
			height: 50px;
			border-radius: 50%;
			background-color: var(--color-component-background);
			display: flex;
			align-items: center;
			justify-content: center;
			transition:
				background-color 0.2s,
				color 0.2s;
			font-size: var(--font-xl);

			&:hover {
				background-color: var(--color-highlight);
			}
		}
	}
}

#mapboxBox {
	width: 100%;
	height: 100%;
	border-radius: 5px;
}

</style>

<style lang="scss">
.ai-tour-marker {
	width: 28px;
	height: 28px;
	border-radius: 50%;
	background-color: #5a9cf8;
	color: white;
	font-size: 14px;
	font-weight: bold;
	display: flex;
	align-items: center;
	justify-content: center;
	border: 2px solid white;
	box-shadow: 0 2px 6px rgba(0, 0, 0, 0.4);
	cursor: pointer;
}
</style>
