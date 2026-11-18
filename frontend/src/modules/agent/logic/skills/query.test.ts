// @vitest-environment node
import { describe, expect, test } from "vitest";
import { getSkill } from "./index";

describe("query skill", () => {
  test("keeps no-persistence SQL review out of the auto-saving editor", () => {
    const guidance = getSkill({ name: "query" });

    expect(guidance).toContain(
      "SQL Editor automatically creates or updates a Saved Query"
    );
    expect(guidance).toContain(
      "Do not promise that SQL entered there is unsaved or unpersisted"
    );
    expect(guidance).toContain(
      "If the member forbids saving or persistence, do not type SQL into SQL Editor"
    );
    expect(guidance).toContain("return the SQL inline in chat instead");
    expect(guidance).toContain("do not write a Saved Query through APIs");
    expect(guidance).toContain("Do not equate not executed with not saved");
    expect(guidance).toContain("Do not execute review-only SQL");
    expect(guidance).toContain("Execution workflow (only when requested)");
  });

  test("offers contextual masking questions within the existing follow-up limit", () => {
    const guidance = getSkill({ name: "query" });

    expect(guidance).toContain(
      "normally offer at most one masking/access question"
    );
    expect(guidance).toContain("existing 2-3 useful Continue questions");
    expect(guidance).toContain("How can I request access to unmasked data?");
    expect(guidance).toContain("How can I mask these columns?");
    expect(guidance).toContain("never append both by default");
  });

  test("offers masking setup from runtime permissions without model-selected policy reads", () => {
    const guidance = getSkill({ name: "query" });

    expect(guidance).toContain(
      "runtime confirms current masking-configuration permission and feature availability independently of model tool choices"
    );
    expect(guidance).toContain(
      'global-masking step has configure.status="allowed"'
    );
    expect(guidance).toContain('setup.features.masking="available"');
    expect(guidance).toContain(
      'global-masking and semantic-types steps must also have readAccess.status="allowed"'
    );
    expect(guidance).toContain(
      "This suggestion does not require get_masking_overview or masking-policy reads"
    );
    expect(guidance).not.toContain(
      "Offer masking setup only with fresh get_masking_overview evidence"
    );
    expect(guidance).toContain(
      "Omit it when permission or availability is denied, unknown, missing, stale, or outside the verified scope"
    );
    expect(guidance).toContain(
      "Query access, catalog-edit permission, role names, and sensitive-looking columns are not permission to configure global masking"
    );
    expect(guidance).toContain(
      "Existing masking rules or exemptions are not required"
    );
  });

  test("preserves ordinary query questions while grounding only masking/access suggestions", () => {
    const guidance = getSkill({ name: "query" });
    expect(guidance).toContain(
      "Preserve contextual analysis questions from the model in their original wording and order"
    );
    expect(guidance).toContain(
      "The runtime filters and rebuilds only masking/access suggestions"
    );
    expect(guidance).toContain(
      "does not replace ordinary questions or add a generic fallback"
    );
    expect(guidance).not.toContain(
      "The runtime owns query-result Continue questions"
    );
    expect(guidance).toContain(
      "Do not fetch an overview merely to make a suggestion appear"
    );
    expect(guidance).toContain(
      "Prepared masking and access-request workflows retain their own follow-ups"
    );
  });

  test("requires masking reasons instead of inferring masking from row values", () => {
    const guidance = getSkill({ name: "query" });

    expect(guidance).toContain(
      "nonempty per-column reasons in results[].masked"
    );
    expect(guidance).toContain("an array of empty objects is not evidence");
    expect(guidance).toContain(
      "Missing metadata, literal asterisks, and SQL-level redaction do not establish policy masking"
    );
    expect(guidance).toContain("even when rows are omitted or truncated");
    expect(guidance).toContain(
      "Never query more rows, inspect row values, or fetch policies merely to generate follow-ups"
    );
    expect(guidance).toContain(
      "Offer unmasked-access help only when this Query response confirms masked data"
    );
    expect(guidance).toContain(
      "It does not require masking-configuration permission or an existing exemption"
    );
  });

  test("does not infer an existing exemption from successful or visible query results", () => {
    const guidance = getSkill({ name: "query" });

    expect(guidance).toContain(
      "In both the answer and follow-ups, query success or visible values never prove an exemption exists, applies, or has an expiration"
    );
    expect(guidance).toContain(
      "Do not suggest checking an exemption's configuration or expiration after a data query"
    );
    expect(guidance).toContain(
      "Without fresh evidence of an applicable exemption, do not refer to it as the member's exemption"
    );
  });

  test("keeps masking suggestions out of unsuccessful and non-data answers", () => {
    const guidance = getSkill({ name: "query" });

    expect(guidance).toContain("After a successful data-query answer");
    expect(guidance).toContain(
      "Do not add these questions for failed/denied queries, canceled execution, review-only SQL, EXPLAIN, or metadata-only answers"
    );
  });

  test("keeps query follow-up scope explicit without treating it as approval", () => {
    const guidance = getSkill({ name: "query" });

    expect(guidance).toContain(
      "Keep the verified database, project, exact SQL, and intended columns as context, not approval"
    );
    expect(guidance).toContain("selected references stay bound");
    expect(guidance).toContain("self-contained question within 140 characters");
    expect(guidance).toMatch(/omit the suggestion if that scope cannot fit/i);
    expect(guidance).toContain(
      "Do not equate an output alias with a physical source column"
    );
    expect(guidance).toContain(
      "clarify unresolved scope instead of widening it"
    );
  });

  test("routes clicked questions through the existing reviewable masking flows", () => {
    const guidance = getSkill({ name: "query" });

    expect(guidance).toContain(
      'get_skill(name="data-masking", section="rule")'
    );
    expect(guidance).toContain(
      'get_skill(name="data-masking", section="exemption")'
    );
    expect(guidance).toContain("preserve its exemption/JIT method choice");
    expect(guidance).toContain("A Continue click asks a question only");
    expect(guidance).toContain(
      "Recheck current workflow eligibility after the member clicks"
    );
    expect(guidance).toContain(
      "never automatically change policy, grant access, open a draft, or rerun the query"
    );
  });
});
