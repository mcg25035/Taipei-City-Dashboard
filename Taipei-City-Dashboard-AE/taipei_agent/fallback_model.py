"""pydantic_ai model wrappers that recover Llama tool-call text-leaks.

`TWCCFallbackChatModel` extends `OpenAIChatModel` to detect responses where
the inference server returned tool calls inside `message.content` as text
(e.g. `tool<function=foo>{...}</function>->>>|separate|`) and converts them
to proper structured tool calls before pydantic_ai consumes the response.

Two paths are covered:

* Non-streamed: `_process_response` mutates the OpenAI `ChatCompletion`
  in-place to populate `tool_calls` and clear leaked content.
* Streamed: `TWCCFallbackStreamedResponse` overrides only `_map_text_delta`
  with a stateful filter that intercepts `<function=...>...</function>`
  regions and emits `ToolCallPart` events instead of leaking them as text.
  Native `delta.tool_calls`, thinking deltas, usage, and finish_reason
  flow through the parent class unchanged.

Detection logic lives in `tools.tool_call_fallback.extract_tool_calls`.
"""

from __future__ import annotations

import json
import logging
from collections.abc import AsyncIterator, Iterable
from dataclasses import dataclass, field
from typing import TYPE_CHECKING

from openai.types import chat as _openai_chat
from openai.types.chat import chat_completion_chunk
from pydantic_ai.messages import ModelResponseStreamEvent
from pydantic_ai.models.openai import OpenAIChatModel, OpenAIStreamedResponse

from tools.tool_call_fallback import (
    TEMPLATE_ARTIFACT_PATTERNS,
    extract_tool_calls,
)

if TYPE_CHECKING:
    from openai.types.chat import ChatCompletion

logger = logging.getLogger(__name__)


_FUNC_OPEN = "<function="
_FUNC_CLOSE = "</function>"


def _strip_artifacts(text: str) -> str:
    out = text
    for pattern in TEMPLATE_ARTIFACT_PATTERNS:
        out = pattern.sub("", out)
    return out


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
    """Streamed response that filters leaked tool calls inside text deltas."""

    _filter_state: str = field(default="normal", init=False)
    _filter_pending: str = field(default="", init=False)
    _filter_capture: str = field(default="", init=False)
    _filter_seq: int = field(default=0, init=False)

    def _emit_text(self, text: str) -> Iterable[ModelResponseStreamEvent]:
        if not text:
            return ()
        return self._parts_manager.handle_text_delta(
            vendor_part_id="content",
            content=text,
            thinking_tags=self._model_profile.thinking_tags,
            ignore_leading_whitespace=self._model_profile.ignore_streamed_leading_whitespace,
        )

    def _emit_calls_from(self, raw: str, prefix: str) -> Iterable[ModelResponseStreamEvent]:
        calls, _ = extract_tool_calls(raw)
        if not calls:
            logger.warning(
                "tool_call_fallback: leak signal without parseable call; raw=%r",
                raw[:200],
            )
            return ()
        events = []
        for i, call in enumerate(calls):
            self._filter_seq += 1
            cid = f"fallback_{prefix}_{self._filter_seq}_{i}"
            events.append(
                self._parts_manager.handle_tool_call_part(
                    vendor_part_id=cid,
                    tool_name=call.name,
                    args=json.dumps(call.arguments),
                    tool_call_id=cid,
                )
            )
        return events

    def _map_text_delta(
        self, choice: chat_completion_chunk.Choice
    ) -> Iterable[ModelResponseStreamEvent]:
        content = choice.delta.content
        if not content:
            return
        yield from self._process_text_chunk(content)

    def _process_text_chunk(self, text: str) -> Iterable[ModelResponseStreamEvent]:
        while text:
            if self._filter_state == "capture":
                close_idx = text.find(_FUNC_CLOSE)
                if close_idx == -1:
                    self._filter_capture += text
                    return
                self._filter_capture += text[: close_idx + len(_FUNC_CLOSE)]
                text = text[close_idx + len(_FUNC_CLOSE):]
                yield from self._emit_calls_from(self._filter_capture, "mid")
                self._filter_capture = ""
                self._filter_state = "normal"
                continue

            buffer = self._filter_pending + text
            self._filter_pending = ""
            text = ""

            lt = buffer.find("<")
            if lt == -1:
                yield from self._emit_text(_strip_artifacts(buffer))
                continue

            yield from self._emit_text(_strip_artifacts(buffer[:lt]))
            rest = buffer[lt:]

            if rest.startswith(_FUNC_OPEN):
                self._filter_state = "capture"
                self._filter_capture = ""
                text = rest
            elif _FUNC_OPEN.startswith(rest):
                self._filter_pending = rest
            else:
                yield from self._emit_text("<")
                text = rest[1:]

    async def _get_event_iterator(self) -> AsyncIterator[ModelResponseStreamEvent]:
        async for event in super()._get_event_iterator():
            yield event

        if self._filter_state == "capture" and self._filter_capture:
            calls, cleaned = extract_tool_calls(self._filter_capture)
            if calls:
                for ev in self._emit_calls_from(self._filter_capture, "eof"):
                    yield ev
            elif cleaned:
                for ev in self._emit_text(cleaned):
                    yield ev
            self._filter_capture = ""
            self._filter_state = "normal"

        if self._filter_pending:
            calls, cleaned = extract_tool_calls(self._filter_pending)
            if calls:
                for ev in self._emit_calls_from(self._filter_pending, "pending"):
                    yield ev
            elif cleaned:
                for ev in self._emit_text(cleaned):
                    yield ev
            self._filter_pending = ""


class TWCCFallbackChatModel(OpenAIChatModel):
    """OpenAIChatModel variant that recovers leaked tool calls."""

    def _process_response(self, response):  # type: ignore[override]
        if isinstance(response, _openai_chat.ChatCompletion):
            _inject_tool_calls_into_completion(response)
        return super()._process_response(response)

    @property
    def _streamed_response_cls(self) -> type[OpenAIStreamedResponse]:
        return TWCCFallbackStreamedResponse
