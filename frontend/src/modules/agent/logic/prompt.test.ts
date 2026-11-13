// @vitest-environment node
import { describe, expect, test } from "vitest";
import { buildSystemPrompt } from "./prompt";
import { getMaskingSkillGuidance, type MaskingSkillSection } from "./skills";

function buildPromptWithMaskingGuidance(
  pageContext: Parameters<typeof buildSystemPrompt>[0],
  sections: readonly MaskingSkillSection[] = ["overview"]
): string {
  return `${buildSystemPrompt(pageContext)}\n\n${getMaskingSkillGuidance(sections)}`;
}

test("composes permission-aware masking choices and custom type draft guidance", () => {
  const prompt = buildPromptWithMaskingGuidance(
    { path: "/semantic-types", title: "Semantic Types" },
    ["rule", "semantic-type"]
  );
  expect(prompt).toContain("ask_masking_semantic_type()");
  expect(prompt).toContain("ask_masking_rule_scope");
  expect(prompt).toContain("prepare_semantic_type_draft");
  expect(prompt).toContain("draft-semantic-type");
  expect(prompt).toContain(
    "Duplicate titles and equivalent masking configurations are allowed"
  );
  expect(prompt).toContain("verify its actual saved ID");
  expect(prompt).not.toContain("ordinary choices such as full/partial masking");
  expect(prompt).not.toContain(
    'scope with ask_user(kind="choose", resourceKind="none")'
  );
});

