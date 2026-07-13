import { readFileSync } from "node:fs";
import { join } from "node:path";
import { describe, expect, test } from "vitest";

const projectDir = import.meta.dirname;
const settingsDir = join(projectDir, "../settings");
const componentsDir = join(projectDir, "../../components");

const projectListPages = [
  "ProjectAccessGrantsPage.tsx",
  "ProjectAuditLogPage.tsx",
  "ProjectDatabaseGroupsPage.tsx",
  "ProjectDatabasesPage.tsx",
  "ProjectGitOpsPage.tsx",
  "ProjectIssueDashboardPage.tsx",
  "ProjectPlanDashboardPage.tsx",
  "ProjectReleaseDashboardPage.tsx",
  "ProjectWebhooksPage.tsx",
] as const;

const projectInfoPages = [
  "ProjectIssueDashboardPage.tsx",
  "ProjectPlanDashboardPage.tsx",
  "ProjectReleaseDashboardPage.tsx",
  "ProjectSyncSchemaPage.tsx",
] as const;

const projectReusableSettingsPages = [
  "MembersPage.tsx",
  "ServiceAccountsPage.tsx",
  "WorkloadIdentitiesPage.tsx",
] as const;

function readProjectPage(
  file: (typeof projectListPages)[number] | (typeof projectInfoPages)[number]
) {
  return readFileSync(join(projectDir, file), "utf8");
}

function readReusableSettingsPage(
  file: (typeof projectReusableSettingsPages)[number]
) {
  return readFileSync(join(settingsDir, file), "utf8");
}

describe("project page layout", () => {
  test("keeps the project shell page rhythm", () => {
    const source = readFileSync(
      join(componentsDir, "ProjectPageLayout.tsx"),
      "utf8"
    );

    expect(source).toContain("paddingBlock: 16");
    expect(source).toContain("paddingInline: 16");
    expect(source).toContain("rowGap: 16");
    expect(source).toContain('overflowX: "clip"');
    expect(source).not.toContain('overflowX: "hidden"');
    expect(source).not.toContain("paddingBottom: 8");
  });

  test("uses the shared project layout shell instead of page-local padding", () => {
    for (const file of projectListPages) {
      const source = readProjectPage(file);

      expect(source, file).toContain("@/react/components/ProjectPageLayout");
      expect(source, file).toContain("<ProjectPageLayout");
      expect(source, file).not.toContain('className="py-4 flex flex-col"');
      expect(source, file).not.toContain(
        'className="py-4 w-full flex flex-col"'
      );
      expect(source, file).not.toContain(
        'className="w-full px-4 flex flex-col gap-y-1 py-4"'
      );
    }
  });

  test("uses project layout when project routes reuse settings pages", () => {
    for (const file of projectReusableSettingsPages) {
      const source = readReusableSettingsPage(file);

      expect(source, file).toContain("ProjectPageLayout");
      expect(source, file).toContain("ProjectPageToolbar");
      expect(source, file).toContain(
        "const PageLayout = projectName ? ProjectPageLayout : WorkspacePageLayout"
      );
      expect(source, file).toContain(
        "const PageToolbar = projectName ? ProjectPageToolbar : WorkspacePageToolbar"
      );
      expect(source, file).toContain("<PageLayout>");
    }
  });

  test("does not reserve top gap for hidden access grants feature attention", () => {
    const source = readProjectPage("ProjectAccessGrantsPage.tsx");

    expect(source).toContain(
      "<FeatureAttention feature={PlanFeature.FEATURE_JIT} />"
    );
    expect(source).not.toContain('<div className="mb-2">');
  });

  test("keeps database groups on the shared page vertical rhythm", () => {
    const source = readProjectPage("ProjectDatabaseGroupsPage.tsx");
    const databaseGroupTable = readFileSync(
      join(componentsDir, "DatabaseGroupTable.tsx"),
      "utf8"
    );

    expect(source).not.toContain('<ProjectPageLayout className="gap-y-');
    expect(source).toContain("<DatabaseGroupTable");
    expect(source).toContain('className="gap-y-4"');
    expect(databaseGroupTable).toContain("readonly className?: string");
    expect(databaseGroupTable).toContain(
      'className={cn("flex flex-col gap-y-3", className)}'
    );
  });

  test("uses shared project info surfaces for persistent page descriptions", () => {
    for (const file of projectInfoPages) {
      const source = readProjectPage(file);

      expect(source, file).toContain("ProjectPageInfo");
      expect(source, file).not.toContain("onDismiss={dismissHint}");
    }
  });

  test("does not stack page-local spacing on top of shared page rhythm", () => {
    for (const file of [
      "ProjectIssueDashboardPage.tsx",
      "ProjectPlanDashboardPage.tsx",
      "ProjectReleaseDashboardPage.tsx",
      "ProjectSyncSchemaPage.tsx",
    ] as const) {
      const source = readProjectPage(file);

      expect(source, file).not.toContain(
        '<ProjectPageContent className="mt-2">'
      );
      expect(source, file).not.toContain("flex flex-col gap-y-2 pb-2");
      expect(source, file).not.toContain(
        'className="pt-4 flex-1 overflow-hidden flex flex-col gap-y-4"'
      );
    }
  });

  test("uses explicit toolbar alignment instead of Tailwind overrides", () => {
    const webhooks = readProjectPage("ProjectWebhooksPage.tsx");
    const accessGrants = readProjectPage("ProjectAccessGrantsPage.tsx");

    expect(webhooks).toContain('<ProjectPageToolbar align="end">');
    expect(webhooks).not.toContain('className="justify-end"');
    expect(accessGrants).toContain('<ProjectPageToolbar align="start">');
    expect(accessGrants).not.toContain('className="justify-start"');
  });

  test("keeps audit log table free of page-level layout", () => {
    const projectAuditLogPage = readProjectPage("ProjectAuditLogPage.tsx");
    const auditLogTable = readFileSync(
      join(componentsDir, "AuditLogTable.tsx"),
      "utf8"
    );

    expect(projectAuditLogPage).toContain("ProjectPageContent");
    expect(auditLogTable).not.toContain("FeatureAttention");
    expect(auditLogTable).not.toContain('className="px-4 py-4');
    expect(auditLogTable).not.toContain('className="mx-4');
    expect(auditLogTable).not.toContain('className="mt-4 mx-2"');
  });

  test("uses bordered table surfaces inside padded project pages", () => {
    const projectPlanPage = readProjectPage("ProjectPlanDashboardPage.tsx");
    const auditLogTable = readFileSync(
      join(componentsDir, "AuditLogTable.tsx"),
      "utf8"
    );

    expect(projectPlanPage).toContain(
      'className="overflow-x-auto rounded-sm border border-block-border"'
    );
    expect(auditLogTable).toContain(
      'className="overflow-x-auto rounded-sm border border-block-border"'
    );
  });
});
