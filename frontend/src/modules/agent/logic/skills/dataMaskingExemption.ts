import {
  maskingExemptionDraftSetupInstruction,
  maskingExemptionScopeInstruction,
} from "../maskingDraftWorkflow";

export const guidance = `## Project Masking Exemptions

## Choose an unmask method

For a general "I want unmasked data" request without an explicit or confirmed method, call ask_unmask_method({project?,database?,queryAccessMissing?,forCurrentUser?}) with confirmed canonical scope. It freshly checks each method independently, without reading masking policy; do not require get_masking_overview first. Do not supply options, a default, or a preferred method. When both methods are eligible, the runtime offers unselected Masking exemption and JIT Access Grant choices; wait for Reply. When only one is eligible, continue the returned method directly. A method:null result means explain the returned setup blockers, not offer an unusable draft. For needsScope, ask only for the missing project and call again.

Set queryAccessMissing:true only when the member explicitly says they cannot query or a structured select/info permission denial establishes it. Masked values, a generic query error, or inferred roles are not that evidence. Known missing query access needs JIT: an exemption alone cannot grant query access. Set forCurrentUser:false for another member's request; JIT is self-only. Preserve the confirmed database, SQL, and intended data scope through the choice.

Explain the distinction: a masking exemption covers selected resources and members, with its own expiration or no expiration, independently of an exact SQL query. A JIT Access Grant covers the current user, one exact read-only query and its targets, with finite expiration; Unmask affects the approved query's whole result, not just one column.

Use the answer or returned method unchanged: masking-exemption continues below; just-in-time loads get_skill(name="just-in-time"). Honor an explicitly requested method the same way, even when both are eligible. Do not repeat ask_unmask_method after a confirmed choice or explicit method, prefer exemption automatically, or require exemption denial before JIT. Recheck the selected method's eligibility before preparation; if it became unavailable, explain its blocker rather than silently changing methods. Unreadable exemption policy alone must not block independent JIT eligibility. Neither method promises exclusive visibility.

## Prepare an exemption

For the selected exemption flow, check fresh get_masking_overview eligibility before exemption-specific scope/member/expiration questions. Do not ask dependent questions or offer an unusable draft when blocked or unverified. Exemptions do not grant missing query access; explain that limitation even for an explicitly requested exemption.

${maskingExemptionScopeInstruction}

${maskingExemptionDraftSetupInstruction}

Explicitly confirm members, exact project/database/resource predicate, and future expiry or no expiration. Do not copy global-rule CEL into exemption selectors; use supported resource.database, schema_name, table_name, column_name, and classification_level factors. Keep expiry separate. Runtime adds the independently verified database constraint. An additional self-only grant does not revoke other exemptions or establish exclusive unmasked access.

Reuse an already explicit expiration, including no expiration, instead of asking again. When expiration is missing, call ask_user(kind="choose", resourceKind="none") with an options array offering "In 1 hour" (value "1-hour"), "In 24 hours" (value "24-hours"), "Choose a date and time" (value "custom"), and "No expiration" (value "no-expiration"), with labels in the conversation language. Leave all options unselected: omit defaultValue and wait for Reply. Do not start with a free-text expiration question. If the member selects the custom date/time option, use ask_user(kind="input", resourceKind="none") to collect the missing date/time in their time zone; custom alone is not a confirmed timestamp. A relative-duration choice confirms that duration; no-expiration explicitly selects null. Neither selection grants permission to save.

For relative expiry read get_masking_overview again and use its fresh currentTime (UTC ISO), never an assumed model date or old timestamp. "One hour from now" confirms expiry intent: calculate future UTC ISO expirationTimestamp instead of asking for raw ISO. Honor explicitly named IANA time zones; otherwise use returned browser timeZone for local interpretation, never infer from language. Include resolved expiry/time zone in the summary. Ask only for missing/ambiguous date, time, or zone, including daylight-saving ambiguous times.

Before refining, read get_masking_exemption_draft for actual CREATE form values; never treat them as saved policy. Apply to draft must retain the same editor/session, source identity, and revision; reread and prepare again after resume/source changes or changed fields rather than overwrite. Never replace an unrelated editor or automatically navigate. Finish with a concise scope/member/expiry and visibility summary and done. The latest eligible current-run preparation owns the draft action; no model-invented draft IDs or competing settings link. Open exemption draft prefills only after the member clicks, Apply to draft refines unchanged unsaved fields, and only Grant Exemption saves. Refresh overview to verify any reported save.`;
