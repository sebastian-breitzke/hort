---
name: leitstand-abschluss
metadata:
  invocation: manual
description: Review and complete a Leitstand project when Abschluss is explicitly requested. Reconcile the full Auftrag and open ends, secure owned execution, then verify the actual project lifecycle. Not for a Status, Checkpoint or unfinished-session Handover.
---

# Leitstand Abschluss

Complete the selected project only after its Auftrag is reconciled and owned
execution is safe. This explicit/manual loader belongs to the repo bundle.

Use `leitstand-web` to resolve the actual project, ownership and exact workflow
pin. Load `close` through `leitstand_workflow` at that version, plus the sections
and current tool schemas it requires. Follow that central close contract for
Auftrag/O-Ton reconciliation, criterion Evidence, open ends, execution cleanup,
actual lifecycle and readback; this Skill adds no second close policy.

An Abschluss request authorizes that close workflow, not new product Scope or
implicit Deferral. Keep unmet work open until fulfilled or explicitly excluded
under user authority with its remaining location. If required sources, pin,
ownership, safe execution controls or lifecycle Capability cannot be verified,
report the precise blocker and next action. Do not substitute State.phase,
archive or a remembered API for actual completion. Do not upgrade the pin.

Return once in the central close output shape, distinguishing verified project
status from pending work and unverified actions. End the turn after the receipt;
an unfinished session departure uses Handover.
