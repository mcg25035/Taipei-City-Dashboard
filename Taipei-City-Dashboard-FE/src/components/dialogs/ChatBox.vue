<script setup>
import { ref, computed, watch, nextTick } from "vue";
import { storeToRefs } from "pinia";
import SendIcon from "../icons/SendIcon.vue";
import BotLogo from "../icons/BotLogo.vue";
import UserLogo from "../icons/UserLogo.vue";
import DashboardComponent from "../../dashboardComponent/DashboardComponent.vue";

import { useChatStore } from "../../store/chatStore";
import { useContentStore } from "../../store/contentStore";
import { useDialogStore } from "../../store/dialogStore";

const chatStore = useChatStore();
const contentStore = useContentStore();
const dialogStore = useDialogStore();
const { addQueryData, resetSession } = chatStore;
const { chatData, attachments } = storeToRefs(chatStore);

const currentLocationIndex = computed(() =>
	attachments.value.findIndex((a) => a.type === "current_location"),
);
const hasCurrentLocation = computed(() => currentLocationIndex.value !== -1);

function removeAttachment(idx) {
	const item = attachments.value[idx];
	attachments.value.splice(idx, 1);
	item?.marker?.remove();
}

function clearAllAttachmentMarkers() {
	attachments.value.forEach((a) => a.marker?.remove());
}

function toggleCurrentLocation() {
	if (hasCurrentLocation.value) {
		removeAttachment(currentLocationIndex.value);
		return;
	}
	if (!navigator.geolocation) {
		dialogStore?.showNotification?.("error", "瀏覽器不支援定位功能");
		return;
	}
	navigator.geolocation.getCurrentPosition(
		(position) => {
			attachments.value.push({
				type: "current_location",
				lng: position.coords.longitude,
				lat: position.coords.latitude,
			});
		},
		(error) => {
			console.error(error.message);
			dialogStore?.showNotification?.("error", "無法取得定位授權");
		},
	);
}

const userMessage = ref("");
const chatAreaRef = ref(null);
const isStickyOpen = ref(false);

const sendBtnHandler = () => {
	if (!userMessage.value.trim()) {
		return;
	}

	const sanitized = attachments.value.map(({ marker, ...rest }) => rest);
	addQueryData({
		role: "user",
		content: userMessage.value,
		attachments: sanitized,
	});

	userMessage.value = "";
	clearAllAttachmentMarkers();
	attachments.value = [];
};

const toggleSticky = () => {
	isStickyOpen.value = !isStickyOpen.value;
};

const startNewChat = () => {
	clearAllAttachmentMarkers();
	attachments.value = [];
	resetSession();
};

watch(
	() => chatData.value.length,
	async () => {
		await nextTick();
		const chat = chatAreaRef.value;
		if (!chat) return;
		chat.scrollTop = chat.scrollHeight - chat.clientHeight;
	},
	{ deep: true },
);
</script>

