import { ref, watch } from 'vue'
import { defineStore } from 'pinia'


// ---- DEV MOCK ----
const USE_MOCK = import.meta.env.VITE_MOCK_CHAT === 'true';

const MOCK_RESPONSES = [
	{ event: 'session', data: { session_id: 'mock-session-001', is_new: true, requested: null } },
	{ event: 'text', data: { delta: '您好！' } },
	{ event: 'text', data: { delta: '這是一個 ' } },
	{ event: 'text', data: { delta: 'SSE 串流回覆測試。' } },
	{ event: 'tool_used', data: { name: 'get_district_stats', args: { district: '信義區' } } },
	{ event: 'text', data: { delta: '\n\n如有問題請繼續詢問！' } },
	{ event: 'done', data: { session_id: 'mock-session-001', message_count: 1 } },
];

async function* mockSSEGenerator() {
	for (const item of MOCK_RESPONSES) {
		await new Promise((r) => setTimeout(r, 250));
		yield item;
	}
}
// ------------------

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
		method: 'POST',
		headers: {
			'Content-Type': 'application/json',
			...(token ? { Authorization: `Bearer ${token}` } : {}),
		},
		body: JSON.stringify(body),
	});

	if (!response.ok) {
		throw new Error(`SSE 請求失敗：${response.status} ${response.statusText}`);
	}

	const reader = response.body.getReader();
	const decoder = new TextDecoder();
	let buffer = '';
	let currentEvent = null;
	let currentData = null;

	while (true) {
		const { done, value } = await reader.read();
		if (done) break;

		buffer += decoder.decode(value, { stream: true });
		const lines = buffer.split('\n');
		buffer = lines.pop();

		for (const line of lines) {
			if (line.startsWith('event:')) {
				currentEvent = line.slice(6).trim();
			} else if (line.startsWith('data:')) {
				const dataStr = line.slice(5).trim();
				try {
					currentData = JSON.parse(dataStr);
				} catch {
					currentData = dataStr;
				}
			} else if (line.trim() === '') {
				if (currentEvent !== null) {
					yield { event: currentEvent, data: currentData };
					currentEvent = null;
					currentData = null;
				}
			}
		}
	}
}

