<!-- Developed By Taipei Urban Intelligence Center 2023-2024 -->
<!-- Department of Information Technology, Taipei City Government -->

<script setup>
/* global gtag */
import { computed, ref, watch } from "vue";
import DashboardComponent from "../dashboardComponent/DashboardComponent.vue";
import { useContentStore } from "../store/contentStore";
import { useDialogStore } from "../store/dialogStore";
import { useMapStore } from "../store/mapStore";
import { useAgentEvent } from "../composables/useAgentEvent";
import MapContainer from "../components/map/MapContainer.vue";
import MoreInfo from "../components/dialogs/MoreInfo.vue";
import ReportIssue from "../components/dialogs/ReportIssue.vue";
import ChatBox from "../components/dialogs/ChatBox.vue";

const contentStore = useContentStore();
const dialogStore = useDialogStore();
const mapStore = useMapStore();
const { onEvent } = useAgentEvent();

onEvent('show_component', (component) => {
  contentStore.addAiSearchedComponent(component);
});

const toggleOn = ref({
	hasMap: [],
	noMap: [],
	basicLayer: [],
});

const parseMapLayers = computed(() => {
	const hasMap = contentStore.aiSearchedComponents.filter(
		(item) => item.map_config[0],
	);
	const noMap = contentStore.aiSearchedComponents.filter(
		(item) => !item.map_config[0],
	);
	return { hasMap, noMap };
});

function handleToggle(value, map_config) {
	if (!map_config[0]) {
		if (value) {
			dialogStore.showNotification("info", "本組件沒有空間資料，不會渲染地圖");
		}
		return;
	}
	if (value) {
		mapStore.addToMapLayerList(map_config);
	} else {
		mapStore.clearByParamFilter(map_config);
		mapStore.turnOffMapLayerVisibility(map_config);
	}
}

function toggleSwitchBtn(value, Btn, BtnIndex) {
	toggleOn.value[Btn][BtnIndex] = value;
}

function shouldDisable(map_config) {
	const allMapLayerIds = map_config.map(
		(el) => `${el.index}-${el.type}-${el.city}`,
	);
	if (mapStore.isPreloading === true) return true;
	return mapStore.loadingLayers.filter((el) => allMapLayerIds.includes(el)).length > 0;
}

function popularThematicLayerGA(map_config) {
	if (map_config[0].city && map_config[0].title) {
		gtag("event", "popular_thematic_layer", {
			dashboard_city: map_config[0].city,
			layer_name: map_config[0].title,
			city_layer: `${map_config[0].city}-${map_config[0].title}`,
			time: Date.now(),
		});
	}
}

function popularBasicLayerGA(map_config) {
	if (map_config[0].city && map_config[0].title) {
		gtag("event", "popular_basic_layer", {
			dashboard_city: map_config[0].city,
			layer_name: map_config[0].title,
			city_layer: `${map_config[0].city}-${map_config[0].title}`,
			time: Date.now(),
		});
	}
}
</script>

