import { mount } from "@vue/test-utils";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import { defineComponent, h, nextTick, reactive } from "vue";

const mocks = vi.hoisted(() => ({
  route: {
    name: "sql-editor.project",
    fullPath: "/sql-editor/projects/prod",
    params: { project: "prod" },
    query: {},
    matched: [] as Array<{ meta: { requiredPermissionList?: () => string[] } }>,
  },
  routerReplace: vi.fn(),
  maybeSwitchProject: vi.fn(async (project: string) => {
    if (mocks.editorStore) {
      mocks.editorStore.project = project;
    }
    return true;
  }),
  editorStore: undefined as
    | {
        project: string;
        projectContextReady: boolean;
        allowViewALLProjects: boolean;
        storedLastViewedProject: string;
        setProject: (project: string) => void;
      }
    | undefined,
  reactGuardAllows: true,
}));

vi.mock("@vueuse/core", () => ({
  useLocalStorage: (_key: string, initial: unknown) => ({ value: initial }),
}));

vi.mock("vue-router", async (importOriginal) => {
  const actual = await importOriginal<typeof import("vue-router")>();
  return {
    ...actual,
    useRoute: () => mocks.route,
    useRouter: () => ({
      replace: mocks.routerReplace,
    }),
  };
});

vi.mock("@/react/ReactPageMount.vue", async () => {
  const { defineComponent, h, onMounted, ref } = await import("vue");
  return {
    default: defineComponent({
      name: "MockReactPageMount",
      props: {
        page: {
          type: String,
          required: true,
        },
        pageProps: {
          type: Object,
          default: undefined,
        },
      },
      setup(props) {
        const target = ref<HTMLDivElement | null>(null);
        onMounted(() => {
          props.pageProps?.onReady?.(
            mocks.reactGuardAllows ? target.value : null
          );
        });
        return () =>
          h("div", { "data-testid": "react-permission-shell" }, [
            h("div", {
              ref: target,
              "data-testid": "react-permission-target",
              class: props.pageProps?.targetClassName,
            }),
            !mocks.reactGuardAllows &&
              h("div", { "data-testid": "react-permission-denied" }),
          ]);
      },
    }),
  };
});

vi.mock("@/bbkit", async () => {
  const { defineComponent, h } = await import("vue");
  return {
    BBSpin: defineComponent({
      name: "MockBBSpin",
      setup() {
        return () => h("div", { "data-testid": "sql-editor-spinner" });
      },
    }),
  };
});

vi.mock("@/plugins/ai", async () => {
  const { defineComponent, h } = await import("vue");
  return {
    ProvideAIContext: defineComponent({
      name: "MockProvideAIContext",
      setup(_, { slots }) {
        return () =>
          h("div", { "data-testid": "provide-ai-context" }, slots.default?.());
      },
    }),
  };
});

vi.mock("@/composables/useEmitteryEventListener", () => ({
  useEmitteryEventListener: vi.fn(),
}));

vi.mock("@/composables/useRouteChangeGuard", () => ({
  useRouteChangeGuard: vi.fn(),
}));

vi.mock("@/plugins/i18n", () => ({
  t: (key: string) => key,
}));

vi.mock("vue-i18n", () => ({
  useI18n: () => ({
    t: (key: string) => key,
  }),
}));

vi.mock("@/router/sqlEditor", () => ({
  SQL_EDITOR_DATABASE_MODULE: "sql-editor.database",
  SQL_EDITOR_HOME_MODULE: "sql-editor.home",
  SQL_EDITOR_INSTANCE_MODULE: "sql-editor.instance",
  SQL_EDITOR_PROJECT_MODULE: "sql-editor.project",
  SQL_EDITOR_WORKSHEET_MODULE: "sql-editor.worksheet",
}));

vi.mock("@/store/modules/sqlEditor/legacy/migration", () => ({
  migrateLegacyCache: vi.fn(async () => undefined),
}));

vi.mock("@/views/sql-editor/context", async () => {
  const { ref } = await import("vue");
  return {
    ASIDE_PANEL_TABS: ["WORKSHEET", "CONNECTION"],
    useSQLEditorContext: () => ({
      asidePanelTab: ref("WORKSHEET"),
      events: {},
      maybeSwitchProject: mocks.maybeSwitchProject,
    }),
  };
});

vi.mock("@/types", () => ({
  DEFAULT_SQL_EDITOR_TAB_MODE: "READONLY",
  BASIC_WORKSPACE_PERMISSIONS: [],
  isValidDatabaseName: (name?: string) => name?.startsWith("instances/"),
  isValidInstanceName: (name?: string) => name?.startsWith("instances/"),
  isValidProjectName: (name?: string) => name?.startsWith("projects/"),
}));

