import { create } from "@bufbuild/protobuf";
import { useDebounceFn } from "@vueuse/core";
import Emittery from "emittery";
import { isEqual, isUndefined, orderBy } from "lodash-es";
import { defineStore, storeToRefs } from "pinia";
import scrollIntoView from "scroll-into-view-if-needed";
import type { Ref } from "vue";
import { computed, nextTick, ref, watch } from "vue";
import { t } from "@/plugins/i18n";
import { useSQLEditorStore as useSQLEditorReactStore } from "@/react/stores/sqlEditor";
import {
  useCurrentUserV1,
  useProjectIamPolicyStore,
  useProjectV1Store,
  useSQLEditorStore,
  useSQLEditorTabStore,
  useWorkSheetStore,
} from "@/store";
import type { SQLEditorTab } from "@/types";
import { DEBOUNCE_SEARCH_DELAY, isValidProjectName } from "@/types";
import {
  type Worksheet,
  Worksheet_Visibility,
  WorksheetSchema,
} from "@/types/proto-es/v1/worksheet_service_pb";
import {
  extractWorksheetConnection,
  storageKeySqlEditorWorksheetFilter,
  storageKeySqlEditorWorksheetTree,
  useDynamicLocalStorage,
} from "@/utils";
import { sqlEditorEvents } from "@/views/sql-editor/events";
import { openWorksheetByName } from "@/views/sql-editor/Sheet";
import type { SheetViewMode } from "@/views/sql-editor/Sheet/types";
import { buildFolderContext } from "./folder";

export type { FolderContext } from "./folder";

// ---- types ------------------------------------------------------------------

type SheetTreeEvents = Emittery<{
  "on-built": { viewMode: SheetViewMode };
}>;

export interface WorksheetLikeItem {
  name: string;
  title: string;
  folders: string[];
  type: "worksheet" | "draft";
}

export interface WorksheetFolderNode {
  key: string;
  label: string;
  editable: boolean;
  isLeaf?: boolean;
  empty?: boolean;
  worksheet?: WorksheetLikeItem;
  children: WorksheetFolderNode[];
  [key: string]: unknown;
}

export interface WorksheetFilter {
  keyword: string;
  onlyShowStarred: boolean;
  showMine: boolean;
  showShared: boolean;
  showDraft: boolean;
}

// ---- per-view tree context --------------------------------------------------

const convertToWorksheetLikeItem = (
  worksheet: Worksheet
): WorksheetLikeItem => ({
  name: worksheet.name,
  title: worksheet.title,
  folders: worksheet.folders,
  type: "worksheet",
});

