// @vitest-environment node
import { describe, expect, test } from "vitest";
import { maskingDraftSetupInstruction } from "../maskingDraftWorkflow";
import { getSkill } from "./index";

describe("getSkill", () => {
  test("uses the complete permission-aware runtime semantic type and scope choices", () => {
    const guidance = getSkill({ name: "data-masking" });
    expect(guidance).toContain("ask_masking_semantic_type()");
    expect(guidance).toContain("all available semantic types");
    expect(guidance).toContain("Do not rank, filter, or deduplicate");
    expect(guidance).toContain("ask_masking_rule_scope");
    expect(guidance).not.toContain(
      'scope with ask_user(kind="choose", resourceKind="none")'
    );
  });

  test("prepares a new custom type before continuing the original rule after verified save", () => {
    const guidance = getSkill({ name: "data-masking" });
    expect(guidance).toContain("prepare_semantic_type_draft");
    expect(guidance).toContain(
      "Duplicate titles and equivalent masking configurations are allowed"
    );
    expect(guidance).toContain("Never accept, ask for, or reveal a hash salt");
    expect(guidance).toContain("Open semantic type draft");
    expect(guidance).toContain("verify its actual saved ID");
    expect(guidance).toContain("semanticTypeId");
    expect(guidance).toContain("resume the original confirmed rule scope");
    expect(guidance).not.toContain(
      "Create a custom semantic type only if another algorithm or configuration is needed"
    );
  });

  test("discovers masking setup guidance without requiring a masking page", () => {
    expect(getSkill({})).toContain("**data-masking**");
    expect(getSkill({})).toContain("step-by-step");
  });

  test("loads goal-driven masking guidance with optional setup paths", () => {
    const guidance = getSkill({ name: "data-masking" });

    expect(guidance).toContain("get_masking_overview");
    expect(guidance).toContain("workspace or project chat");
    expect(guidance).toContain("target audience");
    expect(guidance).toContain("desired visibility");
    expect(guidance).toContain("existing semantic type");
    expect(guidance).toContain("Classifications are optional");
    expect(guidance).toContain("global rule or a direct catalog assignment");
    expect(guidance).toContain("Exemptions are optional");
    expect(guidance).toContain("one actionable step at a time");
  });

  test("uses the dedicated runtime-owned project exemption scope question", () => {
    const guidance = getSkill({ name: "data-masking" });

    expect(guidance).toContain("call ask_masking_exemption_scope()");
    expect(guidance).toContain("The only optional argument is database");
    expect(guidance).toContain("Do not pass a question, options, or a default");
    expect(guidance).toContain("runtime supplies the localized scope question");
    expect(guidance).toContain("wait for Reply");
    expect(guidance).toContain("never auto-submit or write");
    expect(guidance).not.toContain('purpose="masking-exemption-scope"');
  });

  test("keeps exemption expiration and member questions on generic ask_user", () => {
    const guidance = getSkill({ name: "data-masking" });

    expect(guidance).toContain(
      "For expiration and member questions, use ask_user without purpose"
    );
    expect(guidance).toContain(
      "Do not use ask_masking_exemption_scope for ordinary questions"
    );
  });

  test("limits permission blockers to dependent steps and reports exact scope", () => {
    const guidance = getSkill({ name: "data-masking" });

    expect(guidance).toContain("allowed, denied, unknown, and needs-scope");
    expect(guidance).toContain("read and configure");
    expect(guidance).toContain("exact missing permissions and resource scope");
    expect(guidance).toContain("Block only that step and its dependent steps");
    expect(guidance).toContain("independent allowed steps");
    expect(guidance).toContain(
      "unavailable facet is not proof of permission denial"
    );
    expect(guidance).toContain("not a broad administrator role");
    expect(guidance).toContain("Re-read get_masking_overview");
  });

  test("does not offer direct catalog assignment as an override for a matching global rule", () => {
    const guidance = getSkill({ name: "data-masking" });

    expect(guidance).toContain("first matching global rule takes precedence");
    expect(guidance).toContain("when it supplies a semantic type ID");
    expect(guidance).toContain("direct catalog assignment cannot override it");
    expect(guidance).toContain(
      "fallback when global evaluation supplies no semantic type ID"
    );
    expect(guidance).toContain(
      "confirm that no global rule assigns a semantic type to the target columns"
    );
    expect(guidance).toContain(
      "If global rule evidence is unavailable, explain that the effect remains unverified"
    );
    expect(guidance).not.toContain(
      "explicit column semantic type can take precedence"
    );
  });

  test("marks setup ready only with verified permissions, features, and prerequisite configuration", () => {
    const guidance = getSkill({ name: "data-masking" });

    expect(guidance).toContain("Mark a setup step ready only when");
    expect(guidance).toContain(
      "applicable read and configure checks are allowed"
    );
    expect(guidance).toContain(
      "necessary current and prerequisite configuration facets are available"
    );
    expect(guidance).toContain("relevant feature is available");
    expect(guidance).toContain(
      "conditional requirements when their conditions apply"
    );
    expect(guidance).toContain(
      "missing permissions or an unavailable feature as blocked"
    );
    expect(guidance).toContain(
      "unknown permissions, features, or unreadable prerequisite state as unverified"
    );
    expect(guidance).toContain("view-only destination");
  });

  test("keeps saved policy changes member-owned and previews illustrative", () => {
    const guidance = getSkill({ name: "data-masking" });

    expect(guidance).toContain("Do not write saved masking policy");
    expect(guidance).toContain("Never modify masking rules");
    expect(guidance).toContain("catalog metadata");
    expect(guidance).toContain("Never query or read real row values");
    expect(guidance).toContain("synthetic values");
    expect(guidance).toContain("prefix and suffix cover the whole value");
    expect(guidance).toContain(
      "exact target returned by get_masking_overview in this turn"
    );
    expect(guidance).toContain("only after the member clicks");
    expect(guidance).toContain("does not prove a setting was saved");
  });

  test("offers prepared drafts without delegating runtime IDs to the model", () => {
    const guidance = getSkill({ name: "data-masking" });

    expect(guidance).toContain(
      "runtime supplies the latest eligible current-run masking draft action"
    );
    expect(guidance).toContain("do not need to repeat its opaque draft ID");
    expect(guidance).toContain("Never open, apply, or save automatically");
    expect(guidance).toContain(
      "After a resumed run or source change, read the current form and prepare again"
    );
    expect(guidance).toContain("Do not add a competing Open settings action");
    expect(guidance).not.toContain(
      'Return {"kind":"draft-masking-rule","draftId":"the exact returned draftId"}'
    );
  });

  test("makes ready new-rule setup draft-first instead of a manual-entry fallback", () => {
    const guidance = getSkill({ name: "data-masking" });

    expect(guidance).toContain(maskingDraftSetupInstruction);
    expect(guidance).toContain(
      "the normal next step is a prepared draft, not manual field-entry instructions"
    );
    expect(guidance).toContain(
      "Select Open rule draft to open the filled form"
    );
    expect(guidance).toContain(
      "Manual instructions only when explicitly requested"
    );
    expect(guidance).toContain(
      "Do not substitute manual instructions for an eligible prepared draft"
    );
    expect(guidance).toContain(
      "If preparation is blocked or unverified, explain the exact reason"
    );
    expect(guidance).not.toContain("prefer preparing its actual unsaved form");
  });

  test("distinguishes existing-rule review from new-rule draft refinement", () => {
    const guidance = getSkill({ name: "data-masking" });

    expect(guidance).toContain("inspect its mode before proposing changes");
    expect(guidance).toContain("EDIT is review-only for an existing rule");
    expect(guidance).toContain(
      "Never prepare or apply a full-rule replacement in EDIT"
    );
    expect(guidance).toContain("In CREATE mode only");
    expect(guidance).toContain("current unsaved values, not saved policy");
  });
});
