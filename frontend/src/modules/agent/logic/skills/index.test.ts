// @vitest-environment node
import { describe, expect, test } from "vitest";
import { maskingDraftSetupInstruction } from "../maskingDraftWorkflow";
import { skill as databaseChangeSkill } from "./databaseChange";
import { skill as grantPermissionSkill } from "./grantPermission";
import {
  getMaskingSkillGuidance,
  getSkill,
  type MaskingSkillSection,
  resolveSkill,
} from "./index";
import { skill as querySkill } from "./query";

describe("resolveSkill", () => {
  test("defaults masking loads to a bounded discoverable overview", () => {
    const result = resolveSkill({ name: "data-masking" });
    expect(result).toMatchObject({
      kind: "loaded",
      name: "data-masking",
      section: "overview",
    });
    if (result.kind !== "loaded") throw new Error("Expected loaded skill");
    expect(result.content.length).toBeLessThanOrEqual(8000);
    expect(result.content).toContain('section="rule"');
    expect(result.content).toContain('section="exemption"');
    expect(result.content).toContain('section="semantic-type"');
    expect(result.content).not.toContain(maskingDraftSetupInstruction);
    expect(getSkill({ name: "data-masking" })).toBe(result.content);
  });

  test("routes actionable masking questions from overview into the draft workflow", () => {
    const guidance = getSkill({ name: "data-masking" });

    expect(guidance).toContain('including "how to mask ..."');
    expect(guidance).toContain(
      'get_skill(name="data-masking", section="rule")'
    );
    expect(guidance).toContain(
      "If eligible, missing scope/parameters require clarification before done"
    );
    expect(guidance).toContain(
      "Conceptual/effect-only questions can remain explanatory"
    );
  });

  test("resumes setup from fresh entitlement without requiring catalog scope first", () => {
    const guidance = getSkill({ name: "data-masking" });

    expect(guidance).toContain(
      "This turn's refreshed evidence supersedes historical blockers"
    );
    expect(guidance).toContain(
      "a changed plan is not proof the earlier evidence was wrong"
    );
    expect(guidance).toContain(
      "catalogMasking scope-unverified must not block global-rule scope questions"
    );
    expect(guidance).toContain(
      "When eligibility becomes available, continue setup"
    );
  });

  test.each(["rule", "exemption", "semantic-type"] as const)(
    "loads %s with overview safety guidance exactly once",
    (section) => {
      const result = resolveSkill({ name: "data-masking", section });
      expect(result).toMatchObject({ kind: "loaded", section });
      if (result.kind !== "loaded") throw new Error("Expected loaded skill");
      expect(result.content).toBe(getMaskingSkillGuidance([section]));
      expect(result.content.match(/# Data Masking Overview/g)).toHaveLength(1);
      expect(result.content).toContain("Never query or read real row values");
    }
  );

  test("deduplicates dependency closure in registry order", () => {
    expect(getMaskingSkillGuidance([])).toBe("");
    expect(
      getMaskingSkillGuidance(["exemption", "rule", "overview", "rule"])
    ).toBe(getMaskingSkillGuidance(["rule", "exemption"]));
    const all = getMaskingSkillGuidance(["semantic-type", "rule", "exemption"]);
    for (const heading of [
      "# Data Masking Overview",
      "## Global Masking Rules",
      "## Project Masking Exemptions",
      "## Semantic Type Drafts",
    ]) {
      expect(all.split(heading)).toHaveLength(2);
    }
    expect(all.split(maskingDraftSetupInstruction)).toHaveLength(2);
  });

  test.each([querySkill, databaseChangeSkill, grantPermissionSkill])(
    "preserves $name content without masking activation",
    (skill) => {
      expect(resolveSkill({ name: skill.name })).toEqual({
        kind: "loaded",
        name: skill.name,
        content: skill.content,
      });
      expect(getSkill({ name: skill.name })).toBe(skill.content);
      expect(
        resolveSkill({ name: skill.name, section: "overview" })
      ).toMatchObject({ kind: "error" });
    }
  );

  test("returns typed list and input errors without loading guidance", () => {
    expect(resolveSkill({})).toMatchObject({ kind: "listed" });
    for (const args of [
      { name: "missing" },
      { section: "rule" as const },
      { name: "data-masking", section: "missing" as MaskingSkillSection },
      { name: "data-masking", section: "toString" as MaskingSkillSection },
      {
        name: "data-masking",
        section: null as unknown as MaskingSkillSection,
      },
      {
        name: "data-masking",
        section: ["rule"] as unknown as MaskingSkillSection,
      },
    ]) {
      const result = resolveSkill(args);
      expect(result).toMatchObject({ kind: "error" });
      if (result.kind !== "error") throw new Error("Expected error");
      expect(getSkill(args)).toBe(result.message);
    }
  });
});

describe("getSkill", () => {
  test("requires member-selected future expiration instead of a fixed date", () => {
    const guidance = getSkill({ name: "grant-permission" });
    expect(guidance).toContain("explicitly selected by the member");
    expect(guidance).toContain("Never invent a duration or expiration.");
    expect(
      guidance.match(/timestamp\('\{member-selected-future-instant\}'\)/g)
    ).toHaveLength(2);
    expect(guidance).not.toMatch(/timestamp\('\d{4}-\d{2}-\d{2}T/);
  });

  test("keeps specialized guidance out of unrelated section loads", () => {
    const overview = getSkill({ name: "data-masking" });
    const rule = getSkill({ name: "data-masking", section: "rule" });
    const exemption = getSkill({ name: "data-masking", section: "exemption" });
    const semanticType = getSkill({
      name: "data-masking",
      section: "semantic-type",
    });
    expect(overview).not.toContain("## Global Masking Rules");
    expect(rule).not.toContain("## Project Masking Exemptions");
    expect(rule).not.toContain("## Semantic Type Drafts");
    expect(exemption).not.toContain("## Global Masking Rules");
    expect(semanticType).not.toContain("## Project Masking Exemptions");
  });

  test("keeps rule selectors, batch saves, and source revisions bounded", () => {
    const guidance = getSkill({ name: "data-masking", section: "rule" });
    for (const invariant of [
      "selector-compatible CEL",
      "never lowerAscii(), transformed receivers, arbitrary functions",
      "Runtime independently adds verified project/database constraints",
      "prepare ALL confirmed requirements together in one batch",
      "one successful save is not evidence all requested rules saved",
      "current editor session/revision",
      "changed fields require fresh review, not overwrite",
      "Never navigate away from an unrelated editor",
      "Only CREATE supports full-rule preparation and Apply to draft",
    ]) {
      expect(guidance).toContain(invariant);
    }
  });

  test("retains named columns across masking scope and method clarification", () => {
    const guidance = getSkill({ name: "data-masking", section: "rule" });

    expect(guidance).toContain(
      '"mask email" defaults to resource.column_name == "email"'
    );
    expect(guidance).toContain(
      "Preserve named columns through scope/type questions"
    );
    expect(guidance).toContain(
      "Do not ask exact-name versus another pattern merely because broader matches are possible"
    );
    expect(guidance).toContain(
      "Clarify only genuinely ambiguous categories or conflicting column identities"
    );
    expect(guidance).toContain(
      "Never infer suffix, case-insensitive, or semantic-category matching"
    );
  });

  test("keeps exemption member, expiry, selector, and unsaved-form boundaries", () => {
    const guidance = getSkill({ name: "data-masking", section: "exemption" });
    for (const invariant of [
      "Use self for the current authenticated member",
      "Never offer workspace scope for a project exemption",
      "Never automatically modify, narrow, revoke, or replace an existing exemption",
      "does not grant query access by itself",
      "Do not copy global-rule CEL",
      "fresh currentTime (UTC ISO)",
      "expirationTimestamp",
      "IANA time zones",
      "daylight-saving ambiguous times",
      "source identity, and revision",
      "only Grant Exemption saves",
    ]) {
      expect(guidance).toContain(invariant);
    }
  });

  test("keeps new semantic types synthetic, exact, and separately saved", () => {
    const guidance = getSkill({
      name: "data-masking",
      section: "semantic-type",
    });
    for (const invariant of [
      "missing requested type must not be replaced",
      "md5-mask has no salt argument",
      "non-overlapping slices",
      "discriminated plain algorithm shape",
      "never replaces an active create/edit form or saves",
      "Do not prepare a rule referencing the new type before its save is verified",
      "Creation does not assign the type to columns",
    ]) {
      expect(guidance).toContain(invariant);
    }
  });

  test("uses the complete permission-aware runtime semantic type and scope choices", () => {
    const guidance = getSkill({ name: "data-masking", section: "rule" });
    expect(guidance).toContain("ask_masking_semantic_type()");
    expect(guidance).toContain("all available semantic types");
    expect(guidance).toContain("Do not rank, filter, or deduplicate");
    expect(guidance).toContain("ask_masking_rule_scope");
    expect(guidance).not.toContain(
      'scope with ask_user(kind="choose", resourceKind="none")'
    );
  });

  test("prepares a new custom type before continuing the original rule after verified save", () => {
    const guidance = getSkill({
      name: "data-masking",
      section: "semantic-type",
    });
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
    const guidance = getSkill({ name: "data-masking", section: "exemption" });

    expect(guidance).toContain("call ask_masking_exemption_scope()");
    expect(guidance).toContain("The only optional argument is database");
    expect(guidance).toContain("Do not pass a question, options, or a default");
    expect(guidance).toContain("runtime supplies the localized scope question");
    expect(guidance).toContain("wait for Reply");
    expect(guidance).toContain("never auto-submit or write");
    expect(guidance).not.toContain('purpose="masking-exemption-scope"');
  });

  test("keeps exemption expiration and member questions on generic ask_user", () => {
    const guidance = getSkill({ name: "data-masking", section: "exemption" });

    expect(guidance).toContain(
      "For expiration and member questions, use ask_user without purpose"
    );
    expect(guidance).toContain(
      "Do not use ask_masking_exemption_scope for ordinary questions"
    );
  });

  test("offers expiration choices before asking for a custom date and reuses confirmed expiry", () => {
    const guidance = getSkill({ name: "data-masking", section: "exemption" });

    expect(guidance).toContain('ask_user(kind="choose", resourceKind="none")');
    for (const label of [
      "In 1 hour",
      "In 24 hours",
      "Choose a date and time",
      "No expiration",
    ]) {
      expect(guidance).toContain(label);
    }
    expect(guidance).toContain("Leave all options unselected");
    expect(guidance).toContain(
      "If the member selects the custom date/time option"
    );
    expect(guidance).toContain('ask_user(kind="input", resourceKind="none")');
    expect(guidance).toContain("Reuse an already explicit expiration");
    expect(guidance).toContain("fresh currentTime");
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

  test.each(["overview", "rule", "exemption", "semantic-type"] as const)(
    "includes subscription evidence and blocked handoff guidance for %s",
    (section) => {
      const guidance = getMaskingSkillGuidance([section]);

      expect(guidance).toContain("setup.subscription");
      expect(guidance).toContain("setup.featureRequirements");
      expect(guidance).toContain("planTitle");
      expect(guidance).toContain("minimumPlanTitle");
      expect(guidance).toContain("plan-does-not-include-feature");
      expect(guidance).toContain("subscription-expired");
      expect(guidance).toContain("instance-license-required");
      expect(guidance).toContain("subscription-unverified");
      expect(guidance).toContain("scope-unverified needs scope, not upgrade");
      expect(guidance).toContain("do not ask dependent setup questions");
      expect(guidance).toContain("do not offer dependent draft actions");
      expect(guidance).toContain("Never assume trial eligibility");
    }
  );

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
    const guidance = getSkill({ name: "data-masking", section: "rule" });

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
    const guidance = getSkill({ name: "data-masking", section: "rule" });

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
    const guidance = getSkill({ name: "data-masking", section: "rule" });

    expect(guidance).toContain("inspect its mode before proposing changes");
    expect(guidance).toContain("EDIT is review-only for an existing rule");
    expect(guidance).toContain(
      "Never prepare or apply a full-rule replacement in EDIT"
    );
    expect(guidance).toContain("In CREATE mode only");
    expect(guidance).toContain("current unsaved values, not saved policy");
  });
});