<template>
  <div class="aitour">
    <div class="aitour-map">
      <div class="hide-if-mobile">
        <!-- 1. AI searched components -->
        <div
          v-if="contentStore.aiSearchedComponents.length !== 0"
          class="map-charts"
        >
          <DashboardComponent
            v-for="(item, arrayIdx) in parseMapLayers.hasMap"
            :key="`map-layer-${item.index}-${item.city}`"
            :config="item"
            mode="map"
            :info-btn="true"
            :active-city="item.city"
            :select-btn="true"
            :select-btn-disabled="
              contentStore.cityManager.getSelectList(item.city).length === 1
            "
            :select-btn-list="
              contentStore.cityManager.getSelectList(item.city)
            "
            :city-tag="contentStore.cityManager.getTagList(item.city)"
            :toggle-disable="shouldDisable(item.map_config)"
            :toggle-on="toggleOn.hasMap[arrayIdx]"
            @info="(item) => dialogStore.showMoreInfo(item)"
            @toggle="
              (value, map_config) => {
                handleToggle(value, map_config);
                toggleSwitchBtn(value, 'hasMap', arrayIdx);
                popularThematicLayerGA(map_config);
              }
            "
            @filter-by-param="
              (map_filter, map_config, x, y) =>
                mapStore.filterByParam(map_filter, map_config, x, y)
            "
            @filter-by-layer="
              (map_config, layer) => mapStore.filterByLayer(map_config, layer)
            "
            @clear-by-param-filter="
              (map_config) => mapStore.clearByParamFilter(map_config)
            "
            @clear-by-layer-filter="
              (map_config) => mapStore.clearByLayerFilter(map_config)
            "
            @fly="(location) => mapStore.flyToLocation(location)"
            @change-city="
              (city) => {
                const selectedData = contentStore.aiSearchedComponents.find(
                  (data) => data.index === item.index && data.city === city,
                );
                const componentIndex =
                  contentStore.aiSearchedComponents.findIndex(
                    (data) => data.index === item.index && data.city === item.city,
                  );
                if (selectedData && componentIndex !== -1) {
                  mapStore.clearByParamFilter(item.map_config);
                  mapStore.turnOffMapLayerVisibility(item.map_config);
                  mapStore.addToMapLayerList(selectedData.map_config);
                  contentStore.setAiSearchedComponentData(componentIndex, selectedData);
                }
              }
            "
          />
          <h2 v-if="parseMapLayers.noMap?.length > 0">
            無空間資料組件
          </h2>
          <DashboardComponent
            v-for="(item, arrayIdx) in parseMapLayers.noMap"
            :key="`map-layer-${item.index}-${item.city}`"
            :config="item"
            mode="map"
            :info-btn="true"
            :active-city="item.city"
            :select-btn="true"
            :select-btn-disabled="
              contentStore.cityManager.getSelectList(item.city).length === 1
            "
            :select-btn-list="
              contentStore.cityManager.getSelectList(item.city)
            "
            :city-tag="contentStore.cityManager.getTagList(item.city)"
            :toggle-on="toggleOn.noMap[arrayIdx]"
            @info="(item) => dialogStore.showMoreInfo(item)"
            @toggle="
              (value, map_config) => {
                handleToggle(value, map_config);
                toggleSwitchBtn(value, 'noMap', arrayIdx);
              }
            "
            @change-city="
              (city) => {
                const selectedData = contentStore.aiSearchedComponents.find(
                  (data) => data.index === item.index && data.city === city,
                );
                const componentIndex =
                  contentStore.aiSearchedComponents.findIndex(
                    (data) => data.index === item.index && data.city === item.city,
                  );
                if (selectedData && componentIndex !== -1) {
                  contentStore.setAiSearchedComponentData(componentIndex, selectedData);
                }
              }
            "
          />
        </div>
        <!-- 2. Empty -->
        <div
          v-else
          class="map-charts-nodashboard"
        >
          <span>smart_toy</span>
          <h2>請透過 AI 查詢以顯示組件</h2>
        </div>
      </div>
      <MapContainer />
      <MoreInfo />
      <ReportIssue />
    </div>
    <div class="aitour-chat">
      <ChatBox />
    </div>
  </div>
</template>

<style scoped lang="scss">
.aitour {
	display: flex;
	height: 100%;
	overflow: hidden;

	&-map {
		flex: 1;
		display: flex;
		margin: var(--font-m) var(--font-m);
		overflow: hidden;

		.hide-if-mobile {
			display: contents;
		}

		.map-charts {
			width: 360px;
			max-height: 100%;
			height: fit-content;
			display: grid;
			row-gap: var(--font-m);
			margin-right: var(--font-s);
			border-radius: 5px;
			overflow-y: scroll;

			@media (min-width: 1000px) {
				width: 370px;
			}

			@media (min-width: 2000px) {
				width: 400px;
			}

			&-nodashboard {
				width: 360px;
				height: 100%;
				display: flex;
				flex-direction: column;
				align-items: center;
				justify-content: center;
				margin-right: var(--font-s);

				@media (min-width: 1000px) {
					width: 370px;
				}

				@media (min-width: 2000px) {
					width: 400px;
				}

				span {
					margin-bottom: var(--font-ms);
					font-family: var(--font-icon);
					font-size: 2rem;
				}

				button {
					color: var(--color-highlight);
				}

				div {
					width: 2rem;
					height: 2rem;
					border-radius: 50%;
					border: solid 4px var(--color-border);
					border-top: solid 4px var(--color-highlight);
					animation: spin 0.7s ease-in-out infinite;
				}
			}
		}
	}

	&-chat {
		width: 400px;
		height: 100%;
		display: flex;
		flex-direction: column;
		border-left: 1px solid var(--color-border);
		overflow: hidden;

		:deep(.chat-widget) {
			width: 100%;
			height: 100%;
			border-radius: 0;
			border: none;
		}
	}
}

@keyframes spin {
	to {
		transform: rotate(360deg);
	}
}
</style>
