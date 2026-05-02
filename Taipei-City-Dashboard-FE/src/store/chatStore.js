import { ref, watch } from "vue";
import { defineStore } from "pinia";
import { emitAgentEvent } from "../composables/useAgentEvent";
import { fetchComponentData } from "../assets/utilityFunctions/componentDataFetcher";
import { useMapStore } from "./mapStore";

const USE_MOCK = import.meta.env.VITE_MOCK_CHAT === "true";

const MOCK_RESPONSES = [
	{
		event: "session",
		data: { session_id: "mock-session-001", is_new: true, requested: null },
	},
	{ event: "text", data: { delta: "您好！" } },
	{ event: "text", data: { delta: "這是一個 " } },
	{ event: "text", data: { delta: "SSE 串流回覆測試。" } },
	{
		event: "tool_used",
		data: { name: "get_district_stats", args: { district: "信義區" } },
	},
	{
		event: "frontend_action",
		data: {
			action: "add_component",
			params: {
				id: 214,
				index: "dependency_aging",
				name: "扶養比及老化指數",
				chart_config: {
					index: "dependency_aging",
					color: ["#67baca", "#fbf3ac"],
					types: ["ColumnLineChart", "TimelineSeparateChart"],
					unit: "%",
				},
				history_config: null,
				map_config: [null],
				map_filter: null,
				time_from: "static",
				time_to: null,
				update_freq: null,
				update_freq_unit: "",
				source: "主計處",
				short_desc: "顯示雙北扶養比及老化指數時間數列統計資料",
				long_desc:
					"顯示雙北扶養比及老化指數時間數列統計資料。雙北政府主計處提供了扶養比和老化指數資料，詳細記錄了各年齡段人口比例的變化情況。這些資料有助於分析雙北人口結構的演變，評估青壯年人口對幼年和老年人口的扶養負擔，以及社會老化程度。透過這些統計資料，政策制定者和研究人員可以深入了解人口趨勢，為未來的社會福利和經濟發展規劃提供參考。",
				use_case:
					"使用於人口結構分析、社會福利規劃與經濟發展評估，雙北的扶養比與老化指數數據提供決策參考。政府機構可透過這些統計資料評估勞動力供給與社會扶養負擔，進而調整退休政策與醫療資源配置。企業可運用數據研判市場趨勢，規劃銀髮族產品與服務。學術研究則可透過時間序列分析，探討人口老化對經濟與社會的影響，為未來城市發展與人口政策提供科學依據。\r\n",
				links: [
					"https://data.taipei/dataset/detail?id=aafb15dc-5508-4091-bd48-a708e60f6698",
					"https://data.ntpc.gov.tw/datasets/8308ab58-62d1-424e-8314-24b65b7ab492",
				],
				contributors: ["doit", "ntpc"],
				updated_at: "2024-12-10T02:59:39.341Z",
				query_type: "time",
				city: "metrotaipei",
			},
		},
	},
	{
		event: "frontend_action",
		data: {
			action: "show_component",
			params: {
				component_id: 214,
				data: {
					chart: [
						{
							name: "扶養比",
							data: [
								{ x: "2013-01-01T08:00:00+08:00", y: 38 },
								{ x: "2014-01-01T08:00:00+08:00", y: 39 },
								{ x: "2015-01-01T08:00:00+08:00", y: 40 },
								{ x: "2016-01-01T08:00:00+08:00", y: 42 },
								{ x: "2017-01-01T08:00:00+08:00", y: 43 },
								{ x: "2018-01-01T08:00:00+08:00", y: 45 },
								{ x: "2019-01-01T08:00:00+08:00", y: 46 },
								{ x: "2020-01-01T08:00:00+08:00", y: 48 },
								{ x: "2021-01-01T08:00:00+08:00", y: 49 },
								{ x: "2022-01-01T08:00:00+08:00", y: 50 },
							],
						},
						{
							name: "老化指數",
							data: [
								{ x: "2013-01-01T08:00:00+08:00", y: 95 },
								{ x: "2014-01-01T08:00:00+08:00", y: 99 },
								{ x: "2015-01-01T08:00:00+08:00", y: 106 },
								{ x: "2016-01-01T08:00:00+08:00", y: 112 },
								{ x: "2017-01-01T08:00:00+08:00", y: 119 },
								{ x: "2018-01-01T08:00:00+08:00", y: 126 },
								{ x: "2019-01-01T08:00:00+08:00", y: 134 },
								{ x: "2020-01-01T08:00:00+08:00", y: 144 },
								{ x: "2021-01-01T08:00:00+08:00", y: 154 },
								{ x: "2022-01-01T08:00:00+08:00", y: 166 },
							],
						},
					],
					component: {
						id: 214,
						index: "dependency_aging",
						name: "扶養比及老化指數",
						chart_config: {
							index: "dependency_aging",
							color: ["#67baca", "#fbf3ac"],
							types: ["ColumnLineChart", "TimelineSeparateChart"],
							unit: "%",
						},
						history_config: null,
						map_config: [null],
						map_filter: null,
						time_from: "static",
						time_to: null,
						update_freq: null,
						update_freq_unit: "",
						source: "主計處",
						short_desc:
							"顯示臺北市扶養比及老化指數時間數列統計資料",
						long_desc:
							"顯示臺北市扶養比及老化指數時間數列統計資料。臺北市政府主計處提供了扶養比和老化指數資料，詳細記錄了各年齡段人口比例的變化情況。這些資料有助於分析臺北市人口結構的演變，評估青壯年人口對幼年和老年人口的扶養負擔，以及社會老化程度。透過這些統計資料，政策制定者和研究人員可以深入了解人口趨勢，為未來的社會福利和經濟發展規劃提供參考。",
						use_case:
							"使用於人口結構分析、社會福利規劃與經濟發展評估，臺北市的扶養比與老化指數數據提供決策參考。政府機構可透過這些統計資料評估勞動力供給與社會扶養負擔，進而調整退休政策與醫療資源配置。企業可運用數據研判市場趨勢，規劃銀髮族產品與服務。學術研究則可透過時間序列分析，探討人口老化對經濟與社會的影響，為未來城市發展與人口政策提供科學依據。\r\n",
						links: [
							"https://data.taipei/dataset/detail?id=aafb15dc-5508-4091-bd48-a708e60f6698",
						],
						contributors: ["doit"],
						updated_at: "2025-02-25T01:43:21.031142Z",
						query_type: "time",
						city: "taipei",
					},
					query_type: "time",
					status: "success",
				},
			},
		},
	},
	{ event: "text", data: { delta: "\n\n如有問題請繼續詢問！" } },
	{
		event: "done",
		data: { session_id: "mock-session-001", message_count: 1 },
	},
];

