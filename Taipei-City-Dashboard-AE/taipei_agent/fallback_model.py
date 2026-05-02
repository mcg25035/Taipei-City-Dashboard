"""pydantic_ai model wrappers that recover Llama tool-call text-leaks.

`TWCCFallbackChatModel` extends `OpenAIChatModel` to detect responses where
the inference server returned tool calls inside `message.content` as text
(e.g. `tool<function=foo>{...}</function>->>>|separate|`) and converts them
to proper structured tool calls before pydantic_ai consumes the response.

Two paths are covered:

* Non-streamed: `_process_response` mutates the OpenAI `ChatCompletion`
  in-place to populate `tool_calls` and clear leaked content.
* Streamed: `TWCCFallbackStreamedResponse` peeks the first content-bearing
  chunk, switches to buffer mode if a leak signature is detected, and emits
  synthetic `ToolCallPart` events at end of stream.

Detection logic lives in `tools.tool_call_fallback.extract_tool_calls`.
"""

from __future__ import annotations

import json
from collections.abc import AsyncIterator
from dataclasses import dataclass
from typing import TYPE_CHECKING

from pydantic_ai.messages import ModelResponseStreamEvent
from pydantic_ai.models.openai import OpenAIChatModel, OpenAIStreamedResponse

from tools.tool_call_fallback import extract_tool_calls

if TYPE_CHECKING:
    from openai.types.chat import ChatCompletion, ChatCompletionChunk


_LEAK_SIGNAL_TOKENS: tuple[str, ...] = ("<function=", "tool<")


def _looks_like_leak(text: str) -> bool:
    stripped = text.lstrip()
    return any(token in stripped for token in _LEAK_SIGNAL_TOKENS)


def _inject_tool_calls_into_completion(response: "ChatCompletion") -> bool:
    """Mutate `response` in place if leaked tool calls are detected.

    Returns True if a substitution happened, False otherwise.
    """
    if not response.choices:
        return False

    msg = response.choices[0].message
    if msg.tool_calls or not msg.content:
        return False

    parsed, cleaned = extract_tool_calls(msg.content)
    if not parsed:
        return False

    from openai.types.chat.chat_completion_message_function_tool_call import (
        ChatCompletionMessageFunctionToolCall,
        Function,
    )

    msg.tool_calls = [
        ChatCompletionMessageFunctionToolCall(
            id=f"fallback_{i}",
            type="function",
            function=Function(name=p.name, arguments=json.dumps(p.arguments)),
        )
        for i, p in enumerate(parsed)
    ]
    msg.content = cleaned or None
    response.choices[0].finish_reason = "tool_calls"
    return True


@dataclass
class TWCCFallbackStreamedResponse(OpenAIStreamedResponse):
    """Streamed response that buffers + parses leaked tool calls."""

    async def _get_event_iterator(self) -> AsyncIterator[ModelResponseStreamEvent]:
        original = self._response

        peeked: list[ChatCompletionChunk] = []
        is_leak = False
        async for chunk in original:
            peeked.append(chunk)
            content = (
                chunk.choices[0].delta.content
                if chunk.choices and chunk.choices[0].delta
                else None
            )
            if content:
                is_leak = _looks_like_leak(content)
                break

        if not is_leak:
            async def replay() -> AsyncIterator[ChatCompletionChunk]:
                for c in peeked:
                    yield c
                async for c in original:
                    yield c

            self._response = replay()
            async for event in super()._get_event_iterator():
                yield event
            return

        buffer = "".join(
            c.choices[0].delta.content or ""
            for c in peeked
            if c.choices and c.choices[0].delta and c.choices[0].delta.content
        )
        async for chunk in original:
            if chunk.choices and chunk.choices[0].delta and chunk.choices[0].delta.content:
                buffer += chunk.choices[0].delta.content
            chunk_usage = self._map_usage(chunk)
            self._usage += chunk_usage
            if chunk.id:
                self.provider_response_id = chunk.id
            if chunk.model:
                self._model_name = chunk.model

        calls, cleaned = extract_tool_calls(buffer)

        if not calls:
            for event in self._parts_manager.handle_text_delta(
                vendor_part_id="content",
                content=buffer,
                thinking_tags=self._model_profile.thinking_tags,
                ignore_leading_whitespace=self._model_profile.ignore_streamed_leading_whitespace,
            ):
                yield event
            self.finish_reason = self._map_finish_reason("stop")
            return

        if cleaned:
            for event in self._parts_manager.handle_text_delta(
                vendor_part_id="content",
                content=cleaned,
                thinking_tags=self._model_profile.thinking_tags,
                ignore_leading_whitespace=self._model_profile.ignore_streamed_leading_whitespace,
            ):
                yield event

        for i, call in enumerate(calls):
            tool_call_id = f"fallback_{i}"
            yield self._parts_manager.handle_tool_call_part(
                vendor_part_id=tool_call_id,
                tool_name=call.name,
                args=json.dumps(call.arguments),
                tool_call_id=tool_call_id,
            )

        self.finish_reason = self._map_finish_reason("tool_calls")


class TWCCFallbackChatModel(OpenAIChatModel):
    """OpenAIChatModel variant that recovers leaked tool calls."""

    def _process_response(self, response):  # type: ignore[override]
        from openai.types import chat

        if isinstance(response, chat.ChatCompletion):
            _inject_tool_calls_into_completion(response)
        return super()._process_response(response)

    @property
    def _streamed_response_cls(self) -> type[OpenAIStreamedResponse]:
        return TWCCFallbackStreamedResponse