<template>
	<div class="chat-widget">
		<!-- 標題 -->
		<div class="header">
			<h3>臺北城市儀表板小幫手</h3>
			<button class="new-btn" @click="startNewChat">+ New</button>
		</div>

		<!-- 聊天區 -->
		<div ref="chatAreaRef" class="chat-area scrollbar-custom">
			<!-- 置頂訊息 -->
			<div class="chat-message sticky-message">
				<div class="sticky-header" @click="toggleSticky">
					<span>置頂公告：小幫手使用須知</span>
					<button class="toggle-btn">
						{{ isStickyOpen ? "-" : "+" }}
					</button>
				</div>
				<div v-show="isStickyOpen" class="sticky-body">
					<span
						>小幫手會依據您輸入的內容，自動檢索本站臺的組件資料庫，並回傳相似度較高的組件清單，協助您快速找到符合需求的元件或資訊。<br /><br />
						目前小幫手僅提供組件比對與分析服務，不支援一般聊天功能。如造成不便，敬請見諒！</span
					>
				</div>
			</div>
			<div v-for="chat in chatData" :key="chat.id" class="message">
				<!-- 機器人訊息 -->
				<div v-if="chat.role === 'bot'" class="bot">
					<div class="avatar">
						<BotLogo />
					</div>
					<div class="content">
						<div v-if="chat.tool_used" class="tool-used-badge">
							🔧 已使用工具分析
						</div>
						<div
							v-if="chat.loading || chat.content"
							class="message--bubble"
						>
							<div v-if="chat.loading" class="message--loading">
								<span class="dot" />
								<span class="dot" />
								<span class="dot" />
							</div>
							<p v-else>{{ chat.content }}</p>
						</div>
						<DashboardComponent
							v-for="componentData in chat.componentDatas"
							:key="`component-${componentData.index}-${componentData.city}`"
							:config="componentData"
							mode="default"
							:active-city="componentData.city"
							:select-btn="true"
							:select-btn-disabled="
								contentStore.cityManager.getSelectList(
									componentData.city,
								).length === 1
							"
							:select-btn-list="
								contentStore.cityManager.getSelectList(
									componentData.city,
								)
							"
							:city-tag="
								contentStore.cityManager.getTagList(
									componentData.city,
								)
							"
							@change-city="
								(city) => {
									const selectedData =
										contentStore.aiSearchedComponents.find(
											(data) =>
												data.index ===
													componentData.index &&
												data.city === city,
										);
									if (selectedData) {
										componentData = selectedData;
									}
								}
							"
						/>
					</div>
				</div>
				<!-- 使用者訊息 -->
				<div v-else class="user">
					<div class="avatar">
						<UserLogo />
					</div>
					<div
						v-if="chat.content || chat.attachments?.length"
						class="content"
					>
						<div v-if="chat.content" class="message--bubble">
							<div
								v-if="chat.attachments?.length"
								class="message--attachments"
							>
								<div
									v-for="(att, idx) in chat.attachments"
									:key="idx"
									class="attachment-chip attachment-chip--sent"
								>
									<template v-if="att.type === 'location'">
										<span class="attachment-icon"
											>location_on</span
										>
										<span class="attachment-text"
											>{{ att.lng.toFixed(6) }},
											{{ att.lat.toFixed(6) }}</span
										>
									</template>
									<template
										v-else-if="
											att.type === 'current_location'
										"
									>
										<span class="attachment-icon"
											>my_location</span
										>
										<span class="attachment-text"
											>已分享當前位置給小儀</span
										>
									</template>
								</div>
							</div>
							<p>{{ chat.content }}</p>
						</div>
					</div>
				</div>
			</div>
		</div>

		<!-- 附件區 -->
		<div v-if="attachments.length" class="attachment-area">
			<div
				v-for="(attachment, idx) in attachments"
				:key="idx"
				class="attachment-chip"
			>
				<template v-if="attachment.type === 'location'">
					<span class="attachment-icon">location_on</span>
					<span class="attachment-text"
						>{{ attachment.lng.toFixed(6) }},
						{{ attachment.lat.toFixed(6) }}</span
					>
				</template>
				<template v-else-if="attachment.type === 'current_location'">
					<span class="attachment-icon">my_location</span>
					<span class="attachment-text">已分享當前位置給小儀</span>
				</template>
				<button class="attachment-clear" @click="removeAttachment(idx)">
					×
				</button>
			</div>
		</div>

		<!-- 輸入區 -->
		<div class="input-area">
			<input
				v-model="userMessage"
				type="text"
				placeholder="輸入訊息..."
				@keyup.enter="sendBtnHandler()"
			/>
			<button
				class="location-btn"
				:class="{ 'location-btn--active': hasCurrentLocation }"
				:title="hasCurrentLocation ? '取消分享位置' : '分享當前位置'"
				@click="toggleCurrentLocation()"
			>
				<span class="location-btn-icon">my_location</span>
			</button>
			<button @click="sendBtnHandler()">
				<SendIcon />
			</button>
		</div>
	</div>
</template>

<style lang="scss" scoped>
/* === 變數設定 === */
$bg-dark: #090909;
$panel-bg: #494b4e;
$card-bg: #282a2c;
$border-color: #888787;
$input-bg: #d9d9d9;
$white: #ffffff;
$scroll-thumb-hover: #ababab;
$radius-10: 10px;
$radius-15: 15px;
$radius-20: 20px;

