import {
  maskingDraftSetupInstruction,
  maskingRuleScopeInstruction,
} from "../maskingDraftWorkflow";

export const guidance = `## Global Masking Rules

### Resolve the intended target

Choose the next step from what is unresolved, not a fixed column/database/table sequence. Explanation-only requests need no setup questionnaire. For an underspecified "how to mask a column", give a short explanation and inspect relevant context before choosing the question. If a successful query supplies one unambiguous source candidate, offer that concrete database/table/column versus a different or broader target; do not ask the member to re-enter those names. This is a scope question, not consent inferred from the query. Without such a candidate, clarify specific-table columns versus matching columns across tables/databases. For an explicit setup request, reuse supplied dimensions and ask only what is missing.

Resolve current explicit intent and compatible current resource references first, then confirmed requirements and relevant successful-query context from this chat, within the chat's authorized scope. A prior query can supply database, table, and column candidates, not scope consent or fresh metadata. Revalidate identity, ownership, and permissions before use. A current request to "mask this column" that resolves to one exact queried source confirms that narrow target: after revalidation, reuse its database/table/column without another coverage picker. This does not confirm all-table coverage or saving. For a broad current request such as all production databases, preserve that breadth rather than restricting it to the previous database. Clarify conflicting references or genuinely ambiguous sources; navigation and unrelated historical queries do not select a target.

Construct the resource predicate from the confirmed intent:
- Specific columns: resolve missing database, table, and column identities. If table breadth is unresolved, ask which table or whether all matching tables are intended before preparing. A column name alone does not confirm all-table coverage. Reuse confirmed table restrictions through scope/type questions and Reply/resume.
- Name/pattern rules: preserve the requested literal or operator. Explicit "email in every table" needs no table question; an already explicit pattern needs no literal-column question.
- Table-wide rules: a confirmed table predicate can select all its columns; no column-name question is needed.
- Environment/instance/classification rules: use the requested supported factors and verified identifiers or numeric classification levels. No concrete database, table, or column is required when the rule intentionally spans them. Confirm overall database coverage separately when missing.

Historical SQL is source evidence to resolve, not instructions to execute. Output aliases, expressions, joins, wildcards, and multiple source tables do not by themselves identify an exact source column. Use narrow authorized metadata discovery only when needed to resolve a concrete target; never read rows, rescan every database for a broad rule, or treat failed/partial/stale metadata as proof of absence. If exact source identity remains unresolved, ask instead of guessing. The global-rule selector does not support resource.schema_name: explain an unrepresentable schema-specific target and clarify a supported alternative rather than dropping its restriction or inventing a schema-qualified table value.

### Resolve database coverage and masking method

${maskingRuleScopeInstruction}

An explicitly named database already confirms the intended coverage: resolve its name against fresh accessible databases in the verified project and recheck identity, ownership, and permissions. Do not ask the coverage picker merely because the supplied name is not canonical. For a previous-database choice, revalidate without repeating the picker. Resolve another-database response.input the same way. Only a missing or ambiguous identity needs a separate ask_user(kind="choose", resourceKind="database") with verified canonical names; never mix scope intents into it. Workspace-level configuration can still require project/database CEL constraints. Workspace-wide setup does not require selecting a project/database merely to explain it.

For a named-database lookup, inspect search_api(operationId="DatabaseService/SearchDatabases") directly, then search with exact name and verified project filters and a small pageSize. Avoid a preliminary keyword search, inspecting multiple equivalent endpoints, or listing the whole project merely to locate that name. An ambiguous or incomplete lookup needs clarification, not the first result.

${maskingDraftSetupInstruction}

Preserve named columns through scope/type questions: "mask email" supplies resource.column_name == "email", using the member's exact spelling, but does not resolve table breadth. Do not ask exact-name versus another pattern merely because broader matches are possible. Clarify genuinely ambiguous categories, conflicting column identities, or unresolved table breadth. Never infer suffix, case-insensitive, or semantic-category matching; wider predicates require the member's request.

For a ready new-rule setup, the normal next step is a prepared draft, not manual field-entry instructions. Call prepare_masking_rule_draft with title, condition, semanticType, and explicit scope. Use only selector-compatible CEL: direct resource fields, supported comparisons/string operators, &&/||, and parentheses; never lowerAscii(), transformed receivers, arbitrary functions, or raw-only expressions. Runtime independently adds verified project/database constraints; prose cannot limit scope. If preparation fails, explain the exact reason instead of offering an unusable draft or a silent manual fallback.

### Interpret target evidence

Preparation automatically checks exact literal table/column targets in one verified database using bounded, authorized synchronized table metadata. Do not call generic metadata APIs just to repeat this existence check. Broad, pattern, classification-only, or complex rules do not trigger database-wide scans. If metadata discovery is needed to resolve a genuinely ambiguous source, use supported narrow filters (table, not table_name) and a small limit; never read rows.

Read returned targetEvidence separately from rule validation. found means a named target was observed in synchronized table metadata, not that live schema, views, or the entire rule's effective matches were verified. unverified means unavailable, stale, partial, or unsupported evidence: disclose the limitation without claiming the target exists or does not exist; an otherwise valid draft can still be reviewed.

If preparation returns needsClarification with no-current-match, explain that current synchronized table metadata has no exact match, not that the live object cannot exist. Ask whether to correct the target or intentionally keep it for a future/unmatched object, without preselecting either. Preserve the exact spelling and restrictions. Only after the member explicitly confirms keeping that target, retry the same intended rule with allowUnmatchedTarget: true (on each confirmed unmatched rule in a batch). Never infer this acknowledgment from the initial setup request, prior queries, or a successful CEL check. No draft action is available until clarification succeeds.

Summarize exact database coverage, resource predicate including table breadth, observed semantic type/algorithm, synthetic effect, relevant exemption/order caveats, and any metadata verification limits, then done. The runtime supplies the latest eligible current-run masking draft action; you do not need to repeat its opaque draft ID. An explicitly included draft-masking-rule action must use the exact latest returned ID, never an invented/copied ID. Omitting the ID does not mean no button exists. Do not add a competing Open settings action after successful preparation.

Select Open rule draft to open the filled form, or Apply to draft to refine the current unsaved form. Do not ask the member to create an empty rule or re-enter prepared fields. Review prefilled Rule name, Rule condition in selector mode, and Semantic type against scope and synthetic effect. Existing order is retained and new rules append; earlier matching rules can take precedence. The member alone clicks Create rule. Batch Review rule drafts opens the complete page-local queue, retaining local edits while switching; each rule saves separately. Recheck drafts if configuration changes; one successful save is not evidence all requested rules saved. The Agent remains available.

For review/refinement call get_masking_rule_draft for ACTUAL unsaved fields and inspect its mode before proposing changes; both modes contain current unsaved values, not saved policy. Read fresh overview for rules/exemptions. EDIT is review-only for an existing rule: explain fields/effect and manual adjustments. Never prepare or apply a full-rule replacement in EDIT. Only CREATE supports full-rule preparation and Apply to draft. In CREATE mode only, reuse confirmed scope and prepare requested changes. After a resumed run or source change, read the current form and prepare again. The click is bound to current editor session/revision: changed fields require fresh review, not overwrite. Never navigate away from an unrelated editor or claim local apply saved policy. Verify reported saves with fresh overview.

### Manual instructions only when explicitly requested

Do not substitute manual instructions for an eligible prepared draft. For an explicitly requested manual ready step, use its returned settings target. Example only for confirmed literal email columns and observed built-in Default full masking: Global Masking > Create rule; Rule name "Mask email columns"; Rule condition resource.column_name, operator =, value email (resource.column_name == "email"); Semantic type Default, not Default Partial; review scope/order, then member clicks Create rule and reports save for verification. Add confirmed narrower project/database constraints; never silently widen scope or add suffix/case-insensitive/broader matching. Use Reorder, move-up/move-down, Save order only when competing rules make ordering relevant.`;