async function* mockSSEGenerator() {
	for (const item of MOCK_RESPONSES) {
		await new Promise((r) => setTimeout(r, 250));
		yield item;
	}
}

/**
 * 使用 fetch 發出 POST 請求並以 async generator 方式逐一 yield SSE 事件。
 * SSE 行格式：
 *   event: <type>
 *   data: <json>
 *   (空行觸發 dispatch)
 */
async function* fetchSSE(body, token) {
	const baseURL = import.meta.env.VITE_CHAT_API_URL;
	const response = await fetch(`${baseURL}/chat`, {
		method: "POST",
		headers: {
			"Content-Type": "application/json",
			...(token ? { Authorization: `Bearer ${token}` } : {}),
		},
		body: JSON.stringify(body),
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

			const sseSource = USE_MOCK
				? mockSSEGenerator()
				: fetchSSE(requestBody, null);

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
							case "show_component":
								const componentData = await fetchComponentData(
									data.params.data.component,
								);
								botMsg.loading = false;
								botMsg.componentDatas.push(componentData);
								break;
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
									const c =
										data.params.center ?? data.params;
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

	const clearSession = async () => {
		if (!sessionId.value) return;
		try {
			const http = (await import("../router/axios")).default;
			await http.delete(`/api/dev/chat/session/${sessionId.value}`);
		} catch (error) {
			console.error("clearSession error:", error);
		} finally {
			sessionId.value = null;
			chatData.value = [...defaultChatData];
			messageHistory.value = [];
			sessionStorage.removeItem("chatData");
			sessionStorage.removeItem("messageHistory");
		}
	};

	const consumeFrontendAction = (id) => {
		frontendActions.value = frontendActions.value.filter(
			(a) => a.id !== id,
		);
	};

	const saveChatLog = async (question, answer) => {
		try {
			const formData = new FormData();
			const d = new Date();
			const todayId =
				d.getFullYear() +
				String(d.getMonth() + 1).padStart(2, "0") +
				String(d.getDate()).padStart(2, "0");

			formData.append("session", "session_" + todayId);
			formData.append("question", question);
			formData.append("answer", JSON.stringify(answer));

			const http = (await import("../router/axios")).default;
			await http.post("/chatlog/", formData, {
				headers: { "Content-Type": "multipart/form-data" },
			});
		} catch (error) {
			console.error("saveChatLog error:", error);
		}
	};

	return {
		chatData,
		messageHistory,
		frontendActions,
		attachments,
		addChatData,
		addQueryData,
		clearSession,
		saveChatLog,
		consumeFrontendAction,
	};
});