/* === Scrollbar === */
.scrollbar-x-hide {
	scrollbar-width: none;

	&::-webkit-scrollbar {
		display: none;
	}
}

.scrollbar-custom {
	&::-webkit-scrollbar {
		width: 2px;
		background: transparent;
	}

	&::-webkit-scrollbar-thumb {
		background: $white;
		border-radius: 8px;
	}

	&::-webkit-scrollbar-thumb:hover {
		background: $scroll-thumb-hover;
	}
}

/* === 主要樣式 === */
.chat-widget {
	width: 100%;
	border-radius: $radius-20;
	overflow: hidden;
	background: $bg-dark;
	border: 1px solid $border-color;
	display: flex;
	flex-direction: column;

	.header {
		padding: 1rem;
		background: $panel-bg;
		border-bottom: 3px solid $border-color;
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 0.5rem;

		h3 {
			font-size: 18px;
			font-weight: 700;
			color: $white;
			margin: 0;
		}

		.new-btn {
			background: transparent;
			color: $white;
			border: 1px solid $white;
			border-radius: 6px;
			padding: 4px 10px;
			font-size: 13px;
			cursor: pointer;
			white-space: nowrap;

			&:hover {
				filter: brightness(0.7);
			}
		}
	}

	.chat-area {
		flex: 1;
		margin: 0.25rem;
		padding: 0.75rem;
		overflow-y: auto;
		background: $bg-dark;

		.chat-message {
			padding: 4px 10px;
			margin: 0px 8px;
			border-radius: 8px;
			background-color: $bg-dark;
		}

		// 置頂訊息
		.sticky-message {
			border: 1px solid #ffffff;
			position: sticky;
			top: 0;
			z-index: 10;

			.sticky-header {
				display: flex;
				font-weight: bold;
				justify-content: space-between;
				align-items: center;
				cursor: pointer;
				padding: 8px 12px;
			}

			.sticky-body {
				padding: 8px 12px;
				font-weight: 400;
				font-size: 14px;
			}

			.toggle-btn {
				background: none;
				border: none;
				font-size: 14px;
				cursor: pointer;
				color: #ffffff;
			}
		}

		.message {
			padding: 8px;

			.bot,
			.user {
				display: flex;
				gap: 0.5rem;
				align-items: flex-start;

				&.user {
					flex-direction: row-reverse;
				}

				&.user .content {
					align-items: flex-end;
				}

				.avatar {
					width: 40px;
					height: 40px;
					display: flex;
					align-items: center;
					justify-content: center;
					flex-shrink: 0;

					svg {
						width: 100%;
						height: auto;
					}
				}

				.content {
					flex: 1;
					display: flex;
					flex-direction: column;
					gap: 0.5rem;

					.relation-area {
						width: 100%;
						display: flex;
						align-items: center;
						margin-top: 8px;
						margin-bottom: 8px;

						.relation-table {
							min-width: max-content;
							font-size: 13px;
						}

						.relation-table th,
						.relation-table td {
							border: 1px solid #ccc;
							text-align: left;
							padding: 0px 8px;
							line-height: 1.1;
							vertical-align: middle;
						}

						.relation-table td {
							height: 2.5rem;
						}

						.relation-table th {
							font-weight: bold;
							text-align: center;
						}
					}

					.message--attachments {
						display: flex;
						flex-wrap: wrap;
						gap: 0.5rem;
						padding: 0.5rem;
						padding-bottom: 0;

						.attachment-chip--sent {
							background: #2a2a2a;
							border: 1px solid #555;
							border-radius: 6px;
							display: flex;
							align-items: center;
							gap: 4px;
							padding: 3px 10px 3px 8px;
							font-size: 12px;
							color: #ccc;

							.attachment-icon {
								font-family: var(--font-icon);
								font-size: 14px;
							}
						}
					}

					.message--loading {
						display: flex;
						align-items: center;
						gap: 6px;
						padding: 12px 16px;

						.dot {
							width: 8px;
							height: 8px;
							border-radius: 50%;
							background: $white;
							animation: dot-blink 1.2s infinite;

							&:nth-child(2) {
								animation-delay: 0.2s;
							}
							&:nth-child(3) {
								animation-delay: 0.4s;
							}
						}
					}

					@keyframes dot-blink {
						0%,
						80%,
						100% {
							opacity: 0.2;
							transform: scale(0.8);
						}
						40% {
							opacity: 1;
							transform: scale(1);
						}
					}

					.tool-used-badge {
						font-size: 11px;
						color: #aaa;
						padding: 0 16px 8px;
					}

					.message--bubble {
						max-width: 100%;
						width: fit-content;
						display: flex;
						flex-direction: column;
						border: 1px solid $white;
						border-radius: $radius-10;
						background: $card-bg;

						p {
							color: $white;
							white-space: pre-line;
							margin: 0;
							padding: 0.5rem 1rem;
							font-size: 16px;
							word-break: break-word;
						}
					}

					.message--button {
						display: flex;
						gap: 0.5rem;
						overflow-x: auto;

						button {
							flex-shrink: 0;
							background: $panel-bg;
							color: $white;
							font-size: 14px;
							padding: 0.5rem 1rem;
							border-radius: $radius-15;
							border: none;
							cursor: pointer;
							white-space: nowrap;

							&:hover {
								filter: brightness(0.5);
							}
						}
					}
				}
			}
		}
	}

	.ai-dashboard-bar {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		padding: 1.5rem 1rem;
		padding-bottom: 0.5rem;
		background: $panel-bg;
		border-top: 1px solid $border-color;

		.ai-dashboard-btn {
			flex: 1;
			display: flex;
			align-items: center;
			gap: 0.4rem;
			background: #0d1117;
			color: #4fc1e9;
			border: 1px solid #4fc1e9;
			border-radius: 20px;
			padding: 0.4rem 1rem;
			font-size: 13px;
			cursor: pointer;
			transition: opacity 0.2s;

			span {
				font-family: var(--font-icon);
				font-size: 16px;
			}

			&:hover {
				opacity: 0.8;
			}
		}

		.ai-dashboard-clear {
			display: flex;
			align-items: center;
			justify-content: center;
			background: transparent;
			border: none;
			color: $border-color;
			cursor: pointer;
			padding: 4px;
			transition: color 0.2s;

			span {
				font-family: var(--font-icon);
				font-size: 18px;
			}

			&:hover {
				color: $white;
			}
		}
	}

	.attachment-area {
		display: flex;
		flex-wrap: wrap;
		gap: 0.5rem;
		padding: 1rem;
		padding-bottom: 0;
		background: $panel-bg;

		.attachment-chip {
			display: flex;
			align-items: center;
			gap: 4px;
			background: #2a2a2a;
			border: 1px solid #555;
			border-radius: 6px;
			padding: 3px 10px 3px 8px;
			font-size: 12px;
			color: #ccc;
			width: max-content;

			.attachment-text {
				white-space: nowrap;
				overflow: hidden;
				text-overflow: ellipsis;
			}

			.attachment-icon {
				font-family: var(--font-icon);
				font-size: 14px;
			}

			.attachment-clear {
				background: none;
				border: none;
				color: #ccc;
				cursor: pointer;
				font-size: 14px;
				line-height: 1;
				padding: 0 0 0 2px;

				&:hover {
					color: $white;
				}
			}
		}
	}

	.input-area {
		display: flex;
		align-items: center;
		justify-content: center;
		gap: 0.5rem;
		padding: 1rem;
		background: $panel-bg;

		input[type="text"] {
			background: $white;
			height: 35px;
			width: 100%;
			border-radius: 6px;
			padding: 0 1rem;
			border: none;
			outline: none;
			color: black;
		}

		button {
			height: 35px;
			display: flex;
			align-items: center;
			justify-content: center;
			background: transparent;
			border: none;
			cursor: pointer;

			&:hover {
				filter: brightness(0.5);
			}
		}

		.location-btn {
			width: 35px;
			height: 35px;
			border-radius: 50%;
			color: $white;
			flex-shrink: 0;

			&.location-btn--active {
				color: #4fc1e9;
			}

			.location-btn-icon {
				font-family: var(--font-icon);
				font-size: 22px;
				line-height: 1;
			}
		}
	}
}
</style>
