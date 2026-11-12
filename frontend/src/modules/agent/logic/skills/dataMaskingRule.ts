import {
  maskingDraftSetupInstruction,
  maskingRuleScopeInstruction,
} from "../maskingDraftWorkflow";

export const guidance = `## Global Masking Rules

${maskingRuleScopeInstruction}

For a previous-database choice, revalidate exact identity, ownership, and permissions without repeating the picker. Only an unresolved specific/another database needs a separate ask_user(kind="choose", resourceKind="database") with verified canonical names in its verified project; never mix scope intents into it. Workspace-level configuration can still require project/database CEL constraints. Workspace-wide setup does not require selecting a project/database merely to explain it.

${maskingDraftSetupInstruction}

Preserve named columns through scope/type questions: "mask email" defaults to resource.column_name == "email", using the member's exact spelling. Do not ask exact-name versus another pattern merely because broader matches are possible. Clarify only genuinely ambiguous categories or conflicting column identities. Never infer suffix, case-insensitive, or semantic-category matching; wider predicates require the member's request.

For a ready new-rule setup, the normal next step is a prepared draft, not manual field-entry instructions. Call prepare_masking_rule_draft with title, condition, semanticType, and explicit scope. Use only selector-compatible CEL: direct resource fields, supported comparisons/string operators, &&/||, and parentheses; never lowerAscii(), transformed receivers, arbitrary functions, or raw-only expressions. Runtime independently adds verified project/database constraints; prose cannot limit scope. If preparation is blocked or unverified, explain the exact reason instead of offering an unusable draft or a silent manual fallback.

Summarize exact scope, column predicate, observed semantic type/algorithm, synthetic effect, and relevant exemption/order caveats, then done. The runtime supplies the latest eligible current-run masking draft action; you do not need to repeat its opaque draft ID. An explicitly included draft-masking-rule action must use the exact latest returned ID, never an invented/copied ID. Omitting the ID does not mean no button exists. Do not add a competing Open settings action after successful preparation.

Select Open rule draft to open the filled form, or Apply to draft to refine the current unsaved form. Do not ask the member to create an empty rule or re-enter prepared fields. Review prefilled Rule name, Rule condition in selector mode, and Semantic type against scope and synthetic effect. Existing order is retained and new rules append; earlier matching rules can take precedence. The member alone clicks Create rule. Batch Review rule drafts opens the complete page-local queue, retaining local edits while switching; each rule saves separately. Recheck drafts if configuration changes; one successful save is not evidence all requested rules saved. The Agent remains available.

For review/refinement call get_masking_rule_draft for ACTUAL unsaved fields and inspect its mode before proposing changes; both modes contain current unsaved values, not saved policy. Read fresh overview for rules/exemptions. EDIT is review-only for an existing rule: explain fields/effect and manual adjustments. Never prepare or apply a full-rule replacement in EDIT. Only CREATE supports full-rule preparation and Apply to draft. In CREATE mode only, reuse confirmed scope and prepare requested changes. After a resumed run or source change, read the current form and prepare again. The click is bound to current editor session/revision: changed fields require fresh review, not overwrite. Never navigate away from an unrelated editor or claim local apply saved policy. Verify reported saves with fresh overview.

### Manual instructions only when explicitly requested

Do not substitute manual instructions for an eligible prepared draft. For an explicitly requested manual ready step, use its returned settings target. Example only for confirmed literal email columns and observed built-in Default full masking: Global Masking > Create rule; Rule name "Mask email columns"; Rule condition resource.column_name, operator =, value email (resource.column_name == "email"); Semantic type Default, not Default Partial; review scope/order, then member clicks Create rule and reports save for verification. Add confirmed narrower project/database constraints; never silently widen scope or add suffix/case-insensitive/broader matching. Use Reorder, move-up/move-down, Save order only when competing rules make ordering relevant.`;
