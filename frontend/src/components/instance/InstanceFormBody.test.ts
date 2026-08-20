import { readFileSync } from "node:fs";
import { join } from "node:path";
import { describe, expect, test } from "vitest";

describe("InstanceFormBody", () => {
  test("uses boolean values for inert collapsible sections", () => {
    const source = readFileSync(
      join(process.cwd(), "src/components/instance/InstanceFormBody.tsx"),
      "utf-8"
    );

    expect(source).not.toContain(
      'inert={isEngineSelectorCollapsed ? "" : undefined}'
    );
    expect(source).not.toContain(
      'inert={isConnectionOptionsCollapsed ? "" : undefined}'
    );
    expect(source).toContain(
      "inert={isEngineSelectorCollapsed ? true : undefined}"
    );
    expect(source).toContain(
      "inert={isConnectionOptionsCollapsed ? true : undefined}"
    );
  });

  test("renders database sync controls inside the connection card", () => {
    const source = readFileSync(
      join(process.cwd(), "src/components/instance/InstanceFormBody.tsx"),
      "utf-8"
    );

    const basicInfoIndex = source.indexOf("{/* Basic Info Card */}");
    const connectionCardIndex = source.indexOf("{/* Connection Card */}");
    const syncDatabasesIndex = source.indexOf("<SyncDatabases");
    const connectionOptionsIndex = source.indexOf(
      "{/* Connection Options Card */}"
    );

    expect(basicInfoIndex).toBeGreaterThanOrEqual(0);
    expect(connectionCardIndex).toBeGreaterThan(basicInfoIndex);
    expect(syncDatabasesIndex).toBeGreaterThan(connectionCardIndex);
    expect(syncDatabasesIndex).toBeLessThan(connectionOptionsIndex);
  });

  test("renders database sync controls only once", () => {
    const source = readFileSync(
      join(process.cwd(), "src/components/instance/InstanceFormBody.tsx"),
      "utf-8"
    );

    expect(source.match(/<SyncDatabases/g)).toHaveLength(1);
  });

  test("labels project-aware database sync without a redundant alert", () => {
    const source = readFileSync(
      join(process.cwd(), "src/components/instance/InstanceFormBody.tsx"),
      "utf-8"
    );

    expect(source).not.toContain("instance.sync-databases.project-self");
    expect(source).not.toContain("showLabel && !hasProjectContext");
    expect(source).toContain("title=");
    expect(source).toContain("showLabel ? (");
    expect(source).toContain("instance.sync-databases.description");
    expect(source).toContain('t("instance.sync-databases.self")');
    expect(source).not.toContain("instance.sync-databases.project-description");
    expect(source).toContain("instance.sync-databases.project-sync-all");
    expect(source).toContain("projectName");
    expect(source).not.toContain('useProjectByName(projectName ?? "")');
    expect(source).not.toContain("fetchProject(routeProjectName");
    expect(source).toContain("projectName={parent}");
    expect(source).not.toContain("ResourceLink");
  });

  test("preserves the project parent when the resource id changes", () => {
    const source = readFileSync(
      join(process.cwd(), "src/components/instance/InstanceFormBody.tsx"),
      "utf-8"
    );

    expect(source).toContain(
      "parent ? `${parent}/instances/${id}` : `instances/${id}`"
    );
    expect(source).toContain(
      "parent ? `${parent}/instances/${id}` : `${instanceNamePrefix}${id}`"
    );
    expect(source).not.toContain("router.currentRoute.value.query.project");
  });

  test("does not treat an inaccessible instance as a duplicate resource id", () => {
    const source = readFileSync(
      join(process.cwd(), "src/components/instance/InstanceFormBody.tsx"),
      "utf-8"
    );

    expect(source).toContain("existing.name !== UNKNOWN_INSTANCE_NAME");
  });

  test("offers sync-all database details from the create instance form", () => {
    const source = readFileSync(
      join(process.cwd(), "src/components/instance/InstanceFormBody.tsx"),
      "utf-8"
    );
    const titleIndex = source.indexOf('t("instance.sync-databases.self")');
    const infoTriggerIndex = source.indexOf(
      'onOpenInfoPanel("sync-databases")'
    );
    const descriptionIndex = source.indexOf(
      't("instance.sync-databases.description")'
    );
    const checkboxIndex = source.indexOf("<Checkbox", descriptionIndex);

    expect(source).toContain(
      "onOpenInfoPanel?: (section: InfoSection) => void"
    );
    expect(source).toContain("onOpenInfoPanel={onOpenInfoPanel}");
    expect(source).toContain(
      'className="inline-flex size-4 shrink-0 items-center justify-center text-accent leading-none"'
    );
    expect(source).toContain('<Info className="size-3.5" />');
    expect(infoTriggerIndex).toBeGreaterThan(titleIndex);
    expect(infoTriggerIndex).toBeLessThan(descriptionIndex);
    expect(infoTriggerIndex).toBeLessThan(checkboxIndex);
  });

  test("lets users load more database sync options", () => {
    const source = readFileSync(
      join(process.cwd(), "src/components/instance/InstanceFormBody.tsx"),
      "utf-8"
    );

    expect(source).toContain("pendingScrollDatabaseRef");
    expect(source).toContain("scrollIntoView");
    expect(source).toContain("visibleDatabaseCount");
    expect(source).toContain("setVisibleDatabaseCount");
    expect(source).toContain(
      "filteredDatabases.slice(0, visibleDatabaseCount)"
    );
    expect(source).toContain('t("common.load-more")');
    expect(source).not.toContain('{t("common.load-more")} (');
    expect(source).not.toContain(
      "filteredDatabases.length - MAX_VISIBLE_DATABASES"
    );
  });

  test("shows cloud connection instruction under the connection section title", () => {
    const source = readFileSync(
      join(process.cwd(), "src/components/instance/InstanceFormBody.tsx"),
      "utf-8"
    );

    const connectionTitleIndex = source.indexOf(
      't("instance.section.connection")'
    );
    const firewallInfoIndex = source.indexOf(
      't("instance.sentence.firewall-info")'
    );
    const firewallAlertIndex = source.indexOf(
      '<Alert variant="info"',
      connectionTitleIndex
    );
    const connectionGridIndex = source.indexOf(
      'className="mt-3 grid grid-cols-1 gap-y-6 gap-x-4 sm:grid-cols-4"',
      connectionTitleIndex
    );

    expect(connectionTitleIndex).toBeGreaterThanOrEqual(0);
    expect(firewallAlertIndex).toBeGreaterThan(connectionTitleIndex);
    expect(firewallAlertIndex).toBeLessThan(firewallInfoIndex);
    expect(firewallInfoIndex).toBeGreaterThan(connectionTitleIndex);
    expect(firewallInfoIndex).toBeLessThan(connectionGridIndex);
    expect(source).toContain(
      'href="https://docs.bytebase.com/get-started/cloud#prerequisites"'
    );
  });

  test("keeps Docker-only host suggestions out of Bytebase Cloud", () => {
    const source = readFileSync(
      join(process.cwd(), "src/components/instance/InstanceFormBody.tsx"),
      "utf-8"
    );

    expect(source).toContain("if (!updated.host && !isSaaSMode)");
    expect(source).toContain('t("instance.sentence.host.saas")');
    expect(source).toContain('t("instance.sentence.host.none-snowflake")');
  });

  test("uses the localized testing label", () => {
    const source = readFileSync(
      join(process.cwd(), "src/components/instance/InstanceFormBody.tsx"),
      "utf-8"
    );

    expect(source).toContain('t("instance.testing-connection")');
    expect(source).not.toContain('`${t("instance.test-connection")}...`');
  });

  test("shows connection recovery for explicit test connection failures", () => {
    const source = readFileSync(
      join(process.cwd(), "src/components/instance/InstanceFormBody.tsx"),
      "utf-8"
    );

    expect(source).toContain("ConnectionRecovery");
    expect(source).toContain("testConnectionFailure");
    expect(source).toContain("message: result.message");
    expect(source).toContain("failureCategory: result.failureCategory");
    expect(source).toContain("setTestConnectionFailure(undefined)");
    expect(source).toContain(
      "category={testConnectionFailure.failureCategory}"
    );
  });

  test("refetches database previews when pending create instance changes", () => {
    const source = readFileSync(
      join(process.cwd(), "src/components/instance/InstanceFormBody.tsx"),
      "utf-8"
    );

    expect(source).toContain(
      "}, [syncAll, isCreatingProp, pendingCreateInstance, instance]);"
    );
  });

  test("distinguishes sync-all from empty selected database list", () => {
    const source = readFileSync(
      join(process.cwd(), "src/components/instance/InstanceFormBody.tsx"),
      "utf-8"
    );

    expect(source).not.toContain(
      'const key = syncAll ? "" : [...selectedSet].sort().join("\\0");'
    );
    expect(source).toContain('syncAll ? "all" : "selected"');
    expect(source).toContain(
      "[...selectedSet].sort((a, b) => a.localeCompare(b))"
    );
  });

  test("preserves an explicitly empty database sync selection", () => {
    const source = readFileSync(
      join(process.cwd(), "src/components/instance/InstanceFormBody.tsx"),
      "utf-8"
    );

    expect(source).toContain("syncDatabases?: SyncDatabasesMessage");
    expect(source).toContain(
      "const [syncAll, setSyncAll] = useState(syncDatabases === undefined);"
    );
    expect(source).toContain("syncDatabases={basicInfo.syncDatabases}");
    expect(source).not.toContain(
      "syncDatabases={basicInfo.syncDatabases?.databases ?? []}"
    );
  });
});
