import { describe, expect, test } from "vitest";

const sources = import.meta.glob(
  [
    "./pages/project/**/*.{ts,tsx}",
    "./pages/settings/**/*.{ts,tsx}",
    "./pages/auth/**/*.{ts,tsx}",
    "./components/**/*.{ts,tsx}",
    "../components/ProvideSQLEditorContext.vue",
    "./components/ProjectRouteShell.tsx",
    "./components/SettingRouteShell.tsx",
    "./components/InstanceRouteShell.tsx",
    "./components/IssuesRouteShell.tsx",
    "./components/RoutePermissionGuardShell.tsx",
    "./components/ComponentPermissionGuard.tsx",
  ],
  {
    query: "?raw",
    import: "default",
    eager: true,
  }
) as Record<string, string>;

const currentUserMigrationSources = import.meta.glob(
  [
    "./**/*.{ts,tsx}",
    "../plugins/ai/react/**/*.{ts,tsx}",
    "../connect/middlewares/activeInterceptorMiddleware.ts",
    "../utils/pagination.ts",
    "../utils/v1/worksheet.ts",
  ],
  {
    query: "?raw",
    import: "default",
    eager: true,
  }
) as Record<string, string>;

const instanceMigrationSources = import.meta.glob(
  ["../utils/expr.ts", "../utils/v1/issue/issue.ts"],
  {
    query: "?raw",
    import: "default",
    eager: true,
  }
) as Record<string, string>;

const removedLegacyInstanceStoreSources = import.meta.glob(
  ["./pages/settings/InstancesPage.tsx", "../store/modules/v1/index.ts"],
  {
    query: "?raw",
    import: "default",
    eager: true,
  }
) as Record<string, string>;

const databaseMigrationSources = import.meta.glob(
  [
    "../utils/expr.ts",
    "../utils/v1/revision.ts",
    "../utils/v1/changelog.ts",
    "../utils/v1/issue/rollout.ts",
    "../utils/v1/issue/issue.ts",
    "../store/modules/v1/index.ts",
  ],
  {
    query: "?raw",
    import: "default",
    eager: true,
  }
) as Record<string, string>;

const dbSchemaMigrationSources = import.meta.glob(
  ["./**/*.{ts,tsx}", "../store/modules/v1/index.ts"],
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
      "./RoleGrantPanel.vue",
      "@/components/FeatureGuard",
      "@/components/IssueV1/components/Sidebar/IssueLabels",
      "./IssueLabelSelector.vue",
      "@/components/ProjectMember/AddProjectMember/AddProjectMemberForm.vue",
      "@/layouts/ProjectV1Layout.vue",
      "@/layouts/SettingLayout.vue",
      "@/layouts/InstanceLayout.vue",
      "@/layouts/IssuesLayout.vue",
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

  test("route permission guard reads React permission state", () => {
    const violations: string[] = [];
    for (const [file, source] of Object.entries(sources)) {
      if (!file.endsWith("/ComponentPermissionGuard.tsx")) {
        continue;
      }
      for (const bannedImport of [
        "usePermissionStore",
        "@/react/components/PermissionGuard",
      ]) {
        if (source.includes(bannedImport)) {
          violations.push(`${file}: ${bannedImport}`);
        }
      }
    }
    expect(violations).toEqual([]);
  });

  test("ProjectRouteShell does not preload unrelated permission state", () => {
    const violations: string[] = [];
    for (const [file, source] of Object.entries(sources)) {
      if (!file.endsWith("/ProjectRouteShell.tsx")) {
        continue;
      }
      for (const bannedImport of [
        "@/store",
        "useProjectIamPolicyStore",
        "loadSubscription",
      ]) {
        if (source.includes(bannedImport)) {
          violations.push(`${file}: ${bannedImport}`);
        }
      }
    }
    expect(violations).toEqual([]);
  });

  test("Phase 1 protobuf resource consumers use the React app store", () => {
    const bannedImports = [
      "useGroupStore",
      "useIdentityProviderStore",
      "useAccessGrantStore",
      "@/store/modules/idp",
      "@/store/modules/accessGrant",
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

  test("Phase 2 protobuf resource consumers use the React app store", () => {
    const bannedImports = [
      "useUserStore",
      "useRoleStore",
      "@/store/modules/user",
      "@/store/modules/role",
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

  test("migrated React and utility surfaces do not use the legacy current-user hook", () => {
    const legacyCurrentUserHook = ["use", "Current", "User", "V1"].join("");
    const violations: string[] = [];
    for (const [file, source] of Object.entries(currentUserMigrationSources)) {
      if (file.endsWith("/no-legacy-vue-deps.test.ts")) {
        continue;
      }
      if (source.includes(legacyCurrentUserHook)) {
        violations.push(file);
      }
    }
    expect(violations).toEqual([]);
  });

  test("migrated router and utility surfaces do not use legacy instance stores", () => {
    const bannedImports = ["useInstanceV1Store", "useInstanceResourceByName"];
    const violations: string[] = [];
    for (const [file, source] of Object.entries(instanceMigrationSources)) {
      for (const bannedImport of bannedImports) {
        if (source.includes(bannedImport)) {
          violations.push(`${file}: ${bannedImport}`);
        }
      }
    }
    expect(violations).toEqual([]);
  });

  test("migrated instance surfaces do not expose the legacy instance store module", () => {
    const bannedImports = ["@/store/modules/v1/instance", "./instance"];
    const violations: string[] = [];
    for (const [file, source] of Object.entries(
      removedLegacyInstanceStoreSources
    )) {
      for (const bannedImport of bannedImports) {
        if (source.includes(bannedImport)) {
          violations.push(`${file}: ${bannedImport}`);
        }
      }
    }
    expect(violations).toEqual([]);
  });

  test("migrated database surfaces do not use the legacy database store", () => {
    const bannedImports = ["useDatabaseV1Store", "@/store/modules/v1/database"];
    const violations: string[] = [];
    for (const [file, source] of Object.entries(databaseMigrationSources)) {
      for (const bannedImport of bannedImports) {
        if (source.includes(bannedImport)) {
          violations.push(`${file}: ${bannedImport}`);
        }
      }
      if (file.endsWith("../store/modules/v1/index.ts")) {
        const bannedStoreModuleExport = "./database";
        if (source.includes(bannedStoreModuleExport)) {
          violations.push(`${file}: ${bannedStoreModuleExport}`);
        }
      }
    }
    expect(violations).toEqual([]);
  });

  test("migrated DB schema surfaces do not expose the legacy DB schema store", () => {
    const violations: string[] = [];
    for (const [file, source] of Object.entries(dbSchemaMigrationSources)) {
      if (file.endsWith("/no-legacy-vue-deps.test.ts")) {
        continue;
      }
      if (source.includes("@/store/modules/v1/dbSchema")) {
        violations.push(`${file}: @/store/modules/v1/dbSchema`);
      }
      if (
        !file.endsWith(".test.ts") &&
        !file.endsWith(".test.tsx") &&
        source.includes("useDBSchemaV1Store")
      ) {
        violations.push(`${file}: useDBSchemaV1Store`);
      }
      if (
        file.endsWith("../store/modules/v1/index.ts") &&
        source.includes("./dbSchema")
      ) {
        violations.push(`${file}: ./dbSchema`);
      }
    }
    expect(violations).toEqual([]);
  });
});