const buildViewContext = (
  viewMode: SheetViewMode,
  filterRef: Ref<WorksheetFilter>
) => {
  const sheetStore = useWorkSheetStore();
  const folderContext = buildFolderContext(viewMode);
  const me = useCurrentUserV1();
  const events: SheetTreeEvents = new Emittery();
  const tabStore = useSQLEditorTabStore();
  const { project } = storeToRefs(useSQLEditorStore());

  const isInitialized = ref(false);
  const isLoading = ref(false);

  const rootNodeLabel = computed(() => {
    switch (viewMode) {
      case "my":
        return t("sheet.mine");
      case "shared":
        return t("sheet.shared");
      case "draft":
        return t("common.draft");
      default:
        return "";
    }
  });

  const getRootTreeNode = (): WorksheetFolderNode => ({
    isLeaf: false,
    children: [],
    key: folderContext.rootPath.value,
    label: rootNodeLabel.value,
    editable: false,
  });

  const sheetTree = ref<WorksheetFolderNode>(getRootTreeNode());

  const worksheetList = computed((): Worksheet[] => {
    let list: Worksheet[] = [];
    switch (viewMode) {
      case "my":
        list = sheetStore.myWorksheetList;
        break;
      case "shared":
        list = sheetStore.sharedWorksheetList;
        break;
      default:
        break;
    }
    list = list.filter((sheet) => sheet.project === project.value);
    if (filterRef.value.onlyShowStarred) {
      return list.filter((sheet) => sheet.starred);
    }
    return list;
  });

  const sheetLikeItemList = computed((): WorksheetLikeItem[] => {
    switch (viewMode) {
      case "my":
      case "shared":
        return worksheetList.value.map(convertToWorksheetLikeItem);
      case "draft":
        return tabStore.openTabList
          .filter((tab) => !tab.worksheet)
          .map((tab) => ({
            name: tab.id,
            title: tab.title,
            folders: [],
            type: "draft" as const,
          }));
      default:
        return [];
    }
  });

  const getPathesForWorksheet = (worksheet: {
    folders: string[];
  }): string[] => {
    const pathes = [folderContext.rootPath.value];
    let currentPath = folderContext.rootPath.value;
    for (const folder of worksheet.folders) {
      currentPath = folderContext.ensureFolderPath(`${currentPath}/${folder}`);
      pathes.push(currentPath);
    }
    return pathes;
  };

  const getPwdForWorksheet = (worksheet: { folders: string[] }): string => {
    return folderContext.ensureFolderPath(worksheet.folders.join("/"));
  };

  const getKeyForWorksheet = (worksheet: WorksheetLikeItem): string => {
    return [
      getPwdForWorksheet(worksheet),
      `bytebase-${worksheet.type}-${worksheet.name.split("/").slice(-1)[0]}.sql`,
    ].join("/");
  };

  const getFoldersForWorksheet = (path: string): string[] => {
    const pathes = path.replace(folderContext.rootPath.value, "").split("/");
    if (pathes.slice(-1)[0].endsWith(".sql")) {
      pathes.pop();
    }
    return pathes.map((p) => p.trim()).filter((p) => p);
  };

  const buildTree = (
    parent: WorksheetFolderNode,
    worksheetsByFolder: Map<string, WorksheetLikeItem[]>,
    hideEmpty: boolean
  ) => {
    const subfolders = folderContext
      .listSubFolders(parent.key)
      .map((folder) => ({
        isLeaf: false,
        children: [],
        key: folder,
        label: folder.split("/").slice(-1)[0],
        editable: true,
      }));

    let empty = true;
    for (const childNode of subfolders) {
      const subtree = buildTree(childNode, worksheetsByFolder, hideEmpty);
      if (!subtree.empty || !hideEmpty) {
        parent.children.push(subtree);
      }
      if (!subtree.empty) {
        empty = false;
      }
    }

    const sheets = orderBy(
      worksheetsByFolder.get(parent.key) || [],
      (item) => item.title
    ).map((worksheet) => ({
      isLeaf: true,
      key: getKeyForWorksheet(worksheet),
      label: worksheet.title,
      worksheet,
      editable: true,
      children: [],
    }));

    parent.children.push(...sheets);
    parent.empty = sheets.length === 0 && empty;
    if (parent.key !== folderContext.rootPath.value) {
      parent.isLeaf = parent.children.length === 0;
    }

    return parent;
  };

  const folderTree = computed(() =>
    buildTree(getRootTreeNode(), new Map(), false)
  );

  const rebuildTree = useDebounceFn(() => {
    const folderPaths = new Set<string>([]);
    const worksheetsByFolder = new Map<string, WorksheetLikeItem[]>();

    for (const worksheet of sheetLikeItemList.value) {
      for (const path of getPathesForWorksheet(worksheet)) {
        folderPaths.add(path);
      }
      const pwd = getPwdForWorksheet(worksheet);
      if (!worksheetsByFolder.has(pwd)) {
        worksheetsByFolder.set(pwd, []);
      }
      worksheetsByFolder.get(pwd)!.push(worksheet);
    }

    folderContext.mergeFolders(folderPaths);
    sheetTree.value = buildTree(
      getRootTreeNode(),
      worksheetsByFolder,
      filterRef.value.onlyShowStarred
    );
    events.emit("on-built", { viewMode });
  }, DEBOUNCE_SEARCH_DELAY);

  watch(
    [() => folderContext.folders.value, () => sheetLikeItemList.value],
    ([newFolders, newSheetList], [oldFolders, oldSheetList]) => {
      if (
        isEqual(newFolders, oldFolders) &&
        isEqual(newSheetList, oldSheetList)
      ) {
        return;
      }
      rebuildTree();
    }
  );

  const fetchSheetList = async () => {
    isLoading.value = true;
    try {
      switch (viewMode) {
        case "my":
          await sheetStore.fetchWorksheetList(
            project.value,
            `creator == "users/${me.value.email}"`
          );
          break;
        case "shared":
          await sheetStore.fetchWorksheetList(
            project.value,
            [
              `creator != "users/${me.value.email}"`,
              `visibility in ["${Worksheet_Visibility[Worksheet_Visibility.PROJECT_READ]}","${Worksheet_Visibility[Worksheet_Visibility.PROJECT_WRITE]}"]`,
            ].join(" && ")
          );
          break;
        default:
          break;
      }
      rebuildTree();
      isInitialized.value = true;
    } finally {
      isLoading.value = false;
    }
  };

  return {
    events,
    isInitialized,
    isLoading,
    sheetTree,
    folderTree,
    fetchSheetList,
    folderContext,
    getKeyForWorksheet,
    getFoldersForWorksheet,
    getPathesForWorksheet,
    getPwdForWorksheet,
  };
};

export type ViewContext = ReturnType<typeof buildViewContext>;

// ---- constants --------------------------------------------------------------