vi.mock("@/utils", () => ({
  emptySQLEditorConnection: () => ({}),
  extractDatabaseResourceName: (name: string) => ({
    instance: name.split("/databases/")[0],
    databaseName: name.split("/databases/")[1] ?? "",
  }),
  extractInstanceResourceName: (name: string) => name.split("/").pop() ?? name,
  extractProjectResourceName: (name: string) => name.split("/").pop() ?? name,
  extractWorksheetConnection: vi.fn(async () => ({})),
  extractWorksheetID: (name: string) => name.split("/").pop() ?? name,
  getDefaultPagination: () => 100,
  getSheetStatement: () => "",
  isWorksheetReadableV1: () => true,
  STORAGE_KEY_SQL_EDITOR_SIDEBAR_TAB: "bb.sql-editor.sidebar-tab",
  suggestedTabTitleForSQLEditorConnection: () => "tab",
}));

vi.mock("@/store", () => ({
  pushNotification: vi.fn(),
  usePermissionStore: () => ({
    currentPermissions: new Set<string>(["bb.test"]),
    currentPermissionsInProjectV1: () => new Set<string>(["bb.test"]),
  }),
  useActuatorV1Store: () => ({
    serverInfo: { defaultProject: "projects/default" },
  }),
  useDatabaseV1Store: () => ({
    getOrFetchDatabaseByName: vi.fn(async (name: string) => ({
      name,
      project: "projects/prod",
    })),
  }),
  useProjectV1Store: () => ({
    fetchProjectList: vi.fn(async () => ({
      projects: [{ name: "projects/prod" }],
    })),
    getProjectByName: (name: string) => ({
      name,
      allowRequestRole: true,
    }),
  }),
  useSQLEditorStore: () => mocks.editorStore,
  useSQLEditorTabStore: () => ({
    openTabList: [],
    currentTab: undefined,
    getTabByWorksheet: vi.fn(),
    updateTab: vi.fn(),
    addTab: vi.fn(),
    initProject: vi.fn(async () => undefined),
  }),
  useWorkSheetStore: () => ({
    getOrFetchWorksheetByName: vi.fn(async () => undefined),
    getWorksheetByName: vi.fn(() => undefined),
  }),
}));

import ProvideSQLEditorContext from "./ProvideSQLEditorContext.vue";

const mountContext = () =>
  mount(ProvideSQLEditorContext, {
    attachTo: document.body,
    global: {
      stubs: {
        RouterView: defineComponent({
          name: "MockRouterView",
          setup() {
            return () =>
              h("div", { "data-testid": "sql-editor-route-content" });
          },
        }),
      },
    },
  });

beforeEach(() => {
  document.body.innerHTML = '<ul id="sql-editor-debug"></ul>';
  mocks.routerReplace.mockClear();
  mocks.maybeSwitchProject.mockClear();
  mocks.reactGuardAllows = true;
  mocks.route.name = "sql-editor.project";
  mocks.route.fullPath = "/sql-editor/projects/prod";
  mocks.route.params = { project: "prod" };
  mocks.route.query = {};
  mocks.editorStore = reactive({
    project: "projects/prod",
    projectContextReady: false,
    allowViewALLProjects: true,
    storedLastViewedProject: "projects/prod",
    setProject(project: string) {
      this.project = project;
    },
  });
});

afterEach(() => {
  document.body.innerHTML = "";
});

describe("ProvideSQLEditorContext permission shell bridge", () => {
  test("renders the loading state until project context is ready", () => {
    mountContext();

    expect(
      document.body.querySelector("[data-testid='sql-editor-spinner']")
    ).not.toBeNull();
    expect(
      document.body.querySelector("[data-testid='sql-editor-route-content']")
    ).toBeNull();
  });

  test("teleports SQL Editor route content into the React permission target", async () => {
    mountContext();
    await nextTick();

    mocks.editorStore!.projectContextReady = true;
    await nextTick();
    await nextTick();

    const target = document.body.querySelector(
      "[data-testid='react-permission-target']"
    );
    const content = document.body.querySelector(
      "[data-testid='sql-editor-route-content']"
    );

    expect(target?.contains(content)).toBe(true);
    expect(target?.className).toContain("h-full");
    expect(target?.className).toContain("flex");
    expect(
      document.body.querySelector("[data-testid='provide-ai-context']")
    ).not.toBeNull();
  });

  test("withholds SQL Editor route content when the React guard denies access", async () => {
    mocks.reactGuardAllows = false;
    mountContext();
    await nextTick();

    mocks.editorStore!.projectContextReady = true;
    await nextTick();
    await nextTick();

    expect(
      document.body.querySelector("[data-testid='react-permission-denied']")
    ).not.toBeNull();
    expect(
      document.body.querySelector("[data-testid='sql-editor-route-content']")
    ).toBeNull();
  });
});
