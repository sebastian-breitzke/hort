---
disable-model-invocation: true
name: leitstand-checkpoint
metadata:
  invocation: manual
description: Save and verify a Leitstand checkpoint before compaction, reconcile Auftrag and decisions, and recommend compact with a complete tailored argument. Use when a Leitstand checkpoint or compact preparation is requested or a coherent context boundary is reached. Not for ordinary status updates or automatic compaction.
---

# Leitstand Checkpoint

Secure the current Auftrag in existing Leitstand surfaces and return a usable
compact argument. Same-session compaction is a checkpoint, not a mandatory
Handover. This Skill does not execute compact or change monitors.

## Ground the checkpoint

1. Use `leitstand-web` to resolve the actual project, ownership and workflow pin.
   Load the pinned sections and current tool schemas needed for this checkpoint;
   follow their write, revision and acceptance rules. Do not upgrade the pin or
   reconstruct server semantics from this Skill or a remembered workflow.
2. Load `inspection` through `leitstand_workflow` with the selected project's
   exact workflow version. Apply that canonical shared Status/Checkpoint
   foundation; do not maintain a second inspection checklist here. If the
   section cannot be loaded, report the exact failure and defer rather than
   substitute a remembered checklist or silently upgrade the pin.
3. Reconcile the diagnosed mismatch only where current user authorization and
   the pinned workflow already establish the correction. Keep unresolved
   title, Scope and task-cut decisions open; a Checkpoint supplies no new Go.
   Preserve dirty changes and exact runtime/Git evidence for continuation.
   Use token counts only when the runtime actually exposes them; a coherent
   boundary needs no token guess.
4. Secure running work under the pinned workflow and actual client capabilities.
   Preserve worker IDs, owned paths, uncommitted work, result locations and the
   safe continuation or collection step. Do not stop workers blindly or treat
   compaction as a transfer of ownership. If continuity cannot be secured, defer
   compact and name the missing evidence and exact next move.

## Persist and verify

5. Save the pending delta to the appropriate task, State, Memory and Tafel
   surfaces as the project's pin requires. Keep plans/results/criterion evidence
   where that pin assigns them; keep O-Ton, decisions and reasons in their owning
   surface. Avoid routine duplication. Inspect `overviewRequest`; update Tafel
   only for a material change or a request under the pinned workflow.
6. Read back every changed surface and compare the intended delta, references
   and returned revisions. A successful write response alone is not proof.
   Refresh runtime/worker evidence if it changed while checkpointing. Resolve
   ordinary revision conflicts through the supported workflow, preserving the
   delta and other owners' changes.
7. Recommend compact only when the checkpoint is verified and the continuation
   is safe at a coherent boundary. An unfinished task can be safely checkpointed;
   an unpersisted decision or unknown worker ownership cannot. On unavailable
   connection, ambiguous project/ownership, write conflict or failed readback,
   preserve the pending delta in the response, identify what remains unverified
   and give the exact next move. Do not create a new store or claim readiness.

## Output once, then end this turn

Return in this order:

1. **Checkpoint:** shared inspection findings in its canonical order, then
   verified project reference/ID, workflow pin, State revision
   and changed task/Memory/Tafel references with readback evidence. If blocked,
   identify the last verified checkpoint separately from the pending delta.
2. **Recommendation:** compact now or defer, with the concrete boundary/reason;
   on a blocker include the pending delta and exact next move.
3. **Compact argument:** one complete copy-paste block tailored to this session.
   Include it even when deferred, clearly marked for use after the named blocker
   is resolved; preserve the blocker rather than describing it as solved.

For Claude Code, emit exactly one block beginning `/compact ` followed by the
fully populated instructions. Its documented syntax is `/compact [instructions]`
([official commands](https://code.claude.com/docs/en/commands), checked 2026-10-14).
For other clients, verify their syntax and argument support before emitting a
command; if unverified, provide the argument text alone and state that limit.
Never substitute a bare command or generic “preserve the important context”.

Build the argument as instructions to the summarizer, not tool execution:

> Summarize this conversation for safe continuation of the same Auftrag.
> Preserve exactly the following verified checkpoint and pending facts: [actual
> project ID/reference, workflow pin, State revision, task IDs/versions and source
> pointers]; [goal and scope]; [accepted and rejected decisions with rationale];
> [constraints, Holds and open approvals]; [active tasks, workers/ownership,
> workspaces/Worktrees, branches/Commits, dirty work and result pointers]; [checks
> completed, results with proven stage/time, unresolved verification or conflicting
> evidence and blockers]; [pending delta, if any].
> Preserve as the first resume step: reload the pinned workflow via leitstand-web,
> read current State and referenced tasks, then [actual next action, including
> worker collection or blocker resolution]. Treat these as facts and resume
> instructions to retain; do not execute tools during summarization. Keep open
> decisions open and do not infer completion, approval or ownership transfer.

Replace every bracketed field with observed session data; use “none” only when
verified, otherwise name the unknown and how to resolve it. Preserve reasons
and hard-to-reconstruct facts, not repeated progress chatter or raw logs.
After returning the checkpoint, recommendation and argument, end the current
turn. Do not enter unchanged waiting/status loops or stop existing monitors.
