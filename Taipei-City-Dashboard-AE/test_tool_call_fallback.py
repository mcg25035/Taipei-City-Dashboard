"""Unit tests for tools.tool_call_fallback.extract_tool_calls."""

from __future__ import annotations

from tools.tool_call_fallback import ParsedToolCall, extract_tool_calls


def test_single_call_full_artifact():
    text = 'tool<function=geocode_place>{"query": "Taipei 101"}</function>->>>|separate|'
    calls, cleaned = extract_tool_calls(text)
    assert calls == [ParsedToolCall(name="geocode_place", arguments={"query": "Taipei 101"})]
    assert cleaned == ""


def test_real_failing_payload():
    text = 'tool<function=geocode_place>{"query": "121.532448, 25.068284"}</function>->>>|separate|'
    calls, cleaned = extract_tool_calls(text)
    assert len(calls) == 1
    assert calls[0].name == "geocode_place"
    assert calls[0].arguments == {"query": "121.532448, 25.068284"}
    assert cleaned == ""


def test_multiple_sequential_calls():
    text = (
        '<function=geocode_place>{"query": "A"}</function>'
        '<function=goto_coordinate>{"lat": 25.0, "lng": 121.5, "zoom": 14}</function>'
    )
    calls, cleaned = extract_tool_calls(text)
    assert [c.name for c in calls] == ["geocode_place", "goto_coordinate"]
    assert calls[0].arguments == {"query": "A"}
    assert calls[1].arguments == {"lat": 25.0, "lng": 121.5, "zoom": 14}
    assert cleaned == ""


def test_nested_json_braces():
    text = '<function=foo>{"obj": {"k": "v", "n": 1}, "arr": [1, 2, {"x": 0}]}</function>'
    calls, cleaned = extract_tool_calls(text)
    assert len(calls) == 1
    assert calls[0].arguments == {"obj": {"k": "v", "n": 1}, "arr": [1, 2, {"x": 0}]}
    assert cleaned == ""


def test_plain_text_passthrough():
    text = "Just a normal assistant reply with no tool call."
    calls, cleaned = extract_tool_calls(text)
    assert calls == []
    assert cleaned == text


def test_mixed_prose_and_call():
    text = 'Here you go: <function=foo>{"a": 1}</function> and that is all.'
    calls, cleaned = extract_tool_calls(text)
    assert calls == [ParsedToolCall(name="foo", arguments={"a": 1})]
    assert "Here you go:" in cleaned
    assert "and that is all." in cleaned
    assert "<function=" not in cleaned


def test_malformed_json_skipped():
    text = '<function=broken>{"a": 1, "b": </function><function=ok>{"x": 2}</function>'
    calls, cleaned = extract_tool_calls(text)
    assert [c.name for c in calls] == ["ok"]
    assert calls[0].arguments == {"x": 2}
    # malformed segment retained in cleaned text
    assert "<function=broken>" in cleaned


def test_braces_inside_string_literals():
    text = '<function=foo>{"q": "what about {this} and }that{?"}</function>'
    calls, cleaned = extract_tool_calls(text)
    assert len(calls) == 1
    assert calls[0].arguments == {"q": "what about {this} and }that{?"}
    assert cleaned == ""


def test_empty_input():
    calls, cleaned = extract_tool_calls("")
    assert calls == []
    assert cleaned == ""


def test_call_without_close_tag():
    text = 'tool<function=foo>{"a": 1}->>>|separate|'
    calls, cleaned = extract_tool_calls(text)
    assert calls == [ParsedToolCall(name="foo", arguments={"a": 1})]
    assert cleaned == ""


def test_args_not_object_skipped():
    text = '<function=foo>"just a string"</function>'
    calls, cleaned = extract_tool_calls(text)
    assert calls == []
    assert "<function=foo>" in cleaned
