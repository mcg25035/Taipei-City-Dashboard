"""Integration tests for TWCCFallbackStreamedResponse stateful watchdog."""

from __future__ import annotations

import asyncio

import pytest
from openai.types.chat import ChatCompletionChunk
from openai.types.chat.chat_completion_chunk import (
    Choice as ChunkChoice,
    ChoiceDelta,
)
from pydantic_ai.models import ModelRequestParameters
from pydantic_ai.profiles.openai import OpenAIModelProfile

from pydantic_ai.messages import PartDeltaEvent, PartStartEvent, TextPart, TextPartDelta, ToolCallPart

from taipei_agent.fallback_model import TWCCFallbackStreamedResponse


def _chunk(content: str | None, finish: str | None = None) -> ChatCompletionChunk:
    return ChatCompletionChunk(
        id="c",
        object="chat.completion.chunk",
        created=0,
        model="m",
        choices=[
            ChunkChoice(
                index=0,
                delta=ChoiceDelta(content=content),
                finish_reason=finish,
            )
        ],
    )


def _run(chunks: list[ChatCompletionChunk]):
    async def gen():
        for c in chunks:
            yield c

    resp = TWCCFallbackStreamedResponse(
        model_request_parameters=ModelRequestParameters(),
        _model_name="m",
        _model_profile=OpenAIModelProfile(),
        _response=gen(),
        _provider_name="twcc",
        _provider_url="http://x",
    )

    async def collect():
        out = []
        async for ev in resp._get_event_iterator():
            out.append(ev)
        return out

    events = asyncio.run(collect())
    return events, resp


def _text(events) -> str:
    out = ""
    for ev in events:
        if isinstance(ev, PartStartEvent) and isinstance(ev.part, TextPart):
            out += ev.part.content
        elif isinstance(ev, PartDeltaEvent) and isinstance(ev.delta, TextPartDelta):
            out += ev.delta.content_delta
    return out


def _tool_calls(events) -> list[ToolCallPart]:
    return [
        ev.part
        for ev in events
        if isinstance(ev, PartStartEvent) and isinstance(ev.part, ToolCallPart)
    ]


def test_pure_text_stream_passthrough():
    events, _ = _run([_chunk("已規劃路線。"), _chunk("請查看地圖。", finish="stop")])
    assert _text(events) == "已規劃路線。請查看地圖。"
    assert _tool_calls(events) == []


def test_start_of_stream_leak_split_chunks():
    events, _ = _run([
        _chunk("tool"),
        _chunk('<function=foo>{"a":1}</function>'),
        _chunk("->>>|separate|", finish="stop"),
    ])
    calls = _tool_calls(events)
    assert len(calls) == 1
    assert calls[0].tool_name == "foo"
    assert _text(events) == ""


def test_mid_stream_leak_after_legit_text():
    """The bug user reported: text streams fine then `<function=...>` leak appears."""
    events, _ = _run([
        _chunk("已將地圖移到台北101，"),
        _chunk("並開啟YouBike圖層。"),
        _chunk("<function=geocode_place>"),
        _chunk('{"query": "台北101"}'),
        _chunk("</function>->>>|separate|", finish="stop"),
    ])
    assert _text(events) == "已將地圖移到台北101，並開啟YouBike圖層。"
    calls = _tool_calls(events)
    assert len(calls) == 1
    assert calls[0].tool_name == "geocode_place"


def test_text_with_literal_less_than():
    events, _ = _run([
        _chunk("1 < 2 是真的"),
        _chunk("，3 < 5 也是。", finish="stop"),
    ])
    assert _text(events) == "1 < 2 是真的，3 < 5 也是。"
    assert _tool_calls(events) == []


def test_two_tool_calls_back_to_back():
    events, _ = _run([
        _chunk("好的，"),
        _chunk('<function=geocode_place>{"query":"A"}</function>'),
        _chunk('<function=goto_coordinate>{"lat":25,"lng":121,"zoom":14}</function>'),
        _chunk("->>>|separate|", finish="stop"),
    ])
    assert _text(events) == "好的，"
    calls = _tool_calls(events)
    assert [c.tool_name for c in calls] == ["geocode_place", "goto_coordinate"]


def test_incomplete_leak_at_eof_recovered():
    events, _ = _run([
        _chunk("開頭"),
        _chunk('<function=foo>{"a":1}'),
        _chunk(" eof", finish="stop"),
    ])
    assert _text(events) == "開頭"
    calls = _tool_calls(events)
    assert len(calls) == 1
    assert calls[0].tool_name == "foo"


def test_artifact_separator_stripped_from_text():
    events, _ = _run([_chunk("正常回覆 ->>>|separate|some leak", finish="stop")])
    assert "->>>" not in _text(events)


def test_native_tool_calls_pass_through():
    """Native delta.tool_calls must NOT be dropped by the text filter."""
    from openai.types.chat.chat_completion_chunk import (
        ChoiceDeltaToolCall,
        ChoiceDeltaToolCallFunction,
    )

    native_call_chunk = ChatCompletionChunk(
        id="c",
        object="chat.completion.chunk",
        created=0,
        model="m",
        choices=[
            ChunkChoice(
                index=0,
                delta=ChoiceDelta(
                    tool_calls=[
                        ChoiceDeltaToolCall(
                            index=0,
                            id="native_1",
                            type="function",
                            function=ChoiceDeltaToolCallFunction(
                                name="navigate",
                                arguments='{"origin_lat":25}',
                            ),
                        )
                    ]
                ),
                finish_reason=None,
            )
        ],
    )

    events, _ = _run([
        _chunk("好的，"),
        native_call_chunk,
        _chunk("規劃中。", finish="tool_calls"),
    ])
    calls = _tool_calls(events)
    assert any(c.tool_name == "navigate" for c in calls)
    assert _text(events) == "好的，規劃中。"


def test_native_call_plus_leaked_text():
    """Server returns a native tool_call AND leaks an extra <function=...>;
    both should surface as ToolCallParts so the agent can continue."""
    from openai.types.chat.chat_completion_chunk import (
        ChoiceDeltaToolCall,
        ChoiceDeltaToolCallFunction,
    )

    native_call_chunk = ChatCompletionChunk(
        id="c",
        object="chat.completion.chunk",
        created=0,
        model="m",
        choices=[
            ChunkChoice(
                index=0,
                delta=ChoiceDelta(
                    tool_calls=[
                        ChoiceDeltaToolCall(
                            index=0,
                            id="native_1",
                            type="function",
                            function=ChoiceDeltaToolCallFunction(
                                name="geocode_place",
                                arguments='{"query":"台北車站"}',
                            ),
                        )
                    ]
                ),
                finish_reason=None,
            )
        ],
    )

    events, _ = _run([
        native_call_chunk,
        _chunk('<function=geocode_place>{"query":"101"}</function>->>>|separate|', finish="tool_calls"),
    ])
    calls = _tool_calls(events)
    names = [c.tool_name for c in calls]
    assert names.count("geocode_place") == 2
    assert _text(events) == ""
