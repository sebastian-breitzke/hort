---
name: leitstand-handover
metadata:
  invocation: manual
description: Load the pinned Leitstand Handover for continuation in another session or client, reusing Checkpoint where supported. Use for an explicit Handover; not same-session compaction, ordinary Status or parallel delegation.
---

# Leitstand Handover

This manual repo loader routes an explicit Handover to the central workflow.
The pinned `handover` section owns preservation, observed stop, source access,
registration/transfer, failure handling, resume and continuation requirements.

1. Use `leitstand-web` to resolve the actual project, ownership and exact workflow
   pin. Load its manifest and `handover` section, then only the sections and tool
   schemas that path requires. Do not substitute or silently upgrade the pin.
2. Only on an executable pin whose manifest exposes `inspection`, use
   `leitstand-checkpoint` and that shared foundation for reconciliation,
   persistence and readback. Reuse a current verified checkpoint after refreshing
   changed evidence. Suppress its Compact recommendation/argument and emit the
   Handover output once. Historical/non-executable pins follow their pinned
   `handover`/`legacy` preservation and authorized upgrade path; do not require
   `core`, Checkpoint or `inspection` there.
3. Follow the central Handover sequence and its supported capabilities. It grants
   no new Scope or Build-Go. Missing capability or evidence follows that section's
   blocked path; a prompt is not proof of transfer.

Return once in this order, using the central requirements:

1. **Checkpoint:** shared inspection findings and verified reconciliation/readback
   on supported pins; on historical pins, the preserved evidence and exact pin.
2. **Handover:** preservation, actual stops, target access, registration/ownership
   outcome and blocker/next action.
3. **Continuation prompt:** one fully populated copy-paste block meeting the
   central `handover` contract, including unresolved blockers when deferred.

End the turn after this output; do not emit a Compact argument or unchanged wait
loop.
