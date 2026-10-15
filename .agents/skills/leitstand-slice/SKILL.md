---
name: leitstand-slice
metadata:
  invocation: manual
description: Review and update the existing Leitstand Slice plan when explicitly requested. Load the project's pinned planning and task contract. Not for Brainstorm intake, status inspection or starting execution.
---

# Leitstand Slice

Update the existing plan from the currently known Auftrag. This Skill loads the
central planning rules; it does not define a second task contract or grant Go
to execute the planned work.

1. Use `leitstand-web` to resolve the actual project, ownership and exact
   workflow pin. Load `core`, `state` and `orchestration` through
   `leitstand_workflow` with that version, using the supported transport selected
   by `leitstand-web` and its current schemas. Reuse sections already loaded for
   the same pin in this session.
2. Read current State and the existing task tree, then the relevant stored
   task contracts, source pins, source bodies and results. Reconcile them with
   the user's current Slice request and accepted decisions before changing the
   plan. Follow the pinned workflow for known work, open questions, grouping
   parents, historical results and source drift; preserve older pins' duties.
3. Update the matching tasks through the pinned workflow's supported writes
   and current versions. Keep the workplan in the surface assigned by that pin.
   Read back changed tasks and any changed State references to verify the delta.
   Planning ends here; dispatch requires authority covering that work.

Return a compact plan delta with task addresses, the reason for changed Scope,
Verification criteria for known work and the next bounded question where the
path is still unknown. Identify unchanged historical results only when relevant.
On an unresolved project, unavailable connection, source or version conflict,
preserve the proposed delta and name the exact blocker and next move; do not
claim that the plan was saved or upgrade the workflow pin.
