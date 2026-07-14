import { readFileSync } from "node:fs";
import { join } from "node:path";
import { describe, expect, test } from "vitest";

const settingsDir = import.meta.dirname;
const componentsDir = join(settingsDir, "../../components");

const workspaceListPages = [
  "AuditLogPage.tsx",
  "DatabasesPage.tsx",
  "GroupsPage.tsx",
  "InstancesPage.tsx",
  "MembersPage.tsx",
  "ProjectsPage.tsx",
  "SemanticTypesPage.tsx",
  "ServiceAccountsPage.tsx",
  "UsersPage.tsx",
  "WorkloadIdentitiesPage.tsx",
] as const;

const workspaceInfoPages = [
  "RolesPage.tsx",
  "IDPsPage.tsx",
  "IDPDetailPage.tsx",
  "SQLReviewPage.tsx",
  "RiskAssessmentPage.tsx",
  "CustomApprovalPage.tsx",
  "DataClassificationPage.tsx",
  "GlobalMaskingPage.tsx",
  "SemanticTypesPage.tsx",
  "IMPage.tsx",
] as const;

const workspaceRhythmPages = [
  "general/GeneralPage.tsx",
  "SubscriptionPage.tsx",
  "CustomApprovalPage.tsx",
] as const;

const adaptiveWorkspaceListPages = [
  "MembersPage.tsx",
  "ServiceAccountsPage.tsx",
  "WorkloadIdentitiesPage.tsx",
] as const;

const flushWorkspaceListPages = [
  "DatabasesPage.tsx",
  "InstancesPage.tsx",
  "ProjectsPage.tsx",
] as const;

function readSettingsPage(
  file:
    | (typeof workspaceListPages)[number]
    | (typeof workspaceInfoPages)[number]
    | (typeof workspaceRhythmPages)[number]
) {
  return readFileSync(join(settingsDir, file), "utf8");
}

describe("workspace list page layout", () => {
  test("supports explicit workspace shell padding modes", () => {
    const source = readFileSync(
      join(componentsDir, "WorkspacePageLayout.tsx"),
      "utf8"
    );

    expect(source).toContain("paddingBlock: 16");
    expect(source).toContain("rowGap: 16");
    expect(source).toContain('overflowX: "clip"');
    expect(source).not.toContain('overflowX: "hidden"');
    expect(source).toContain("marginInline: 8");
    expect(source).toContain('padding?: "page" | "flush"');
    expect(source).toContain('padding = "page"');
    expect(source).toContain("pagePadding");
    expect(source).toContain("paddingInline: 16");
    expect(source).toContain('<Alert role="note"');
    expect(source).not.toContain("paddingBottom: 8");
  });

  test("uses the shared workspace layout shell instead of page-local padding", () => {
    for (const file of workspaceListPages) {
      const source = readSettingsPage(file);

      expect(source, file).toContain("@/react/components/WorkspacePageLayout");
      if (
        adaptiveWorkspaceListPages.includes(
          file as (typeof adaptiveWorkspaceListPages)[number]
        )
      ) {
        expect(source, file).toContain(
          "const PageLayout = projectName ? ProjectPageLayout : WorkspacePageLayout"
        );
        expect(source, file).toContain("<PageLayout>");
      } else {
        expect(source, file).toContain("<WorkspacePageLayout");
      }
      if (
        flushWorkspaceListPages.includes(
          file as (typeof flushWorkspaceListPages)[number]
        )
      ) {
        expect(source, file).toContain('padding="flush"');
        expect(source, file).toContain('<WorkspacePageToolbar className="px-4');
      }
      expect(source, file).not.toContain(
        'className="w-full px-4 overflow-x-hidden flex flex-col pt-2 pb-4"'
      );
      expect(source, file).not.toContain('className="py-4 flex flex-col"');
      expect(source, file).not.toContain(
        'className="py-4 flex flex-col relative"'
      );
    }
  });

  test("uses shared workspace info surfaces for persistent page descriptions", () => {
    for (const file of workspaceInfoPages) {
      const source = readSettingsPage(file);

      expect(source, file).toContain("WorkspacePageInfo");
    }
  });

  test("uses shared workspace shell instead of page-local top spacing", () => {
    for (const file of workspaceRhythmPages) {
      const source = readSettingsPage(file);

      expect(source, file).toContain("<WorkspacePageLayout");
      expect(source, file).not.toContain('className="w-full px-4 py-4');
      expect(source, file).not.toContain('className="px-4 py-4');
    }
  });

  test("does not stack page top padding with general settings form sections", () => {
    const source = readSettingsPage("general/GeneralPage.tsx");

    expect(source).toContain(
      '<WorkspacePageLayout className="py-0 divide-y divide-block-border">'
    );
    expect(source).not.toContain('className="-mb-4"');
  });

  test("does not reserve top gap for hidden custom approval feature attention", () => {
    const source = readSettingsPage("CustomApprovalPage.tsx");

    expect(source).toContain(
      '<div ref={featureAttentionRef} className="empty:hidden">'
    );
  });

  test("combines global masking info into one shared info surface", () => {
    const source = readSettingsPage("GlobalMaskingPage.tsx");

    expect(source).not.toContain("@/react/components/ui/alert");
    expect(source).toContain("custom-approval.rule.first-match-wins");
    expect(source).toContain(
      "settings.sensitive-data.global-rules.description"
    );
  });
});