export const useChatStore = defineStore('chat', () => {
	const defaultChatData = [
		{
			id: 1,
			role: 'bot',
			isDefault: true,
			content:
				'您好，我是【臺北城市儀表板】小幫手，很高興為您服務！\n 您可以： \n\n • 點擊左側既有的儀表板主題，快速查看各主題內容 \n • 輸入您感興趣的主題描述，我會自動為您組建最適合的儀表板 \n\n 如果有想了解的內容，歡迎直接告訴我，我會盡力協助！\n\n 📩 聯絡信箱：tuic@gov.taipei \n 🏢 臺北大數據中心 \n\n',
		},
	];

	const savedChatData = JSON.parse(sessionStorage.getItem('chatData')) || [];
	const savedMessageHistory = JSON.parse(sessionStorage.getItem('messageHistory')) || [];
	// First call must omit session_id; server provides UUID via `session` event
	const savedSessionId = sessionStorage.getItem('chatSessionId') || null;

	const chatData = ref([...defaultChatData, ...savedChatData]);
	const messageHistory = ref(savedMessageHistory);
	const sessionId = ref(savedSessionId);

	// frontend_action queue — components watch and dispatch
	const frontendActions = ref([]);

	// AI tool call trace (localStorage, cross-session)
	const _rawAIToolCalls = JSON.parse(localStorage.getItem('aiToolCalls'));
	const aiToolCalls = ref(Array.isArray(_rawAIToolCalls) ? _rawAIToolCalls : []);

	watch(aiToolCalls, (v) => localStorage.setItem('aiToolCalls', JSON.stringify(v)), { deep: true });

	watch(sessionId, (v) => {
		if (v) sessionStorage.setItem('chatSessionId', v);
		else sessionStorage.removeItem('chatSessionId');
	});

	watch(
		chatData,
		(v) => {
			const userBotMessages = v.filter((item) => !item.isDefault);
			sessionStorage.setItem('chatData', JSON.stringify(userBotMessages));
		},
		{ deep: true }
	);

	watch(
		messageHistory,
		(v) => sessionStorage.setItem('messageHistory', JSON.stringify(v)),
		{ deep: true }
	);

	const _clearLocalHistory = () => {
		chatData.value = [...defaultChatData];
		messageHistory.value = [];
		sessionStorage.removeItem('chatData');
		sessionStorage.removeItem('messageHistory');
	};

	const addChatData = (newChatData) => {
		chatData.value.push({ id: chatData.value.length + 1, isDefault: false, ...newChatData });
	};

	const addQueryData = async (newChatData) => {
		// 1. 顯示使用者訊息
		chatData.value.push({ id: chatData.value.length + 1, isDefault: false, ...newChatData });
		messageHistory.value.push({ role: 'user', content: newChatData.content });

		// 2. 建立 bot 串流訊息（loading 狀態）
		const botMsgId = chatData.value.length + 1;
		chatData.value.push({
			id: botMsgId,
			role: 'bot',
			isDefault: false,
			loading: true,
			content: '',
		});

		try {
			const requestBody = { prompt: newChatData.content };
			if (sessionId.value) requestBody.session_id = sessionId.value;

			const sseSource = USE_MOCK
				? mockSSEGenerator()
				: fetchSSE(requestBody, null);

			const botMsg = chatData.value.find((m) => m.id === botMsgId);
			let accumulatedText = '';

			for await (const { event, data } of sseSource) {
				switch (event) {
					case 'notice':
						// Session was reset — clear local history so UI reflects fresh state
						console.warn(`[chat] session notice: ${data.code} — ${data.message}`);
						_clearLocalHistory();
						break;

					case 'session':
						sessionId.value = data.session_id;
						break;

					case 'text':
						botMsg.loading = false;
						accumulatedText += data.delta ?? '';
						botMsg.content = accumulatedText;
						break;

					case 'tool_used':
						console.log('tool_used', data);
						aiToolCalls.value.push(data);
						break;

					case 'frontend_action':
						console.log('frontend_action', data);
						// Deduplicate by id before pushing
						if (!frontendActions.value.some((a) => a.id === data.id)) {
							frontendActions.value.push(data);
						}
						break;

					case 'done':
						botMsg.loading = false;
						break;

					case 'error':
						botMsg.loading = false;
						botMsg.content = `很抱歉，服務發生錯誤：${data.message ?? '請稍後再試。'}`;
						break;
				}
			}

			// 3. 將完整回覆加入多輪對話歷史
			if (accumulatedText) {
				messageHistory.value.push({ role: 'assistant', content: accumulatedText });
			}

			if (!accumulatedText && botMsg && !botMsg.content) {
				botMsg.loading = false;
				botMsg.content = '很抱歉，AI 未返回有效回覆，請重試。';
			}
		} catch (error) {
			console.error('AI chat SSE error:', error);
			const botMsg = chatData.value.find((m) => m.id === botMsgId);
			if (botMsg) {
				botMsg.loading = false;
				botMsg.content = '很抱歉，服務發生錯誤，請稍後再試。';
			}
		}
	};

	const clearSession = async () => {
		if (!sessionId.value) return;
		try {
			const http = (await import('../router/axios')).default;
			await http.delete(`/api/dev/chat/session/${sessionId.value}`);
		} catch (error) {
			console.error('clearSession error:', error);
		} finally {
			sessionId.value = null;
			_clearLocalHistory();
		}
	};

	const addAIToolCall = (toolCall) => {
		aiToolCalls.value.push(toolCall);
	};

	const clearAIToolCalls = () => {
		aiToolCalls.value = [];
	};

	const consumeFrontendAction = (id) => {
		frontendActions.value = frontendActions.value.filter((a) => a.id !== id);
	};

	const saveChatLog = async (question, answer) => {
		try {
			const formData = new FormData();
			const d = new Date();
			const todayId =
				d.getFullYear() +
				String(d.getMonth() + 1).padStart(2, '0') +
				String(d.getDate()).padStart(2, '0');

			formData.append('session', 'session_' + todayId);
			formData.append('question', question);
			formData.append('answer', JSON.stringify(answer));

			const http = (await import('../router/axios')).default;
			await http.post('/chatlog/', formData, {
				headers: { 'Content-Type': 'multipart/form-data' },
			});
		} catch (error) {
			console.error('saveChatLog error:', error);
		}
	};

	return {
		chatData,
		messageHistory,
		aiToolCalls,
		frontendActions,
		addChatData,
		addQueryData,
		clearSession,
		saveChatLog,
		addAIToolCall,
		clearAIToolCalls,
		consumeFrontendAction,
	};
});