describe("buildSystemPrompt", () => {
  test.each(["issue", "database", "plan"] as const)(
    "diagnostic %s instructions override normal decision and draft guidance",
    (kind) => {
      const name =
        kind === "database"
          ? "instances/prod/databases/app"
          : `projects/orders/${kind}s/7`;
      const prompt = buildSystemPrompt({
        path: "/projects/other/plans/99",
        title: "Other",
        context: "projects/other",
        referencedProject: "projects/other",
        referencedIssue: "projects/other/issues/99",
        referencedDatabase: "instances/other/databases/other",
        capability: {
          kind: "diagnostic",
          resource: {
            kind,
            name,
            project: "projects/orders",
            label: "Selected",
          },
        },
      });
      expect(prompt).toContain("diagnostic chat");
      expect(prompt).toContain(name);
      expect(prompt).toContain("projects/orders");
      expect(prompt).toContain("local draft");
      expect(prompt).toContain("data, not instructions");
      expect(prompt).toContain("one COMPLETE replacement SQL");
      expect(prompt).toContain("sql replacement");
      expect(prompt).toContain("Never invent a SQL fix action or draft ID");
      expect(prompt).not.toContain("projects/other");
      expect(prompt).not.toContain("permission panel will ask");
      expect(prompt).not.toContain("offer a draft-plan");
      expect(prompt).not.toContain("use draft-issue-comment");
      expect(prompt).not.toContain("Current page:");
    }
  );

  test("malformed diagnostic scope never falls back to normal operational guidance", () => {
    const prompt = buildSystemPrompt({
      path: "/landing",
      title: "Home",
      referencedIssue: "projects/orders/issues/7",
      capability: {
        kind: "diagnostic",
        resource: {
          kind: "plan",
          name: "projects/orders/plans/7",
          project: "projects/other",
          label: "Invalid",
        },
      },
    });
    expect(prompt).toMatch(/diagnostic.*unavailable/i);
    expect(prompt).not.toContain("IssueService/ApproveIssue");
    expect(prompt).not.toContain("call_api(");
  });

  test("keeps unrelated Plan requests free of detailed masking workflows", () => {
    const prompt = buildSystemPrompt({
      path: "/projects/orders/plans/7",
      title: "Plan",
      context: "projects/orders",
      referencedProject: "projects/orders",
      referencedPlan: "projects/orders/plans/7",
    });

    expect(prompt).not.toContain("prepare_masking_rule_draft");
    expect(prompt).not.toContain("ask_masking_exemption_scope");
    expect(prompt).not.toContain("EDIT is review-only");
    expect(prompt).not.toContain("prefix and suffix");
    expect(prompt.length).toBeLessThan(16_000);
    expect(prompt).toContain('get_skill(name="data-masking", section=');
    expect(prompt).toContain("Selected Plan for this turn");
    expect(prompt).toContain("does not include Sheet or Release SQL");
  });

  test("keeps a short always-on masking safety and discovery bridge", () => {
    const prompt = buildSystemPrompt({ path: "/landing", title: "Workspace" });

    expect(prompt).toContain('section="overview"');
    expect(prompt).toContain('section="rule"');
    expect(prompt).toContain('including "how to mask ..."');
    expect(prompt).toContain('"exemption"');
    expect(prompt).toContain('"semantic-type"');
    expect(prompt).toContain("Never read or query real row values");
    expect(prompt).toContain("via APIs or DOM actions");
    expect(prompt).toContain("member-clicked typed drafts");
    expect(prompt).toContain("local unsaved forms");
  });

  test.each(["workspaces/default", "projects/orders"])(
    "distinguishes verified subscription restrictions from read failures in %s",
    (context) => {
      const prompt = buildSystemPrompt({
        path: "/landing",
        title: "Home",
        context,
      });

      expect(prompt).toContain("verified subscription evidence");
      expect(prompt).toContain("current plan and required feature/plan");
      expect(prompt).toContain(
        "Never infer a subscription limit from a permission error or failed read"
      );
      expect(prompt).toContain("could not be verified");
      expect(prompt).toContain("do not offer unusable actions");
      expect(prompt).not.toContain("Current subscription:");
    }
  );

  test("respects no-persistence SQL review despite SQL Editor auto-save", () => {
    const prompt = buildSystemPrompt({
      path: "/sql-editor/projects/orders",
      title: "SQL Editor",
      context: "projects/orders",
    });

    expect(prompt).toContain(
      "SQL Editor automatically creates or updates a Saved Query"
    );
    expect(prompt).toContain(
      "Do not promise that SQL entered there is unsaved or unpersisted"
    );
    expect(prompt).toContain(
      "If the member forbids saving or persistence, do not type SQL into SQL Editor"
    );
    expect(prompt).toContain("return the SQL inline in chat instead");
    expect(prompt).toContain("Do not equate not executed with not saved");
    expect(prompt).toContain(
      "The typed Plan draft handoff remains local and unsaved"
    );
  });

  test("uses code-owned exemption scope choices instead of model-generated alternatives", () => {
    const prompt = buildPromptWithMaskingGuidance(
      { path: "/landing", title: "Home", context: "projects/orders" },
      ["exemption"]
    );
    expect(prompt).toContain(
      "call ask_masking_exemption_scope({project?,database?})"
    );
    expect(prompt).toContain("pass its canonical name in a workspace chat");
    expect(prompt).toContain("Do not pass a question, options, or a default");
    expect(prompt).toContain("runtime supplies the localized scope question");
    expect(prompt).toContain("All databases in this project");
    expect(prompt).toContain("wait for Reply");
    expect(prompt).not.toContain('purpose="masking-exemption-scope"');
  });

  test("keeps exemption expiration and member questions on generic ask_user", () => {
    const prompt = buildPromptWithMaskingGuidance(
      { path: "/landing", title: "Home", context: "projects/orders" },
      ["exemption"]
    );

    expect(prompt).toContain(
      "For expiration and member questions, use ask_user without purpose"
    );
    expect(prompt).toContain(
      "Do not use ask_masking_exemption_scope for ordinary questions"
    );
    expect(prompt).toContain('ask_user(kind="choose", resourceKind="none")');
    expect(prompt).toContain(
      "Do not start with a free-text expiration question"
    );
    expect(prompt).toContain("Choose a date and time");
    expect(prompt).toContain("No expiration");
  });
  test("uses verified resource defaults for database clarification prefills", () => {
    const prompt = buildSystemPrompt({
      path: "/landing",
      title: "Home",
      context: "projects/orders",
    });

    expect(prompt).toContain(
      'ask_user(kind="input", resourceKind="database", defaultValue=canonicalName)'
    );
    expect(prompt).toContain("renders it as a bound @Database mention");
    expect(prompt).toContain(
      "A prefill is a suggestion, not a submitted answer or permission to execute"
    );
  });

  test("reviews current existing-rule fields without preparing a replacement", () => {
    const prompt = buildPromptWithMaskingGuidance(
      {
        path: "/settings/global-masking",
        title: "Global Masking",
        context: "workspaces/default",
        maskingPage: { kind: "global-masking" },
      },
      ["rule"]
    );

    expect(prompt).toContain("EDIT is review-only");
    expect(prompt).toContain("current unsaved values, not saved policy");
    expect(prompt).toContain(
      "Only CREATE supports full-rule preparation and Apply to draft"
    );
    expect(prompt).toContain(
      "Never prepare or apply a full-rule replacement in EDIT"
    );
  });

  test.each(["workspaces/default", "projects/orders"])(
    "requires a prepared draft for ready masking setup in %s",
    (context) => {
      const prompt = buildPromptWithMaskingGuidance(
        { path: "/landing", title: "Workspace", context },
        ["rule"]
      );

      expect(prompt).toContain(
        "MUST call prepare_masking_rule_draft before done"
      );
      expect(prompt).toContain(
        "Do not finish a ready setup with only manual steps or an Open settings action"
      );
      expect(prompt).toContain(
        "Do not ask for extra confirmation just to prepare"
      );
      expect(prompt).toContain(
        "use ask_masking_rule_scope for missing rule scope"
      );
      expect(prompt).toContain(
        "Conceptual or effect questions, explicitly requested manual-only instructions"
      );
    }
  );

  test("inspects the live page only for page-dependent requests", () => {
    const prompt = buildSystemPrompt({
      path: "/landing",
      title: "Workspace",
      context: "workspaces/default",
    });

    expect(prompt).toContain(
      "Call get_page_state when the request depends on the live page"
    );
    expect(prompt).toContain(
      "skip it for self-contained questions or casual conversation"
    );
    expect(prompt).not.toContain("Always call get_page_state first");
  });

  test("omits project-only workflow details from workspace chats", () => {
    const prompt = buildSystemPrompt({
      path: "/projects/other",
      title: "Other project",
      context: "workspaces/default",
    });

    expect(prompt).toContain(
      "call get_workspace_overview before recommending a next step"
    );
    expect(prompt).not.toContain("When get_project_overview confirms");
    expect(prompt).not.toContain("In a project-bound chat");
    expect(prompt).not.toContain("A historical failed Task Run");
  });

  test("uses the bounded instance overview for instance sync questions", () => {
    const prompt = buildSystemPrompt({
      path: "/instances/primary",
      title: "Primary instance",
      context: "workspaces/default",
      instanceName: "instances/primary",
      runMode: "read-only",
    });

    expect(prompt).toContain("call get_instance_overview");
    expect(prompt).toContain("reads one page");
    expect(prompt).toContain("visible to the member");
    expect(prompt).toContain(
      "never claim there are no out-of-sync databases across the entire instance"
    );
  });

  test("uses masking configuration evidence and synthetic-only previews on masking pages", () => {
    const prompt = buildPromptWithMaskingGuidance({
      path: "/settings/global-masking",
      title: "Masking",
      context: "workspaces/default",
      maskingPage: { kind: "global-masking" },
    });

    expect(prompt).toMatch(/call get_masking_overview/i);
    expect(prompt).toContain("synthetic values only");
    expect(prompt).toMatch(/never read or query real row values/i);
    expect(prompt).toContain(
      "Never change saved masking rules, semantic types, classifications, catalog metadata, or exemptions via APIs or DOM actions"
    );
    expect(prompt).toContain("prefix and suffix cover the whole value");
    expect(prompt).toContain("Inner Mask masks the interior");
    expect(prompt).toContain("Outer Mask masks prefix/suffix");
    expect(prompt).toContain("target user or role");
  });

  test.each(["workspaces/default", "projects/orders"])(
    "discovers permission-aware masking setup from ordinary %s chats",
    (context) => {
      const prompt = buildPromptWithMaskingGuidance({
        path: "/landing",
        title: "Workspace",
        context,
      });

      expect(prompt).toContain(
        'get_skill(name="data-masking", section="rule")'
      );
      expect(prompt).toMatch(/call get_masking_overview/i);
      expect(prompt).toContain("even from a workspace or project chat");
      expect(prompt).toContain(
        "before settings/access claims, selecting an existing semantic type, or setup"
      );
      expect(prompt).toContain("exact missing permissions and resource scope");
      expect(prompt).toContain("Block only that step and its dependent steps");
      expect(prompt).toContain("continue independent allowed steps");
      expect(prompt).toContain("Unknown/needs-scope is unverified");
      expect(prompt).toContain(
        "unavailable facet is not proof of permission denial"
      );
      expect(prompt).toContain("Check read and configure separately");
      expect(prompt).toContain("Re-read get_masking_overview");
      expect(prompt).toContain("Never read or query real row values");
      expect(prompt).not.toContain(
        "This is a data-masking assistance request on the"
      );
    }
  );

  test("keeps read-only masking discovery available for landing starters", () => {
    const prompt = buildPromptWithMaskingGuidance({
      path: "/projects/orders",
      title: "Orders",
      context: "projects/orders",
      runMode: "read-only",
    });

    expect(prompt).toContain('get_skill(name="data-masking", section="rule")');
    expect(prompt).toMatch(/call get_masking_overview/i);
    expect(prompt).toContain("Never change saved masking rules");
    expect(prompt).toContain("catalog metadata");
  });

  test("keeps project guidance when the selected project differs from the workspace chat", () => {
    const prompt = buildSystemPrompt({
      path: "/landing",
      title: "Workspace",
      context: "workspaces/default",
      referencedProject: "projects/orders",
    });

    expect(prompt).toContain("When get_project_overview confirms");
    expect(prompt).toContain(
      "call get_project_governance_overview before describing"
    );
    expect(prompt).not.toContain(
      "In a workspace-bound chat, call get_workspace_overview"
    );
  });

  test("does not repeat count workflows for a selected project", () => {
    const prompt = buildSystemPrompt({
      path: "/landing",
      title: "Workspace",
      context: "workspaces/default",
      referencedProject: "projects/orders",
    });

    expect(prompt.match(/precise instance filter/g)).toHaveLength(1);
  });

  test.each([
    { runMode: "read-only" as const },
    { referencedProject: "projects/orders" },
  ])("omits page-action workflows for restricted turns: %j", (restriction) => {
    const prompt = buildSystemPrompt({
      path: "/projects/orders",
      title: "Orders",
      context: "projects/orders",
      ...restriction,
    });

    expect(prompt).not.toContain("Confirm destructive DOM actions");
    expect(prompt).not.toContain("DOM interaction workflow");
    expect(prompt).not.toContain("DOM-first");
    expect(prompt).not.toContain("navigate(list=true");
    expect(prompt).toContain("2-3 distinct, contextual Continue questions");
  });

  test("requires fresh DOM inspection before interacting with the page", () => {
    const prompt = buildSystemPrompt({
      path: "/projects/orders",
      title: "Orders",
      context: "projects/orders",
    });

    expect(prompt).toContain(
      'Before interacting with the page, call get_page_state(mode="dom") for fresh element refs'
    );
    expect(prompt).toContain("Confirm destructive DOM actions");
  });

  test("requests useful English follow-up questions that send on click", () => {
    const prompt = buildSystemPrompt({ path: "/landing", title: "Workspace" });
    expect(prompt).toContain(
      "at most one primary action: Open, draft-plan, or draft-issue-comment"
    );
    expect(prompt).toContain("current-turn overview evidence");
    expect(prompt).toContain("2-3 distinct, contextual Continue questions");
    expect(prompt).toContain("For a successful answer");
    expect(prompt).toContain("English");
    expect(prompt).toContain("Do not add filler questions");
    expect(prompt).toContain("sends it as a new user message");
    expect(prompt).toContain("does not itself approve or execute a change");
  });

  test("offers only evidenced exact Issue links without taking a review action", () => {
    const prompt = buildSystemPrompt({
      path: "/projects/orders/issues/123",
      title: "Issue",
      context: "projects/orders",
      referencedProject: "projects/orders",
      referencedIssue: "projects/orders/issues/123",
    });
    expect(prompt).toContain("open-issue");
    expect(prompt).toContain("review-issue-plan");
    expect(prompt).toContain("view-failed-task");
    expect(prompt).toContain("current failed Task");
    expect(prompt).toContain("never approves, retries, or executes");
  });

  test("uses empty-project setup evidence in the same priority as Project Home", () => {
    const prompt = buildSystemPrompt({
      path: "/projects/sample",
      title: "Sample",
      context: "projects/sample",
    });
    expect(prompt).toContain("transfer an eligible unassigned database first");
    expect(prompt).toContain("connect an instance directly");
    expect(prompt).toContain("create a database on an available instance");
    expect(prompt).toContain("transfer-project-databases");
    expect(prompt).toContain("specific setup request in any active project");
  });
  test("requests fresh validation only for database choices", () => {
    const prompt = buildSystemPrompt({ path: "/landing", title: "Workspace" });
    expect(prompt).toContain("resourceKind");
    expect(prompt).toContain("canonical option values");
    expect(prompt).toContain("Ask which Project first");
    expect(prompt).toContain('resourceKind="none"');
    expect(prompt).toContain(
      "classifies option values, not the question context"
    );
    expect(prompt).toContain('retry kind="choose"');
    expect(prompt).not.toContain("ask a free-form question instead");
    expect(prompt).not.toContain('resourceKind="environment"');
  });
  test("requires a failed completion when an API request is rejected", () => {
    const prompt = buildSystemPrompt({
      path: "/projects/orders/issues",
      title: "Issues",
      context: "projects/orders",
    });

    expect(prompt).toContain(
      "If call_api returns an error or HTTP status 400 or higher"
    );
    expect(prompt).toContain("done({ text, success: false, actions: [] })");
    expect(prompt).toContain("Do not claim the query or change succeeded");
    expect(prompt).toContain("Query result contains permissionDenied");
  });

  test("uses a selected project for one turn without narrowing normal API calls", () => {
    const prompt = buildSystemPrompt({
      path: "/projects/original",
      title: "Original project",
      context: "projects/original",
      referencedProject: "projects/referenced",
      runMode: "read-only",
    });

    expect(prompt).toContain("Chat context: projects/original");
    expect(prompt).toContain(
      "Referenced project for this turn: projects/referenced"
    );
    expect(prompt).toContain(
      "bounded project tools target the referenced project"
    );
    expect(prompt).toContain("This is not a chat context change");
    expect(prompt).toContain("matching the referenced project");
    expect(prompt).toContain("search_api");
    expect(prompt).toContain("call_api");
    expect(prompt).toContain("existing permissions and may change data");
    expect(prompt).toContain("only make changes when the user explicitly asks");
    expect(prompt).not.toContain("Do not navigate, call APIs");
    expect(prompt).not.toContain("Either works for mutations");
  });

  test("uses a selected database in its owning project for only this turn", () => {
    const prompt = buildSystemPrompt({
      path: "/projects/orders",
      title: "Orders",
      context: "projects/orders",
      referencedProject: "projects/billing",
      referencedDatabase: "instances/prod/databases/billing",
      runMode: "read-only",
    });

    expect(prompt).toContain(
      "Selected database for this turn: instances/prod/databases/billing"
    );
    expect(prompt).toContain("owning project must match projects/billing");
    expect(prompt).toContain("fresh get_database_overview result");
    expect(prompt).toContain("does not change the chat context");
    expect(prompt).toContain(
      "only make changes when the member explicitly asks"
    );
    expect(prompt).toContain("search_api");
    expect(prompt).toContain("call_api");
    expect(prompt).toContain(
      'use get_database_count with scope "referenced-instance"'
    );
    expect(prompt).toContain(
      "A workspace instance may span projects; a project instance remains in its owning project"
    );
    expect(prompt).toContain(
      "For filtered counts in a Workspace or Project, use search_api and call_api SearchDatabases with a project filter for Project scope"
    );
    expect(prompt).toContain("inspect one page");
    expect(prompt).toContain(
      'If partial is true, say "at least N" rather than claiming an exact total'
    );
    expect(prompt).not.toContain("Referenced project for this turn");
  });

  test("grounds a selected Plan in a fresh overview for only this turn", () => {
    const prompt = buildSystemPrompt({
      path: "/projects/orders/plans",
      title: "Plans",
      context: "projects/orders",
      referencedProject: "projects/billing",
      referencedPlan: "projects/billing/plans/7",
      runMode: "read-only",
    });

    expect(prompt).toContain(
      "Selected Plan for this turn: projects/billing/plans/7"
    );
    expect(prompt).toContain("fresh get_plan_overview result");
    expect(prompt).toContain("owning project must match projects/billing");
    expect(prompt).toContain("does not include Sheet or Release SQL");
    expect(prompt).toContain("corresponding Sheet or Release read API");
    expect(prompt).toContain("does not change the chat context");
    expect(prompt).toContain(
      "only make changes when the member explicitly asks"
    );
    expect(prompt).not.toContain("Referenced project for this turn");
  });

  test("offers an unsaved Plan draft only for generated DDL grounded in this turn's selected database", () => {
    const selectedPrompt = buildSystemPrompt({
      path: "/projects/orders",
      title: "Orders",
      context: "projects/orders",
      referencedProject: "projects/billing",
      referencedDatabase: "instances/prod/databases/billing",
    });
    const unselectedPrompt = buildSystemPrompt({
      path: "/projects/orders",
      title: "Orders",
      context: "projects/orders",
    });

    expect(selectedPrompt).toContain("draft-plan");
    expect(selectedPrompt).toContain('"create a table"');
    expect(selectedPrompt).toContain("DatabaseService/GetDatabaseSchema");
    expect(selectedPrompt).toContain("instances/prod/databases/billing/schema");
    expect(selectedPrompt).toContain("generated DDL");
    expect(selectedPrompt).toContain("exactly one fenced `sql` code block");
    expect(selectedPrompt).toContain("opens a populated, unsaved Plan");
    expect(selectedPrompt).toContain("never persists or executes SQL");
    expect(selectedPrompt).toContain(
      "at most one primary action: Open, draft-plan, or draft-issue-comment"
    );
    expect(unselectedPrompt).not.toContain("generated DDL");
  });

  test("leaves masking draft actions to the current-run runtime handoff", () => {
    const prompt = buildPromptWithMaskingGuidance(
      { path: "/landing", title: "Workspace" },
      ["rule"]
    );

    expect(prompt).toContain(
      "runtime supplies the latest eligible current-run masking draft action"
    );
    expect(prompt).toContain("do not need to repeat its opaque draft ID");
    expect(prompt).toContain("Never open, apply, or save automatically");
    expect(prompt).toContain(
      "After a resumed run or source change, read the current form and prepare again"
    );
    expect(prompt).not.toContain(
      'return {kind:"draft-masking-rule",draftId:<exact returned ID>}'
    );
  });

  test("grounds a selected issue and its approval consequence in fresh state", () => {
    const prompt = buildSystemPrompt({
      path: "/projects/orders",
      title: "Orders",
      context: "projects/orders",
      referencedProject: "projects/billing",
      referencedIssue: "projects/billing/issues/42",
      runMode: "read-only",
    });

    expect(prompt).toContain(
      "Selected Issue for this turn: projects/billing/issues/42"
    );
    expect(prompt).toContain("fresh get_issue_overview result");
    expect(prompt).toContain("does not change the chat context");
    expect(prompt).toContain("IssueService/ApproveIssue");
    expect(prompt).toContain("IssueService/RejectIssue");
    expect(prompt).toContain("may queue rollout creation");
    expect(prompt).toContain("does not mean tasks have executed");
    expect(prompt).toContain("may immediately grant access or a project role");
    expect(prompt).toContain(
      "After an approved decision request, re-read the Issue"
    );
    expect(prompt).not.toContain("Referenced project for this turn");
  });

  test("offers only a reviewed and unsent Issue comment draft", () => {
    const prompt = buildSystemPrompt({
      path: "/projects/orders/issues/42",
      title: "Issue",
      context: "projects/orders",
      referencedProject: "projects/orders",
      referencedIssue: "projects/orders/issues/42",
    });
    expect(prompt).toContain("security review of an Issue");
    expect(prompt).toContain("draft-issue-comment");
    expect(prompt).toContain("exact observed Issue name");
    expect(prompt).toContain("unsent editor draft");
    expect(prompt).toContain("Do not include speculative findings");
  });

  test("names the chat context separately from the live page", () => {
    const prompt = buildSystemPrompt({
      path: "/projects/other/issues",
      title: "Other project",
      context: "projects/orders",
    });

    expect(prompt).toContain("Chat context: projects/orders");
    expect(prompt).toContain("Current page: /projects/other/issues");
    expect(prompt).toContain("Navigation does not change the chat context");
  });

  test("accepts a workspace resource as the chat context", () => {
    expect(
      buildSystemPrompt({
        path: "/landing",
        title: "Workspace",
        context: "workspaces/demo",
      })
    ).toContain("Chat context: workspaces/demo");
  });

  test("does not carry a past project reference into an unqualified workspace follow-up", () => {
    const prompt = buildSystemPrompt({
      path: "/landing",
      title: "Workspace",
      context: "workspaces/demo",
    });

    expect(prompt).toContain(
      "A selected resource in an earlier user message does not scope this turn"
    );
    expect(prompt).toContain(
      "Do not infer a project from prior conversation when the current message has no project reference or explicit project name"
    );
    expect(prompt).toContain(
      "For an unqualified list or count in a workspace-bound chat, use the whole chat workspace"
    );
    expect(prompt).toContain(
      "For an unfiltered active-database count in the chat Workspace or Project, or the referenced Project for this turn, use get_database_count with its default scope"
    );
    expect(prompt).toContain(
      "For filtered counts in a Workspace or Project, use search_api and call_api SearchDatabases"
    );
    expect(prompt).toContain(
      "For other instance scopes, use search_api and call_api SearchDatabases with the precise instance filter"
    );
    expect(prompt).toContain(
      "ask which project only when the member requests a project-specific result without identifying one"
    );
    expect(prompt).toContain(
      'If partial is true, say "at least N" rather than claiming an exact total'
    );
    expect(prompt).toContain(
      "Do not automatically follow pagination tokens to count all resources"
    );
    expect(prompt).not.toContain("follow every page");
    expect(prompt).not.toContain(
      "if it is a workspace and the request requires a specific project, ask which project"
    );
  });

  test("uses a compact count for both chat-bound and referenced projects", () => {
    const projectPrompt = buildSystemPrompt({
      path: "/projects/orders",
      title: "Orders",
      context: "projects/orders",
    });
    const referencePrompt = buildSystemPrompt({
      path: "/landing",
      title: "Workspace",
      context: "workspaces/demo",
      referencedProject: "projects/orders",
    });

    expect(projectPrompt).toContain(
      "For an unfiltered active-database count in the chat Workspace or Project, or the referenced Project for this turn, use get_database_count with its default scope"
    );
    expect(referencePrompt).toContain(
      "For an unfiltered active-database count in the chat Workspace or Project, or the referenced Project for this turn, use get_database_count with its default scope"
    );
    expect(referencePrompt).toContain("referenced Project for this turn");
    expect(referencePrompt).toContain(
      "For filtered counts in a Workspace or Project, use search_api and call_api SearchDatabases with a project filter for Project scope"
    );
  });

  test("teaches the ref-based DOM workflow", () => {
    const prompt = buildSystemPrompt({
      path: "/issues/123",
      title: "Issue 123",
      role: "DBA",
    });

    expect(prompt).toContain(
      'DOM interaction workflow: get_page_state(mode="dom") → read element refs like [e1] in the DOM tree → dom_action(type, ref, value).'
    );
    expect(prompt).not.toContain("read element indices");
    expect(prompt).not.toContain("dom_action(type, index, value)");
  });

  test("keeps a read-only landing starter on its current page", () => {
    const prompt = buildSystemPrompt({
      path: "/projects/orders",
      title: "Order service",
      context: "projects/orders",
      runMode: "read-only",
    });

    expect(prompt).toContain("Stay on the current page");
    expect(prompt).toContain("Do not navigate");
    expect(prompt).toContain("Do not navigate, call APIs");
    expect(prompt).toContain("get_project_open_issues");
    expect(prompt).toContain("get_project_overview");
    expect(prompt).toContain("get_project_governance_overview");
    expect(prompt).toContain("get_issue_overview");
    expect(prompt).toContain("get_database_overview");
    expect(prompt).toContain("get_project_recent_activity");
    expect(prompt).toContain("get_workspace_recent_activity");
  });

  test("asks the Agent to use a project overview for a current-state question", () => {
    const prompt = buildSystemPrompt({
      path: "/projects/orders",
      title: "Order service",
      context: "projects/orders",
      runMode: "read-only",
    });

    expect(prompt).toContain(
      "call get_project_overview before recommending a next step"
    );
    expect(prompt).toContain(
      "answer the member's actual question directly, not with a catalog of capabilities"
    );
    expect(prompt).toContain(
      "recommend at most one next step when the available evidence supports it"
    );
    expect(prompt).not.toContain("<project_readiness_evidence>");
  });

  test.each([
    ["project", "projects/orders"],
    ["workspace", "workspaces/default"],
  ])("requires complete %s facets before claiming absence", (_, context) => {
    const prompt = buildSystemPrompt({
      path: "/landing",
      title: "Home",
      context,
    });

    expect(prompt).toContain(
      "Do not claim a resource or issue is absent unless its facet is available, partial is false, and it is empty"
    );
    expect(prompt).toContain(
      "A partial or unavailable facet cannot establish absence"
    );
  });

  test("asks the Agent to inspect project governance before describing it", () => {
    const prompt = buildSystemPrompt({
      path: "/projects/orders",
      title: "Order service",
      context: "projects/orders",
      runMode: "read-only",
    });

    expect(prompt).toContain(
      "call get_project_governance_overview before describing how Database Changes are reviewed"
    );
    expect(prompt).toContain("conditions are evaluated per Database Change");
  });

  test("asks the Agent to inspect a selected Issue before describing its state", () => {
    const prompt = buildSystemPrompt({
      path: "/projects/orders",
      title: "Order service",
      context: "projects/orders",
      runMode: "read-only",
    });

    expect(prompt).toContain(
      "call get_issue_overview before explaining a specific Issue's approval"
    );
    expect(prompt).toContain(
      "not_started for a review run means no run is recorded; it does not prove that review is configured or applicable"
    );
    expect(prompt).toContain("matching the chat project");
  });

  test("asks the Agent to inspect a selected database before describing its state", () => {
    const prompt = buildSystemPrompt({
      path: "/projects/orders",
      title: "Order service",
      context: "projects/orders",
      runMode: "read-only",
    });

    expect(prompt).toContain(
      "call get_database_overview before explaining a specific database's current state"
    );
    expect(prompt).toContain("DML prior-backup readiness");
    expect(prompt).toContain(
      "does not establish that an existing recoverable backup exists"
    );
  });

  test("asks the Agent to use a workspace overview without recent activity", () => {
    const prompt = buildSystemPrompt({
      path: "/landing",
      title: "Workspace",
      context: "workspaces/default",
      runMode: "read-only",
    });

    expect(prompt).toContain(
      "call get_workspace_overview before recommending a next step"
    );
    expect(prompt).toContain(
      "Use get_workspace_recent_activity only when the member asks what changed recently"
    );
  });
});
