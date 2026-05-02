import { onUnmounted } from 'vue'

const handlers = new Map()

export function emitAgentEvent(eventName, payload) {
	handlers.get(eventName)?.forEach((fn) => fn(payload))
}

export function useAgentEvent() {
	const registered = []

	const onEvent = (eventName, handler) => {
		if (!handlers.has(eventName)) handlers.set(eventName, new Set())
		handlers.get(eventName).add(handler)
		registered.push({ eventName, handler })
	}

	onUnmounted(() => {
		registered.forEach(({ eventName, handler }) => {
			handlers.get(eventName)?.delete(handler)
			if (handlers.get(eventName)?.size === 0) handlers.delete(eventName)
		})
	})

	return { onEvent }
}
