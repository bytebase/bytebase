import { describe, expect, test } from "vitest";

const sources = import.meta.glob(
  [
    "./pages/project/**/*.{ts,tsx}",
    "./pages/settings/**/*.{ts,tsx}",
    "../router/dashboard/projectV1.ts",
    "../router/dashboard/workspace.ts",
    "../router/dashboard/workspaceSetting.ts",
    "../layouts/BodyLayout.vue",
    "../layouts/ProjectV1Layout.vue",
    "../layouts/SQLEditorLayout.vue",
    "../components/ProvideSQLEditorContext.vue",
    "./components/RoutePermissionGuardShell.tsx",
  ],
  {
    query: "?raw",
    import: "default",
    eager: true,
  }
) as Record<string, string>;

const legacyVueImportPattern =
  /@\/components\/(?:Plan|RolloutV1|IssueV1|PlanCheckRun|DatabaseDetail)\/[^"']+\.vue/g;

const legacyImplementationPaths = [
  "@/components/Plan/components",
  "@/components/RolloutV1/components",
  "@/components/IssueV1/components/RoleGrant",
  "@/components/DatabaseDetail/Settings",
];

describe("React Project and Settings legacy Vue dependencies", () => {
  test("do not import deleted Project Vue implementation paths", () => {
    const violations: string[] = [];
    for (const [file, source] of Object.entries(sources)) {
      const vueImports = source.match(legacyVueImportPattern) ?? [];
      for (const vueImport of vueImports) {
        violations.push(`${file}: ${vueImport}`);
      }
      for (const legacyPath of legacyImplementationPaths) {
        if (source.includes(legacyPath)) {
          violations.push(`${file}: ${legacyPath}`);
        }
      }
    }
    expect(violations).toEqual([]);
  });

  test("console and SQL Editor shell do not re-import legacy Vue permission chain", () => {
    const bannedImports = [
      "@/components/Permission/RoutePermissionGuard.vue",
      "@/components/Permission/ComponentPermissionGuard.vue",
      "@/components/RoleGrantPanel",
      "./RoleGrantPanel.vue",
      "@/components/FeatureGuard",
      "@/components/IssueV1/components/Sidebar/IssueLabels",
      "./IssueLabelSelector.vue",
      "@/components/ProjectMember/AddProjectMember/AddProjectMemberForm.vue",
    ];
    const violations: string[] = [];
    for (const [file, source] of Object.entries(sources)) {
      for (const bannedImport of bannedImports) {
        if (source.includes(bannedImport)) {
          violations.push(`${file}: ${bannedImport}`);
        }
      }
    }
    expect(violations).toEqual([]);
  });
});
