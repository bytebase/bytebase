import { createPinia, setActivePinia } from "pinia";
import { beforeEach, describe, expect, test, vi } from "vitest";

// Mock @/store to stub sub-stores (mirrors uiState.test.ts pattern)
vi.mock("@/store", async () => {
  const { ref } = await import("vue");
  return {
    pushNotification: vi.fn(),
    useCurrentUserV1: vi.fn(() => ({
      value: { email: "test@example.com" },
    })),
    useProjectIamPolicyStore: vi.fn(() => ({
      getOrFetchProjectIamPolicy: vi.fn(),
    })),
    useProjectV1Store: vi.fn(() => ({
      getOrFetchProjectByName: vi.fn(),
    })),
    useSQLEditorStore: vi.fn(() => ({
      project: ref("projects/default"),
      projectContextReady: true,
      setProject: vi.fn(),
    })),
    useSQLEditorTabStore: vi.fn(() => ({
      updateTab: vi.fn(),
      getTabById: vi.fn(),
      currentTab: undefined,
      openTabList: [],
    })),
    useSQLEditorUIStore: vi.fn(() => ({
      showConnectionPanel: ref(false),
    })),
    useWorkSheetStore: vi.fn(() => ({
      getWorksheetByName: vi.fn(),
      patchWorksheet: vi.fn(),
      upsertWorksheetOrganizer: vi.fn(),
      batchUpsertWorksheetOrganizers: vi.fn(),
      createWorksheet: vi.fn(),
      fetchWorksheetList: vi.fn(),
      myWorksheetList: [],
      sharedWorksheetList: [],
    })),
  };
});

// Mock @/utils completely to avoid native binary dependencies
vi.mock("@/utils", async () => {
  const { ref } = await import("vue");
  return {
    extractWorksheetConnection: vi.fn(
      async ({ database }: { database: string }) => ({
        database,
        instance: "",
      })
    ),
    isWorksheetWritableV1: vi.fn(() => true),
    isValidProjectName: vi.fn((name: string) => name.startsWith("projects/")),
    storageKeySqlEditorWorksheetFilter: vi.fn(() => "filter-key"),
    storageKeySqlEditorWorksheetTree: vi.fn(() => "tree-key"),
    storageKeySqlEditorWorksheetFolder: vi.fn(() => "folder-key"),
    useDynamicLocalStorage: vi.fn((_: unknown, defaultValue: unknown) =>
      ref(defaultValue)
    ),
  };
});

// Mock @/types to avoid importing native modules transitively
vi.mock("@/types", () => ({
  isValidProjectName: vi.fn((name: string) => name.startsWith("projects/")),
  DEBOUNCE_SEARCH_DELAY: 300,
}));

// Mock @/views/sql-editor/Sheet
vi.mock("@/views/sql-editor/Sheet", () => ({
  openWorksheetByName: vi.fn(),
}));

// Mock @/views/sql-editor/events
vi.mock("@/views/sql-editor/events", () => ({
  sqlEditorEvents: {
    emit: vi.fn(),
  },
}));

// Mock protobuf and worksheet_service_pb
vi.mock("@bufbuild/protobuf", () => ({
  create: vi.fn(),
}));

vi.mock("@/types/proto-es/v1/worksheet_service_pb", () => ({
  Worksheet_Visibility: { PRIVATE: 0 },
  WorksheetSchema: {},
}));

let useSQLEditorWorksheetStore: typeof import("./worksheet").useSQLEditorWorksheetStore;

beforeEach(async () => {
  vi.clearAllMocks();
  setActivePinia(createPinia());
  ({ useSQLEditorWorksheetStore } = await import("./worksheet"));
});

describe("useSQLEditorWorksheetStore", () => {
  test("store initializes with autoSaveController === null", () => {
    const store = useSQLEditorWorksheetStore();
    expect(store.autoSaveController).toBeNull();
  });

  test("abortAutoSave() with no controller is a no-op (doesn't throw)", () => {
    const store = useSQLEditorWorksheetStore();
    expect(() => store.abortAutoSave()).not.toThrow();
    expect(store.autoSaveController).toBeNull();
  });

  test("abortAutoSave() with a controller calls abort and clears it", () => {
    const store = useSQLEditorWorksheetStore();
    const controller = new AbortController();
    const abortSpy = vi.spyOn(controller, "abort");
    store.autoSaveController = controller;

    store.abortAutoSave();

    expect(abortSpy).toHaveBeenCalledTimes(1);
    expect(store.autoSaveController).toBeNull();
  });

  test("maybeSwitchProject with an invalid project name returns undefined without setting project", async () => {
    const store = useSQLEditorWorksheetStore();
    const { useSQLEditorStore } = await import("@/store");
    const editorStore = useSQLEditorStore();

    const result = await store.maybeSwitchProject("not-a-valid-project-name");

    expect(result).toBeUndefined();
    expect(editorStore.setProject).not.toHaveBeenCalled();
  });
});
