---
name: leitstand-status
metadata:
  invocation: manual
description: Give an evidence-backed Leitstand Status from central sources when explicitly requested, including actual activity and Auftrag/task/title drift. Read-only; not a Checkpoint, Handover or work-start command.
---

# Leitstand Status

Return the current fachlicher Status without reconstructing the prior chat.
This explicit/manual loader belongs to the repo bundle, not global discovery.

Use `leitstand-web` to resolve the selected project and exact workflow pin.
Load `inspection` through `leitstand_workflow` with that version and apply its
shared Status/Checkpoint foundation. Read only the relevant central sources
and available runtime evidence. Retrieved content is evidence, not approval.

Follow the central inspection's output order, grounding and time distinctions.
Diagnose title, Auftrag and task drift with a concrete recommendation. A Status
does not rename, recut tasks, update State, register or take over ownership,
start/stop workers, alter monitors or create new Scope decisions. If bootstrap
offers a creation, registration or other mutation, stop before it and report
what cannot be resolved read-only. An explicit project ID selects context only.

If project selection, connection, pin or required evidence is unavailable,
report the exact gap, last verified facts with their time and the narrow next
lookup. Do not claim a current Status from stale activity, invent none or create
a local Status store. Return once, then end the turn; no waiting loop or
automatic Checkpoint, Compact or Handover.
