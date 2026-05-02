import { ref, watch } from "vue";
import { defineStore } from "pinia";
import { emitAgentEvent } from "../composables/useAgentEvent";
import { fetchComponentData } from "../assets/utilityFunctions/componentDataFetcher";
import { useMapStore } from "./mapStore";

async function* fetchSSE(body) {
	const response = await fetch(`${import.meta.env.VITE_API_URL}/chat`, {
		method: "POST",
		body: JSON.stringify(body),
		headers: {
			"Content-Type": "application/json",
		},
	});

	if (!response.ok) {
		throw new Error(
			`SSE 請求失敗：${response.status} ${response.statusText}`,
		);
	}

	const reader = response.body.getReader();
	const decoder = new TextDecoder();
	let buffer = "";
	let currentEvent = null;
	let currentData = null;

	while (true) {
		const { done, value } = await reader.read();
		if (done) break;

		buffer += decoder.decode(value, { stream: true });
		const lines = buffer.split("\n");
		buffer = lines.pop();

		for (const line of lines) {
			if (line.startsWith("event:")) {
				currentEvent = line.slice(6).trim();
			} else if (line.startsWith("data:")) {
				const dataStr = line.slice(5).trim();
				try {
					currentData = JSON.parse(dataStr);
				} catch {
					currentData = dataStr;
				}
			} else if (line.trim() === "") {
				if (currentEvent !== null) {
					yield { event: currentEvent, data: currentData };
					currentEvent = null;
					currentData = null;
				}
			}
		}
	}
}

export const useChatStore = defineStore("chat", () => {
	const defaultChatData = [
		{
			id: 1,
			role: "bot",
			isDefault: true,
			content:
				"你好，我是小儀！\n\n 你可以問我跟交通相關的問題，例如： \n\n • 「幫我規劃從台北車站到 101 的路線」 \n • 「附近哪裡有 YouBike 站？」 \n • 「中山區的即時交通狀況」 \n\n 也可以在地圖上雙擊新增最多 6 個地點，我會根據這些位置互動式操作圖資並導覽說明。\n\n 直接輸入你的問題，我會即時為你解答！",
			componentDatas: [],
		},
	];

	const chatData = ref([...defaultChatData]);
	const messageHistory = ref([]);
	const sessionId = ref(null);
	const attachments = ref([]);

	// frontend_action queue — components watch and dispatch
	const frontendActions = ref([]);

	watch(sessionId, (v) => {
		if (v) sessionStorage.setItem("chatSessionId", v);
		else sessionStorage.removeItem("chatSessionId");
	});

	watch(
		chatData,
		(v) => {
			const userBotMessages = v.filter((item) => !item.isDefault);
			sessionStorage.setItem("chatData", JSON.stringify(userBotMessages));
		},
		{ deep: true },
	);

	watch(
		messageHistory,
		(v) => sessionStorage.setItem("messageHistory", JSON.stringify(v)),
		{ deep: true },
	);

	const addChatData = (newChatData) => {
		chatData.value.push({
			id: chatData.value.length + 1,
			isDefault: false,
			...newChatData,
		});
	};

	const addMessageHistory = (newMessageHistory) => {
		messageHistory.value.push(newMessageHistory);
	};

	const addQueryData = async (newChatData) => {
		addChatData(newChatData);
		addMessageHistory(newChatData);

		// 2. 建立 bot 串流訊息（loading 狀態）
		const botMsgId = chatData.value.length + 1;
		chatData.value.push({
			id: botMsgId,
			role: "bot",
			isDefault: false,
			loading: true,
			content: "",
			componentDatas: [],
		});

		try {
			const prefix = attachments.value
				.map((a, index) => {
					if (a.type === "location")
						return `[位置${index + 1}: ${a.lng.toFixed(6)}, ${a.lat.toFixed(6)}]`;
					if (a.type === "current-location")
						return `[用戶當前位置: ${a.lng.toFixed(6)}, ${a.lat.toFixed(6)}]`;
					return "";
				})
				.filter(Boolean)
				.join("\n");

			const prompt = prefix
				? `${prefix}\n${newChatData.content}`
				: newChatData.content;
			if (!prompt.trim()) return;

			const requestBody = { prompt };
			if (sessionId.value) requestBody.session_id = sessionId.value;

			const sseSource = fetchSSE(requestBody);

			const botMsg = chatData.value.find((m) => m.id === botMsgId);
			let accumulatedText = "";

			for await (const { event, data } of sseSource) {
				switch (event) {
					case "notice":
						// Session was reset — clear local history so UI reflects fresh state
						console.warn(
							`[chat] session notice: ${data.code} — ${data.message}`,
						);
						break;

					case "session":
						sessionId.value = data.session_id;
						break;

					case "text":
						botMsg.loading = false;
						accumulatedText += data.delta ?? "";
						botMsg.content = accumulatedText;
						break;

					case "tool_used":
						console.log(`[chat] tool_used: ${data.name}`);

						addMessageHistory({
							role: "bot",
							tool_used: data.name,
						});
						break;

					case "frontend_action":
						console.log(
							`[chat] frontend_action: ${data.action}, params: ${JSON.stringify(data.params)}`,
						);

						switch (data.action) {
							case "add_card_in_chat": {
								const componentData = await fetchComponentData(
									data.params.data.component,
								);
								botMsg.loading = false;
								botMsg.componentDatas.push(componentData);
								break;
							}
							case "goto":
							case "goto_coordinate": {
								const c = data.params.center ?? data.params;
								useMapStore().gotoCoordinate(
									c.lng,
									c.lat,
									data.params.zoom,
								);
								break;
							}
							case "zoom_to":
							case "zoom_to_coordinate": {
								const bbox = data.params.bbox;
								if (bbox?.sw && bbox?.ne) {
									useMapStore().fitBboxBounds(
										bbox.sw.lng,
										bbox.sw.lat,
										bbox.ne.lng,
										bbox.ne.lat,
									);
								} else {
									const c = data.params.center ?? data.params;
									useMapStore().zoomToCoordinate(
										c.lng,
										c.lat,
										data.params.radius_m ?? 150,
									);
								}
								break;
							}
							default:
								emitAgentEvent(data.action, data.params);
								break;
						}
						break;

					case "done":
						botMsg.loading = false;
						break;

					case "error":
						botMsg.loading = false;
						botMsg.content = `很抱歉，服務發生錯誤：${data.message ?? "請稍後再試。"}`;
						break;
				}
			}

			// 3. 將完整回覆加入多輪對話歷史
			if (accumulatedText) {
				messageHistory.value.push({
					role: "assistant",
					content: accumulatedText,
				});
			}

			if (!accumulatedText && botMsg && !botMsg.content) {
				botMsg.loading = false;
				botMsg.content = "很抱歉，AI 未返回有效回覆，請重試。";
			}
		} catch (error) {
			console.error("AI chat SSE error:", error);
			const botMsg = chatData.value.find((m) => m.id === botMsgId);
			if (botMsg) {
				botMsg.loading = false;
				botMsg.content = "很抱歉，服務發生錯誤，請稍後再試。";
			}
		}
	};

	return {
		chatData,
		messageHistory,
		frontendActions,
		attachments,
		addChatData,
		addQueryData,
	};
});
