---
name: leitstand-web
description: Use Leitstand for durable coordination, meaningful repository changes, substantial briefs, project workflow, resume, handover, or completion. Load the central workflow through the supported authenticated transport in this client. Not for trivial questions, narrow lookups, one-step edits or delegated workers inside a brief.
---

# Leitstand

The Leitstand service owns project workflow and durable State. Runtime and current user instructions remain authoritative.

## First use in a session

The repo installation puts this Skill in `.agents/skills/leitstand-web/` for Codex
and `.claude/skills/leitstand-web/` for Claude Code. Resolve the actual loaded path;
do not use a Home-tool path or a second Skill copy. If the client selects a
global copy instead of the intended repo copy, report that discovery conflict;
installation does not remove personal Skills.

1. Honor the caller's selected authenticated transport when supported. For a local CLI session, use `scripts/leitstand.py` relative to this loaded Skill as `CLI`; it retains its existing macOS Keychain login. For native access, including Web sessions, use the Leitstand MCP connector only when its callable tools and schemas are actually exposed in this session. A connection saved in ChatGPT does not prove availability in Codex Web. Do not export credentials or create a new auth path to bridge clients.
2. Retrieve only the `leitstand_workflow` schema: `CLI help leitstand_workflow` locally, or the native client's supported tool discovery. Call that tool with `{"section":"bootstrap"}` and omit `version`. Locally use `CLI call leitstand_workflow --json @FILE`, with a private JSON file of mode `600`, removed immediately after the call. The native connector accepts the same object directly; it requires no Home files, CLI login or macOS tools.
3. Follow the returned bootstrap and pinned project workflow. Refresh them at session start, after resume or handover, and when a workflow change is announced. Load only the sections or tool schemas needed for the current step, using the exact project workflow version when required.

Both transports load the same central bootstrap, project pin and tool contracts;
the selected transport carries subsequent calls, not a second local workflow.
If it is missing, unavailable, or unauthorized, report the exact failure and use
only the client's supported connection flow. Do not invent credentials, endpoints,
capabilities, or project identifiers. Do not claim a project was resolved or
changed without successful service evidence. Preserve pending input in the
conversation while access is blocked; do not create a replacement local State.