const INITIAL_FILTER: WorksheetFilter = {
  keyword: "",
  showShared: true,
  showMine: true,
  showDraft: true,
  onlyShowStarred: false,
};

// ---- Pinia store ------------------------------------------------------------

export const useSQLEditorWorksheetStore = defineStore(
  "sqlEditorWorksheet",
  () => {
    const editorStore = useSQLEditorStore();
    const tabStore = useSQLEditorTabStore();
    const projectStore = useProjectV1Store();
    const projectIamPolicyStore = useProjectIamPolicyStore();
    const worksheetStore = useWorkSheetStore();
    const me = useCurrentUserV1();
    const worksheetV1Store = useWorkSheetStore();
    const { project } = storeToRefs(useSQLEditorStore());

    // ---- save-flow actions --------------------------------------------------

    const autoSaveController = ref<AbortController | null>(null);

    const abortAutoSave = () => {
      if (autoSaveController.value) {
        autoSaveController.value.abort();
        autoSaveController.value = null;
      }
    };

    const maybeSwitchProject = async (
      projectName: string
    ): Promise<string | undefined> => {
      editorStore.projectContextReady = false;
      try {
        if (!isValidProjectName(projectName)) {
          return;
        }
        const project = await projectStore.getOrFetchProjectByName(projectName);
        // Fetch IAM policy to ensure permission checks work correctly
        await projectIamPolicyStore.getOrFetchProjectIamPolicy(project.name);
        editorStore.setProject(project.name);
        await sqlEditorEvents.emit("project-context-ready", {
          project: project.name,
        });
        return project.name;
      } catch {
        // Nothing
      } finally {
        editorStore.projectContextReady = true;
      }
    };

    const maybeUpdateWorksheet = async ({
      tabId,
      worksheet,
      title,
      database,
      statement,
      folders,
      signal,
    }: {
      tabId: string;
      worksheet?: string;
      title?: string;
      database: string;
      statement: string;
      folders?: string[];
      signal?: AbortSignal;
    }): Promise<SQLEditorTab | undefined> => {
      const connection = await extractWorksheetConnection({ database });

      // `title === undefined` means "don't change the title" — preserves the
      // current title on auto-save calls from useSQLEditorAutoSave which
      // never pass one. `title === ""` is a real, explicit empty title that
      // should be persisted (renders as the Untitled placeholder elsewhere).
      const currentSheet = worksheet
        ? worksheetStore.getWorksheetByName(worksheet)
        : undefined;
      if (worksheet && !currentSheet) {
        return;
      }
      const worksheetTitle = title ?? currentSheet?.title ?? "";

      if (worksheet && currentSheet) {
        const updated = await worksheetStore.patchWorksheet(
          {
            ...currentSheet,
            title: worksheetTitle,
            database,
            content: new TextEncoder().encode(statement),
          },
          ["title", "database", "content"],
          signal
        );
        if (!updated) {
          return;
        }
        if (!isUndefined(folders)) {
          await worksheetStore.upsertWorksheetOrganizer(
            {
              worksheet: updated.name,
              folders: folders,
            },
            ["folders"]
          );
        }
      }

      return tabStore.updateTab(tabId, {
        status: "CLEAN",
        connection,
        title: worksheetTitle,
        worksheet,
      });
    };

    const createWorksheet = async ({
      tabId,
      title,
      statement = "",
      folders = [],
      database = "",
    }: {
      tabId?: string;
      title?: string;
      statement?: string;
      folders?: string[];
      database?: string;
    }): Promise<SQLEditorTab | undefined> => {
      const worksheetTitle = title ?? "";
      const connection = await extractWorksheetConnection({ database });

      const newWorksheet = await worksheetStore.createWorksheet(
        create(WorksheetSchema, {
          title: worksheetTitle,
          database,
          content: new TextEncoder().encode(statement),
          project: editorStore.project,
          visibility: Worksheet_Visibility.PRIVATE,
        })
      );

      if (folders.length > 0) {
        await worksheetStore.upsertWorksheetOrganizer(
          {
            worksheet: newWorksheet.name,
            folders: folders,
          },
          ["folders"]
        );
      }

      if (tabId) {
        return tabStore.updateTab(tabId, {
          status: "CLEAN",
          title: worksheetTitle,
          statement,
          connection,
          worksheet: newWorksheet.name,
        });
      } else {
        const tab = await openWorksheetByName({
          worksheet: newWorksheet.name,
          forceNewTab: true,
        });
        nextTick(() => {
          if (tab && !tab.connection?.database) {
            useSQLEditorReactStore.getState().setShowConnectionPanel(true);
          }
        });
        return tab;
      }
    };

    // ---- shared filter ------------------------------------------------------

    const filter = useDynamicLocalStorage<WorksheetFilter>(
      computed(() =>
        storageKeySqlEditorWorksheetFilter(project.value, me.value.email)
      ),
      { ...INITIAL_FILTER }
    );
    const filterChanged = computed(
      () => !isEqual(filter.value, INITIAL_FILTER)
    );

    // Safe computed ref — useDynamicLocalStorage can yield null during SSR
    // or on first boot; fall back to INITIAL_FILTER so consumers always get
    // a non-null WorksheetFilter.
    const safeFilter = computed<WorksheetFilter>(
      () => filter.value ?? { ...INITIAL_FILTER }
    );

    // ---- per-view contexts (lazily initialised) -----------------------------
    // Use a plain Map (not reactive) — the ViewContext objects have their own
    // internal reactive state. Wrapping in reactive() would auto-unwrap refs.

    const contexts = new Map<SheetViewMode, ViewContext>();

    const ensureContext = (view: SheetViewMode): ViewContext => {
      if (!contexts.has(view)) {
        contexts.set(view, buildViewContext(view, safeFilter));
      }
      return contexts.get(view)!;
    };

    const getContextByView = (view: SheetViewMode) => ensureContext(view);

    // Eagerly build all three so watchers/selection logic works on startup.
    const viewContexts = {
      get my() {
        return ensureContext("my");
      },
      get shared() {
        return ensureContext("shared");
      },
      get draft() {
        return ensureContext("draft");
      },
    };

    // ---- shared tree-selection state ----------------------------------------

    const expandedKeys = useDynamicLocalStorage<Set<string>>(
      computed(() =>
        storageKeySqlEditorWorksheetTree(project.value, me.value.email)
      ),
      new Set([
        ensureContext("my").folderContext.rootPath.value,
        ensureContext("shared").folderContext.rootPath.value,
        ensureContext("draft").folderContext.rootPath.value,
      ])
    );
    const selectedKeys = ref<string[]>([]);

    const isWorksheetCreator = (worksheet: { creator: string }) =>
      worksheet.creator === `users/${me.value.email}`;

    watch(
      () => ({
        tabId: tabStore.currentTab?.id,
        worksheetName: tabStore.currentTab?.worksheet,
      }),
      async ({ tabId, worksheetName }) => {
        selectedKeys.value = [];
        if (!tabId) {
          return;
        }

        let viewMode: SheetViewMode = "draft";
        let worksheetLikeItem: WorksheetLikeItem | undefined;

        if (worksheetName) {
          const worksheet = worksheetV1Store.getWorksheetByName(worksheetName);
          if (!worksheet) {
            return;
          }
          worksheetLikeItem = {
            name: worksheet.name,
            title: worksheet.title,
            folders: worksheet.folders,
            type: "worksheet",
          };
          viewMode = isWorksheetCreator(worksheet) ? "my" : "shared";
        } else {
          worksheetLikeItem = {
            name: tabId,
            folders: [],
            title: "",
            type: "draft",
          };
        }

        const viewContext = ensureContext(viewMode);
        const key = viewContext.getKeyForWorksheet(worksheetLikeItem);
        selectedKeys.value = [key];

        for (const path of viewContext.getPathesForWorksheet(
          worksheetLikeItem
        )) {
          expandedKeys.value.add(path);
        }

        await nextTick();
        const dom = document.querySelector(`[data-item-key="${key}"]`);
        if (dom) {
          scrollIntoView(dom, {
            scrollMode: "if-needed",
            block: "nearest",
          });
        }
      },
      { immediate: true }
    );

    // ---- editing node -------------------------------------------------------

    const editingNode = ref<
      { node: WorksheetFolderNode; rawLabel: string } | undefined
    >();

    // ---- batch operations ---------------------------------------------------

    const batchUpdateWorksheetFolders = async (
      worksheets: { name: string; folders: string[] }[]
    ) => {
      if (worksheets.length === 0) {
        return;
      }
      await worksheetV1Store.batchUpsertWorksheetOrganizers(
        worksheets.map((worksheet) => ({
          organizer: {
            worksheet: worksheet.name,
            folders: worksheet.folders,
          },
          updateMask: ["folders"],
        }))
      );
    };

    // ---- view ref (preserved for SheetContext compat) -----------------------

    const view = ref<SheetViewMode>("my");

    return {
      // Save-flow actions
      autoSaveController,
      abortAutoSave,
      maybeSwitchProject,
      maybeUpdateWorksheet,
      createWorksheet,
      // SheetContext-compatible fields
      view,
      viewContexts,
      filter,
      filterChanged,
      expandedKeys,
      selectedKeys,
      editingNode,
      isWorksheetCreator,
      batchUpdateWorksheetFolders,
      // view context accessor
      getContextByView,
    };
  }
);
