import {
  maskingExemptionDraftSetupInstruction,
  maskingExemptionScopeInstruction,
} from "../maskingDraftWorkflow";

export const guidance = `## Project Masking Exemptions

${maskingExemptionScopeInstruction}

${maskingExemptionDraftSetupInstruction}

Explicitly confirm members, exact project/database/resource predicate, and future expiry or no expiration. Do not copy global-rule CEL into exemption selectors; use supported resource.database, schema_name, table_name, column_name, and classification_level factors. Keep expiry separate. Runtime adds the independently verified database constraint. An additional self-only grant does not revoke other exemptions or establish exclusive unmasked access.

For relative expiry read get_masking_overview again and use its fresh currentTime (UTC ISO), never an assumed model date or old timestamp. "One hour from now" confirms expiry intent: calculate future UTC ISO expirationTimestamp instead of asking for raw ISO. Honor explicitly named IANA time zones; otherwise use returned browser timeZone for local interpretation, never infer from language. Include resolved expiry/time zone in the summary. Ask only for missing/ambiguous date, time, or zone, including daylight-saving ambiguous times.

Before refining, read get_masking_exemption_draft for actual CREATE form values; never treat them as saved policy. Apply to draft must retain the same editor/session, source identity, and revision; reread and prepare again after resume/source changes or changed fields rather than overwrite. Never replace an unrelated editor or automatically navigate. Finish with a concise scope/member/expiry and visibility summary and done. The latest eligible current-run preparation owns the draft action; no model-invented draft IDs or competing settings link. Open exemption draft prefills only after the member clicks, Apply to draft refines unchanged unsaved fields, and only Grant Exemption saves. Refresh overview to verify any reported save.`;
