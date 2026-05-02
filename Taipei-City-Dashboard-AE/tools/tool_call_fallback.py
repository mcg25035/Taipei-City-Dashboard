"""Regex fallback parser for Llama tool-call text-leak.

Some inference servers (notably TWCC FFM Llama 3.3) return tool calls as
plain text inside `message.content` instead of populating the OpenAI-style
`message.tool_calls` field. The leaked text looks like:

    tool<function=geocode_place>{"query": "Taipei 101"}</function>->>>|separate|

This module extracts those embedded calls so downstream code can treat
them as if the server had returned proper structured tool calls.

Pure-function, stateless. No project dependencies.
"""

from __future__ import annotations

import json
import logging
import re
from dataclasses import dataclass

logger = logging.getLogger(__name__)


FUNCTION_HEADER_RE = re.compile(r"<function=([a-zA-Z_][a-zA-Z0-9_]*)>")
FUNCTION_CLOSE_RE = re.compile(r"\s*</function>")

TEMPLATE_ARTIFACT_PATTERNS: tuple[re.Pattern[str], ...] = (
    re.compile(r"->>>\|separate\|.*$", re.DOTALL),
    re.compile(r"^\s*tool\b\s*", re.IGNORECASE),
)


@dataclass(frozen=True)
class ParsedToolCall:
    name: str
    arguments: dict


def extract_tool_calls(text: str) -> tuple[list[ParsedToolCall], str]:
    """Scan `text` for `<function=NAME>{json}</function>` patterns.

    Returns (calls, cleaned_text). Parsed segments are removed from
    cleaned_text and known template artifacts are stripped. Malformed
    JSON segments are left untouched in cleaned_text and skipped.
    """
    if not text:
        return [], text or ""

    calls: list[ParsedToolCall] = []
    cleaned_parts: list[str] = []
    cursor = 0
    decoder = json.JSONDecoder()

    for match in FUNCTION_HEADER_RE.finditer(text):
        if match.start() < cursor:
            continue

        json_region = text[match.end():]
        leading_ws = len(json_region) - len(json_region.lstrip())
        json_start_local = leading_ws

        try:
            args, json_end_local = decoder.raw_decode(json_region[json_start_local:])
        except json.JSONDecodeError as exc:
            logger.warning(
                "tool_call_fallback: malformed JSON for function=%s at offset %d: %s",
                match.group(1),
                match.end() + json_start_local,
                exc,
            )
            cleaned_parts.append(text[cursor:match.end()])
            cursor = match.end()
            continue

        if not isinstance(args, dict):
            logger.warning(
                "tool_call_fallback: function=%s args not an object (%s); skipping",
                match.group(1),
                type(args).__name__,
            )
            cleaned_parts.append(text[cursor:match.end()])
            cursor = match.end()
            continue

        cleaned_parts.append(text[cursor:match.start()])
        json_end_abs = match.end() + json_start_local + json_end_local

        close = FUNCTION_CLOSE_RE.match(text, json_end_abs)
        cursor = close.end() if close else json_end_abs

        calls.append(ParsedToolCall(name=match.group(1), arguments=args))

    cleaned_parts.append(text[cursor:])
    cleaned = "".join(cleaned_parts)

    for pattern in TEMPLATE_ARTIFACT_PATTERNS:
        cleaned = pattern.sub("", cleaned)

    return calls, cleaned.strip()
