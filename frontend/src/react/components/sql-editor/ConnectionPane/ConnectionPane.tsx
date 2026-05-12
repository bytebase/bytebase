import { cloneDeep } from "lodash-es";
import { ChevronDown, ChevronRight, Info, X } from "lucide-react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import {
  AdvancedSearch,
  emptySearchParams,
  type SearchParams,
} from "@/react/components/AdvancedSearch";
import { DatabaseGroupTable } from "@/react/components/DatabaseGroupTable";
import { EngineIcon } from "@/react/components/EngineIcon";
import { FeatureBadge } from "@/react/components/FeatureBadge";
import { Checkbox } from "@/react/components/ui/checkbox";
import { RadioGroup, RadioGroupItem } from "@/react/components/ui/radio-group";
import { Separator } from "@/react/components/ui/separator";
import {
  Tabs,
  TabsList,
  TabsPanel,
  TabsTrigger,
} from "@/react/components/ui/tabs";
import { Tooltip } from "@/react/components/ui/tooltip";
import { Tree, type TreeDataNode } from "@/react/components/ui/tree";
import { countVisibleRows } from "@/react/components/ui/tree-utils";
import { useCommonSearchScopeOptions } from "@/react/components/useCommonSearchScopeOptions";
import { useVueState } from "@/react/hooks/useVueState";
import { cn } from "@/react/lib/utils";
import {
  featureToRef,
  pushNotification,
  useCurrentUserV1,
  useDatabaseV1Store,
  useDBGroupStore,
  useEnvironmentV1Store,
  useInstanceV1Store,
  useProjectV1Store,
  useSQLEditorStore,
  useSQLEditorTabStore,
  useSQLEditorTreeStore,
  useSQLEditorUIStore,
} from "@/store";
import { instanceNamePrefix } from "@/store/modules/v1/common";
import type { DatabaseFilter } from "@/store/modules/v1/database";
import type {
  BatchQueryContext,
  QueryDataSourceType,
  SQLEditorTreeNode,
} from "@/types";
import {
  isValidDatabaseGroupName,
  isValidDatabaseName,
  isValidProjectName,
  UNKNOWN_ENVIRONMENT_NAME,
  unknownEnvironment,
} from "@/types";
import { Engine } from "@/types/proto-es/v1/common_pb";
import { DatabaseGroupView } from "@/types/proto-es/v1/database_group_service_pb";
import type { Database } from "@/types/proto-es/v1/database_service_pb";
import { DataSourceType } from "@/types/proto-es/v1/instance_service_pb";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import {
  extractDatabaseResourceName,
  getConnectionForSQLEditorTab,
  getInstanceResource,
  getValueFromSearchParams,
  getValuesFromSearchParams,
  instanceV1Name,
} from "@/utils";
import { sqlEditorEvents } from "@/views/sql-editor/events";
import { setConnection } from "./actions";
import {
  ConnectionContextMenu,
  type ConnectionContextMenuHandle,
} from "./ConnectionContextMenu";
import { DatabaseGroupTag } from "./DatabaseGroupTag";
import { DatabaseHoverPanel } from "./DatabaseHoverPanel/DatabaseHoverPanel";
import {
  HoverStateProvider,
  useHoverState,
  useProvideHoverState,
} from "./DatabaseHoverPanel/hover-state";
import { Label } from "./TreeNode/Label";
import { useSQLEditorTreeByEnvironment } from "./tree";

type Props = {
  readonly show: boolean;
  /**
   * Bubble paywall triggers up to the parent `ConnectionPanel`. The Vue
   * version owned the `<FeatureModal>` inside the same template as the
   * tree, but in React both Sheet and FeatureModal are Base UI dialogs
   * portaling to the same overlay layer — keeping the FeatureModal as a
   * descendant of the Sheet caused stacking-order bugs (its backdrop
   * could land below the Sheet's popup). The parent now hosts the modal
   * as a sibling of the Sheet, and `ConnectionPaneInner` notifies it via
   * this callback.
   */
  readonly onMissingFeature: (feature: PlanFeature | undefined) => void;
};

type SelectionMode = "DATABASE" | "DATABASE-GROUP";

const getDataSourceTypeLabel = (
  t: (key: string) => string,
  type: DataSourceType
): string =>
  type === DataSourceType.ADMIN
    ? t("sql-editor.batch-query.select-data-source.admin")
    : t("sql-editor.batch-query.select-data-source.readonly");

/**
 * Replaces frontend/src/views/sql-editor/ConnectionPanel/ConnectionPane/ConnectionPane.vue.
 * Environment-grouped database tree + batch-mode batch-query selection +
 * database-group selection tab. Consumes every Phase 3/4a/4b/4c artifact:
 *   - TreeNode/Label             (Phase 3)
 *   - DatabaseHoverPanel         (Phase 3)
 *   - DatabaseGroupTag           (Phase 4a)
 *   - tree.ts hook               (Phase 4a)
 *   - setConnection, useConnectionMenu, ConnectionContextMenu (Phase 4b)
 *   - FeatureModal               (Phase 4b)
 *   - DatabaseGroupTable         (Phase 4c)
 *
 * Uses the shared React `AdvancedSearch` + `useCommonSearchScopeOptions`
 * for `instance` / `label` / `engine` scope chips. Filter fields are
 * derived from `params.scopes` and forwarded to the per-environment
 * `useSQLEditorTreeByEnvironment` hook, matching the Vue behavior.
 */
export function ConnectionPane(props: Props) {
  return <ConnectionPaneWithHoverState {...props} />;
}

function ConnectionPaneWithHoverState(props: Props) {
  const hoverState = useProvideHoverState();
  return (
    <HoverStateProvider value={hoverState}>
      <ConnectionPaneInner {...props} />
    </HoverStateProvider>
  );
}

function ConnectionPaneInner({ show, onMissingFeature }: Props) {
  const { t } = useTranslation();
  const tabStore = useSQLEditorTabStore();
  const editorStore = useSQLEditorStore();
  const uiStore = useSQLEditorUIStore();
  const databaseStore = useDatabaseV1Store();
  const dbGroupStore = useDBGroupStore();
  const environmentStore = useEnvironmentV1Store();
  const projectStore = useProjectV1Store();
  const instanceStore = useInstanceV1Store();
  const treeStore = useSQLEditorTreeStore();
  const currentUser = useCurrentUserV1();

  const supportBatchMode = useVueState(() => tabStore.supportBatchMode);
  const isInBatchMode = useVueState(() => tabStore.isInBatchMode);
  const treeStoreState = useVueState(() => treeStore.state);
  const currentTab = useVueState(() => tabStore.currentTab);
  const currentUserEmail = useVueState(() => currentUser.value.email);
  const projectName = useVueState(() => editorStore.project);
  const projectContextReady = useVueState(
    () => editorStore.projectContextReady
  );
  const environmentList = useVueState(() => environmentStore.environmentList);

  const hasBatchQueryFeature = useVueState(
    () => featureToRef(PlanFeature.FEATURE_BATCH_QUERY).value
  );
  const hasDatabaseGroupFeature = useVueState(
    () => featureToRef(PlanFeature.FEATURE_DATABASE_GROUPS).value
  );

  // Paywall triggers go to the parent (lifted out of this subtree so the
  // FeatureModal portal isn't a descendant of the Sheet's portal).
  const setMissingFeature = onMissingFeature;
  const [switchingConnection, setSwitchingConnection] = useState(false);
  const [selectionMode, setSelectionMode] = useState<SelectionMode>("DATABASE");
  const [searchParams, setSearchParams] = useState<SearchParams>(() =>
    emptySearchParams()
  );
  const queryText = searchParams.query;
  const [showMissingQueryDatabases, setShowMissingQueryDatabases] = useState(
    () => readShowMissingFromStorage(currentUserEmail)
  );

  // The Vue version re-bound to a computed storage key whenever the user
  // email changed (initial hydration from anonymous → real email, or an
  // account switch). Mirror that here: when the email key changes, reload
  // from storage and skip the next write so we don't clobber the new
  // user's saved preference with the previous user's value.
  const loadedForEmailRef = useRef(currentUserEmail);
  useEffect(() => {
    if (loadedForEmailRef.current !== currentUserEmail) {
      loadedForEmailRef.current = currentUserEmail;
      setShowMissingQueryDatabases(
        readShowMissingFromStorage(currentUserEmail)
      );
      return;
    }
    writeShowMissingToStorage(currentUserEmail, showMissingQueryDatabases);
  }, [currentUserEmail, showMissingQueryDatabases]);

  // Local state, seeded `READ_ONLY` on every panel mount. Matches Vue's
  // `state.batchQueryDataSourceType: DataSourceType.READ_ONLY` initial +
  // `watch(..., { immediate: true })` that pushes the value into the tab
  // store on mount — i.e. the tab's saved data-source resets to READ_ONLY
  // each time the drawer opens.
  const [dataSourceType, setDataSourceType] = useState<QueryDataSourceType>(
    DataSourceType.READ_ONLY
  );

  const selectedDatabaseGroupNames = useMemo(
    () => currentTab?.batchQueryContext?.databaseGroups ?? [],
    [currentTab?.batchQueryContext?.databaseGroups]
  );

  // Map<databaseResourceName, groupTitle> for every database covered by
  // any currently-selected database group. Mirrors Vue's
  // `flattenSelectedDatabasesFromGroup` and drives the tree-row checkbox
  // so users can see which databases are already implicitly included via
  // group selection (rendered as checked + disabled + tooltip in batch
  // mode). useVueState — the underlying group cache mutates without the
  // store reference changing, so a deep subscription catches new
  // matchedDatabases as they arrive from the FULL-view fetch.
  const groupCoveredDatabaseTitles = useVueState(
    () => {
      const map = new Map<string, string>();
      for (const groupName of selectedDatabaseGroupNames) {
        const group = dbGroupStore.getDBGroupByName(groupName);
        if (!isValidDatabaseGroupName(group.name)) continue;
        for (const m of group.matchedDatabases) {
          map.set(m.name, group.title);
        }
      }
      return map;
    },
    { deep: true }
  );

  const selectedDatabaseNames = useMemo(() => {
    const databases = currentTab?.batchQueryContext?.databases ?? [];
    if (
      databases.length === 0 &&
      selectedDatabaseGroupNames.length === 0 &&
      currentTab?.connection.database
    ) {
      return [currentTab.connection.database];
    }
    return databases;
  }, [
    currentTab?.batchQueryContext?.databases,
    currentTab?.connection.database,
    selectedDatabaseGroupNames.length,
  ]);

  const projectTitle = useVueState(() => {
    const p = editorStore.project;
    if (!p) return "";
    return projectStore.getProjectByName(p).title;
  });

  const scopeOptions = useCommonSearchScopeOptions([
    "instance",
    "label",
    "engine",
  ]);

  // Derive DatabaseFilter fields from the search scopes, matching the Vue
  // version's slicing of `state.params` into `instance`, `labels`, and
  // `engines`.
  const selectedLabels = useMemo(
    () => getValuesFromSearchParams(searchParams, "label"),
    [searchParams]
  );
  const selectedInstance = useMemo(
    () =>
      getValueFromSearchParams(searchParams, "instance", instanceNamePrefix),
    [searchParams]
  );
  const selectedEngines = useMemo<Engine[]>(
    () =>
      getValuesFromSearchParams(searchParams, "engine")
        .map((name) => Engine[name as keyof typeof Engine])
        .filter((v): v is Engine => typeof v === "number"),
    [searchParams]
  );

  // Per-environment tree hooks are mounted by `EnvironmentTreeSection`;
  // this component only decides which environments to render and holds the
  // "show missing query" toggle.
  const filter = useMemo(
    () => ({
      query: queryText,
      instance: selectedInstance || undefined,
      labels: selectedLabels.length > 0 ? selectedLabels : undefined,
      engines: selectedEngines.length > 0 ? selectedEngines : undefined,
    }),
    [queryText, selectedInstance, selectedLabels, selectedEngines]
  );

  // Vue's `watch(() => props.show, ..., { immediate: true })` only fires
  // when `show` toggles, NOT when batch-mode selections change. Mirror
  // that — depend on `show` alone so toggling a checkbox inside the panel
  // doesn't yank the user back to the DATABASE tab. Read selection
  // lengths through refs so they don't widen the dep array.
  const selectedDatabaseNamesRef = useRef(selectedDatabaseNames);
  selectedDatabaseNamesRef.current = selectedDatabaseNames;
  const selectedDatabaseGroupNamesRef = useRef(selectedDatabaseGroupNames);
  selectedDatabaseGroupNamesRef.current = selectedDatabaseGroupNames;
  useEffect(() => {
    if (!show) return;
    if (selectedDatabaseNamesRef.current.length > 0) {
      setSelectionMode("DATABASE");
    } else if (selectedDatabaseGroupNamesRef.current.length > 0) {
      setSelectionMode("DATABASE-GROUP");
    } else {
      setSelectionMode("DATABASE");
    }
  }, [show]);

  // Keep the `dataSourceType` in the tab store aligned with the UI.
  useEffect(() => {
    tabStore.updateBatchQueryContext({ dataSourceType });
  }, [tabStore, dataSourceType]);

  // Pre-fetch display data for currently-selected databases (so tags render
  // with the right title immediately).
  useEffect(() => {
    if (!currentTab) return;
    void databaseStore.batchGetOrFetchDatabases(selectedDatabaseNames);
  }, [currentTab, databaseStore, selectedDatabaseNames]);

  // Drive treeStore.state transitions so the mask spinner lifts when the
  // project is ready and hides again when the project changes.
  // `tree-ready` is NOT emitted here — at this point each env section is
  // still asynchronously fetching+building its slice of the tree, and the
  // listener uses `treeStore.nodeKeysByTarget` which would return [] for
  // the current connection. Each `EnvironmentTreeSection` emits
  // `tree-ready` when its own buildTree resolves.
  useEffect(() => {
    if (!isValidProjectName(projectName)) return;
    if (!projectContextReady) {
      treeStore.state = "LOADING";
      return;
    }
    treeStore.state = "READY";
  }, [projectName, projectContextReady, treeStore]);

  // Highlight the current tab's connection node in the tree. Mirrors
  // Vue's `getSelectedKeys` + `tree-ready` event listener: when the tree
  // becomes ready, resolve the current connection (database, then
  // instance fallback) and ask `treeStore.nodeKeysByTarget` for the keys
  // that point to that target.
  const [selectedKeys, setSelectedKeys] = useState<string[]>([]);
  useEffect(() => {
    let cancelled = false;
    const compute = async () => {
      const connection = tabStore.currentTab?.connection;
      if (!connection) {
        if (!cancelled) setSelectedKeys([]);
        return;
      }
      if (connection.database) {
        const database = await databaseStore.getOrFetchDatabaseByName(
          connection.database
        );
        if (cancelled) return;
        setSelectedKeys(treeStore.nodeKeysByTarget("database", database));
        return;
      }
      if (connection.instance) {
        const instance = instanceStore.getInstanceByName(connection.instance);
        if (cancelled) return;
        setSelectedKeys(treeStore.nodeKeysByTarget("instance", instance));
        return;
      }
      if (!cancelled) setSelectedKeys([]);
    };
    const unsubscribe = sqlEditorEvents.on("tree-ready", () => {
      void compute();
    });
    // The tree may already be ready by the time we mount (mount-after-emit
    // race). Compute eagerly so we don't miss the initial highlight.
    void compute();
    return () => {
      cancelled = true;
      unsubscribe();
    };
  }, [tabStore, databaseStore, instanceStore, treeStore]);

  // Context-menu imperative handle.
  const contextMenuRef = useRef<ConnectionContextMenuHandle>(null);

  const connect = useCallback(
    (node: SQLEditorTreeNode) => {
      if (node.disabled || node.meta.type !== "database") return;
      const database = (node as SQLEditorTreeNode<"database">).meta
        .target as Database;
      const batchQueryDatabases = [
        ...selectedDatabaseGroupNames,
        database.name,
      ].filter(isValidDatabaseName);
      setConnection({
        database,
        mode: currentTab?.mode,
        newTab: false,
        batchQueryContext: {
          databases: [...new Set(batchQueryDatabases)],
        },
      });
      // Mirror the Vue ConnectionPane: clicking a row commits the
      // connection AND closes the drawer. Without this the drawer stays
      // open and the user can't see the schema panel underneath, so
      // subsequent re-clicks look like nothing happened.
      uiStore.showConnectionPanel = false;
    },
    [selectedDatabaseGroupNames, currentTab?.mode, uiStore]
  );

  const onBatchQueryContextChange = useCallback(
    async (ctx: BatchQueryContext): Promise<boolean> => {
      setSwitchingConnection(true);
      try {
        const queryable = await getQueryableDatabase(
          ctx,
          databaseStore,
          dbGroupStore
        );
        const currentConnection = getConnectionForSQLEditorTab(
          tabStore.currentTab
        );
        if (
          !currentConnection.database?.name ||
          currentConnection.database?.name !== queryable?.name
        ) {
          setConnection({
            database: queryable,
            mode: "WORKSHEET",
            newTab: false,
            batchQueryContext: ctx,
          });
        } else {
          tabStore.updateBatchQueryContext(ctx);
        }
        return !!queryable;
      } finally {
        setSwitchingConnection(false);
      }
    },
    [tabStore, databaseStore, dbGroupStore]
  );

  const handleToggleDatabase = useCallback(
    async (database: string, check: boolean) => {
      let dbs: string[] = [...selectedDatabaseNames];
      if (check) {
        if (!dbs.includes(database)) dbs.push(database);
        if (dbs.length > 1 && !hasBatchQueryFeature) {
          setMissingFeature(PlanFeature.FEATURE_BATCH_QUERY);
          return;
        }
      } else {
        dbs = dbs.filter((d) => d !== database);
      }
      const ctx: BatchQueryContext = cloneDeep({
        ...currentTab?.batchQueryContext,
        databases: dbs,
      });
      await onBatchQueryContextChange(ctx);
    },
    [
      selectedDatabaseNames,
      hasBatchQueryFeature,
      currentTab?.batchQueryContext,
      onBatchQueryContextChange,
    ]
  );

  const handleSelectDatabaseGroup = useCallback(
    async (name: string) => {
      const ctx: BatchQueryContext = cloneDeep({
        ...(currentTab?.batchQueryContext ?? { databases: [] }),
        databaseGroups: [name],
      });
      const ok = await onBatchQueryContextChange(ctx);
      if (ok) {
        uiStore.showConnectionPanel = false;
      } else {
        pushNotification({
          module: "bytebase",
          style: "CRITICAL",
          title: t("sql-editor.no-queriable-database"),
        });
      }
    },
    [currentTab?.batchQueryContext, onBatchQueryContextChange, uiStore, t]
  );

  const handleSelectedGroupsChange = useCallback(
    async (next: string[]) => {
      if (next.length > 0) {
        if (!hasBatchQueryFeature) {
          setMissingFeature(PlanFeature.FEATURE_BATCH_QUERY);
          return;
        }
        if (!hasDatabaseGroupFeature) {
          setMissingFeature(PlanFeature.FEATURE_DATABASE_GROUPS);
          return;
        }
      }
      const ctx: BatchQueryContext = cloneDeep({
        ...(currentTab?.batchQueryContext ?? { databases: [] }),
        databaseGroups: next,
      });
      const ok = await onBatchQueryContextChange(ctx);
      if (next.length > 0 && !ok) {
        pushNotification({
          module: "bytebase",
          style: "CRITICAL",
          title: t("sql-editor.no-queriable-database"),
        });
      }
    },
    [
      hasBatchQueryFeature,
      hasDatabaseGroupFeature,
      currentTab?.batchQueryContext,
      onBatchQueryContextChange,
      t,
    ]
  );

  // Vue routes uncheck through `onDatabaseGroupSelectionUpdate`, which
  // performs the feature gate + notification. Mirror that here so feature
  // checks fire on remove too.
  const handleUncheckDatabaseGroup = useCallback(
    async (groupName: string) => {
      const next = selectedDatabaseGroupNames.filter((n) => n !== groupName);
      await handleSelectedGroupsChange(next);
    },
    [selectedDatabaseGroupNames, handleSelectedGroupsChange]
  );

  return (
    <div className="sql-editor-tree h-full relative">
      {supportBatchMode && (
        <BatchModeHeader
          projectTitle={projectTitle}
          selectedDatabaseCount={selectedDatabaseNames.length}
          selectedDatabaseNames={selectedDatabaseNames}
          selectedDatabaseGroupNames={selectedDatabaseGroupNames}
          switchingConnection={switchingConnection}
          hasDatabaseGroupFeature={hasDatabaseGroupFeature}
          isInBatchMode={isInBatchMode}
          dataSourceType={dataSourceType}
          onDataSourceTypeChange={setDataSourceType}
          onToggleDatabase={handleToggleDatabase}
          onUncheckDatabaseGroup={handleUncheckDatabaseGroup}
        />
      )}

      {currentTab && <Separator className="my-3" />}

      <Tabs
        value={selectionMode}
        onValueChange={(v) => setSelectionMode(v as SelectionMode)}
        className="px-4"
      >
        <TabsList>
          <TabsTrigger value="DATABASE">{t("common.databases")}</TabsTrigger>
          <TabsTrigger
            value="DATABASE-GROUP"
            disabled={!hasBatchQueryFeature || !hasDatabaseGroupFeature}
          >
            <Tooltip
              content={
                hasBatchQueryFeature && hasDatabaseGroupFeature
                  ? ""
                  : t("subscription.contact-to-upgrade")
              }
            >
              {t("common.database-group")}
            </Tooltip>
          </TabsTrigger>
        </TabsList>

        {/*
          `keepMounted` keeps the inactive Tabs.Panel attached to the DOM
          instead of unmounting it. Without this, switching to the
          "Database Group" tab unmounts `EnvironmentTreeSection`s and
          tabbing back remounts them — which re-runs their mount-effect
          and fires fresh `listDatabases` RPCs every time.
        */}
        <TabsPanel value="DATABASE" keepMounted>
          <div className="flex flex-col gap-y-1">
            <AdvancedSearch
              params={searchParams}
              scopeOptions={scopeOptions}
              placeholder={t("database.filter-database")}
              onParamsChange={setSearchParams}
            />
            {treeStoreState === "READY" && (
              <div className="flex flex-col gap-y-2 text-sm select-none pt-1">
                <label className="inline-flex items-center gap-x-2">
                  <Checkbox
                    checked={showMissingQueryDatabases}
                    onCheckedChange={(checked) =>
                      setShowMissingQueryDatabases(checked)
                    }
                  />
                  {t("sql-editor.show-databases-without-query-permission")}
                </label>

                {[...environmentList, unknownEnvironment()].map((env) => (
                  <EnvironmentTreeSection
                    key={env.name}
                    environmentName={env.name}
                    email={currentUserEmail}
                    filter={filter}
                    showMissingQueryDatabases={showMissingQueryDatabases}
                    projectContextReady={projectContextReady}
                    query={queryText}
                    isUnknownEnvironment={env.name === UNKNOWN_ENVIRONMENT_NAME}
                    selectedDatabaseNames={selectedDatabaseNames}
                    groupCoveredDatabaseTitles={groupCoveredDatabaseTitles}
                    selectedKeys={selectedKeys}
                    switchingConnection={switchingConnection}
                    onConnect={connect}
                    onToggleDatabase={handleToggleDatabase}
                    onContextMenu={(node, e) =>
                      contextMenuRef.current?.show(node, e)
                    }
                  />
                ))}
              </div>
            )}
          </div>
        </TabsPanel>

        {/*
          Only mount the DATABASE-GROUP panel when the user's plan
          actually allows the tab. The trigger above is `disabled` for
          plans missing either feature flag, but `keepMounted` on the
          panel would otherwise still mount `DatabaseGroupTable`, whose
          mount effect fires `fetchDBGroupListByProjectName` — wasted
          API traffic on every ConnectionPane open and a chance to
          surface 403s for an inaccessible UI path.
        */}
        {hasBatchQueryFeature && hasDatabaseGroupFeature && (
          <TabsPanel value="DATABASE-GROUP" keepMounted>
            <DatabaseGroupTable
              projectName={projectName}
              view={DatabaseGroupView.FULL}
              leadingLabel={t("database-group.select")}
              showSelection
              showExternalLink
              selectedDatabaseGroupNames={selectedDatabaseGroupNames}
              onSelectedDatabaseGroupNamesChange={handleSelectedGroupsChange}
              onRowClick={(_, group) => handleSelectDatabaseGroup(group.name)}
            />
          </TabsPanel>
        )}
      </Tabs>

      <ConnectionContextMenu ref={contextMenuRef} />

      <DatabaseHoverPanel offsetX={10} offsetY={4} margin={4} />

      {treeStoreState !== "READY" && (
        <div className="absolute inset-0 bg-white/75 flex items-center justify-center">
          <span className="text-control text-sm">
            {t("sql-editor.loading-databases")}
          </span>
        </div>
      )}
    </div>
  );
}

// ---- Batch-mode header --------------------------------------------------

function BatchModeHeader({
  projectTitle,
  selectedDatabaseCount,
  selectedDatabaseNames,
  selectedDatabaseGroupNames,
  switchingConnection,
  hasDatabaseGroupFeature,
  isInBatchMode,
  dataSourceType,
  onDataSourceTypeChange,
  onToggleDatabase,
  onUncheckDatabaseGroup,
}: {
  projectTitle: string;
  selectedDatabaseCount: number;
  selectedDatabaseNames: string[];
  selectedDatabaseGroupNames: string[];
  switchingConnection: boolean;
  hasDatabaseGroupFeature: boolean;
  isInBatchMode: boolean;
  dataSourceType: QueryDataSourceType;
  onDataSourceTypeChange: (v: QueryDataSourceType) => void;
  onToggleDatabase: (name: string, check: boolean) => void;
  onUncheckDatabaseGroup: (name: string) => void;
}) {
  const { t } = useTranslation();
  const databaseStore = useDatabaseV1Store();

  return (
    <div className="w-full px-4 mt-4">
      <div className="text-control-light text-sm mb-2 w-full leading-4 flex flex-col items-start gap-x-1">
        <div className="flex items-center gap-x-1">
          <FeatureBadge feature={PlanFeature.FEATURE_BATCH_QUERY} />
          {t("sql-editor.batch-query.description", {
            database: selectedDatabaseCount,
            group: selectedDatabaseGroupNames.length,
            project: projectTitle,
          })}
        </div>
      </div>

      <div className="w-full mt-1 flex flex-row justify-start items-start flex-wrap gap-2">
        {selectedDatabaseNames.map((db) => (
          <SelectedDatabaseTag
            key={db}
            name={db}
            disabled={switchingConnection}
            onClose={() => onToggleDatabase(db, false)}
            resolveDatabase={() => databaseStore.getDatabaseByName(db)}
          />
        ))}
        {hasDatabaseGroupFeature &&
          selectedDatabaseGroupNames.map((name) => (
            <DatabaseGroupTag
              key={name}
              databaseGroupName={name}
              disabled={switchingConnection}
              onUncheck={onUncheckDatabaseGroup}
            />
          ))}
        {isInBatchMode && <Separator className="my-2 w-full" />}
        {isInBatchMode && (
          <div className="w-full">
            <div className="text-control-light text-sm flex items-center gap-x-1">
              {t("sql-editor.batch-query.select-data-source.self")}
              <Tooltip
                content={t("sql-editor.batch-query.select-data-source.tooltip")}
              >
                <Info className="size-4" />
              </Tooltip>
            </div>
            <RadioGroup
              value={String(dataSourceType)}
              onValueChange={(v) =>
                onDataSourceTypeChange(Number(v) as QueryDataSourceType)
              }
            >
              <RadioGroupItem value={String(DataSourceType.ADMIN)}>
                {getDataSourceTypeLabel(t, DataSourceType.ADMIN)}
              </RadioGroupItem>
              <RadioGroupItem value={String(DataSourceType.READ_ONLY)}>
                {getDataSourceTypeLabel(t, DataSourceType.READ_ONLY)}
              </RadioGroupItem>
            </RadioGroup>
          </div>
        )}
      </div>
    </div>
  );
}

function SelectedDatabaseTag({
  name,
  disabled,
  onClose,
  resolveDatabase,
}: {
  name: string;
  disabled: boolean;
  onClose: () => void;
  resolveDatabase: () => ReturnType<
    ReturnType<typeof useDatabaseV1Store>["getDatabaseByName"]
  >;
}) {
  const { t } = useTranslation();
  const database = useVueState(resolveDatabase);
  const instance = useMemo(() => {
    if (!database) return null;
    return getInstanceResource(database);
  }, [database]);
  const dbLabel = database
    ? extractDatabaseResourceName(database.name).databaseName
    : extractDatabaseResourceName(name).databaseName;

  return (
    <span
      className={cn(
        "inline-flex items-center gap-x-1 rounded-sm border border-control-border bg-control-bg/60 pl-2 pr-1 py-0.5 text-sm",
        disabled && "opacity-50"
      )}
    >
      {instance && <EngineIcon engine={instance.engine} className="size-4" />}
      {instance && (
        <span className="truncate max-w-[8rem]">
          {instanceV1Name(instance)}
        </span>
      )}
      <ChevronRight className="size-3 shrink-0" />
      <span className="truncate max-w-[10rem]">{dbLabel}</span>
      <button
        type="button"
        className={cn(
          "inline-flex items-center justify-center size-4 rounded-sm",
          "hover:bg-control-bg-hover disabled:opacity-50 disabled:cursor-not-allowed"
        )}
        aria-label={t("common.close")}
        disabled={disabled}
        onClick={(e) => {
          e.stopPropagation();
          if (!disabled) onClose();
        }}
      >
        <X className="size-3" />
      </button>
    </span>
  );
}

// ---- Per-environment sub-tree ------------------------------------------

function EnvironmentTreeSection(props: {
  environmentName: string;
  email: string;
  filter: DatabaseFilter;
  showMissingQueryDatabases: boolean;
  projectContextReady: boolean;
  query: string;
  isUnknownEnvironment: boolean;
  selectedDatabaseNames: string[];
  /** databaseResourceName → groupTitle for every database implicitly
   *  selected via a chosen database group. The row checkbox unions this
   *  with `selectedDatabaseNames` and uses the title for the disabled-
   *  checkbox tooltip. */
  groupCoveredDatabaseTitles: Map<string, string>;
  /** Tree-row keys to highlight as the current connection (1:1 with Vue
   *  `selectedKeys` from `getSelectedKeys()`). */
  selectedKeys: string[];
  switchingConnection: boolean;
  onConnect: (node: SQLEditorTreeNode) => void;
  onToggleDatabase: (name: string, check: boolean) => void;
  onContextMenu: (node: SQLEditorTreeNode, e: React.MouseEvent) => void;
}) {
  const {
    environmentName,
    email,
    filter,
    showMissingQueryDatabases,
    projectContextReady,
    query,
    isUnknownEnvironment,
    selectedDatabaseNames,
    groupCoveredDatabaseTitles,
    selectedKeys,
    switchingConnection,
    onConnect,
    onToggleDatabase,
    onContextMenu,
  } = props;

  const treeByEnv = useSQLEditorTreeByEnvironment(environmentName, { email });

  // Read the latest show-missing flag from inside the async `.then()`
  // below. The fetch can outlive the render that started it, so capturing
  // `showMissingQueryDatabases` directly would let an in-flight rebuild
  // overwrite the tree with a stale flag (e.g. user toggles the checkbox
  // mid-fetch — the toggle's own buildTree runs first, then the older
  // .then() lands and clobbers it).
  const showMissingRef = useRef(showMissingQueryDatabases);
  showMissingRef.current = showMissingQueryDatabases;

  // Kick off fetch when filter / project-readiness changes. The parent
  // memoizes `filter` on (queryText, instance, labels, engines), so a
  // single dep on `filter` covers scope-chip changes too — depending only
  // on `filter.query` (as before) left scope edits stale until the user
  // also retyped the search.
  // treeByEnv identity is stable per-render via the hook's internals.
  // After buildTree resolves we emit `tree-ready` so the parent's
  // current-connection-highlight effect can recompute against the now-
  // populated `treeStore.nodeKeysByTarget`. Emitting here (per-env, after
  // populate) instead of in the project-readiness effect avoids a race
  // where the highlight is computed against an empty tree.
  useEffect(() => {
    if (!projectContextReady) return;
    void treeByEnv.prepareDatabases(filter).then(() => {
      treeByEnv.buildTree(showMissingRef.current);
      void sqlEditorEvents.emit("tree-ready");
    });
  }, [projectContextReady, filter]);

  // Rebuild when the "show missing" toggle flips.
  useEffect(() => {
    treeByEnv.buildTree(showMissingQueryDatabases);
  }, [showMissingQueryDatabases]);

  // First-time default-expand. The Vue version did this by listening to the
  // `tree-ready` event and seeding `expandedKeys` from the global tree
  // store; in React each environment owns its own expand state, so we
  // expand from the per-env tree directly. After the user toggles anything
  // (`expandedState.initialized = true`), we leave their preference alone.
  useEffect(() => {
    if (treeByEnv.expandedState.initialized) return;
    if (treeByEnv.tree.length === 0) return;
    treeByEnv.setExpandedKeys(collectAllNodeKeys(treeByEnv.tree));
  }, [treeByEnv.tree, treeByEnv.expandedState.initialized]);

  const selectedSet = useMemo(
    () => new Set(selectedDatabaseNames),
    [selectedDatabaseNames]
  );

  // Hide the env section entirely when it's the "unknown" bucket with no
  // children (matches Vue's `v-if="env !== UNKNOWN || !treeIsEmpty"` guard).
  if (isUnknownEnvironment && treeIsEmpty(treeByEnv.tree)) {
    return null;
  }

  const data = treeByEnv.tree.map(toTreeDataNode);

  const expandedKeySet = useMemo(
    () => new Set(treeByEnv.expandedState.expandedKeys),
    [treeByEnv.expandedState.expandedKeys]
  );
  // Mirror SheetTree's pattern: the Tree primitive has no internal scroll,
  // so its `height` must equal the count of visible rows × row height. The
  // parent decides whether to clip / scroll. This lets each environment
  // subtree expand naturally instead of being capped at a single row.
  // Filter is server-side via `params.scopes` + `filter.query` so we pass
  // an empty keyword + a no-op `searchMatch` predicate.
  const visibleRowCount = useMemo(
    () =>
      treeByEnv.tree.reduce(
        (total, root) =>
          total + countVisibleRows(root, expandedKeySet, "", () => false),
        0
      ),
    [treeByEnv.tree, expandedKeySet]
  );

  return (
    <div className="flex flex-col gap-y-1 pt-2 pb-2">
      <Tree<SQLEditorTreeNode>
        data={data}
        selectedIds={selectedKeys}
        expandedIds={treeByEnv.expandedState.expandedKeys}
        onToggle={(id) => {
          const next = new Set(treeByEnv.expandedState.expandedKeys);
          if (next.has(id)) next.delete(id);
          else next.add(id);
          treeByEnv.setExpandedKeys([...next]);
        }}
        height={Math.max(visibleRowCount, 1) * ROW_HEIGHT}
        rowHeight={ROW_HEIGHT}
        renderNode={({ node, style }) => {
          const data = node.data.data;
          const isDatabase = data.meta.type === "database";
          const databaseName = isDatabase
            ? (data.meta.target as { name: string }).name
            : "";
          const matchedGroupTitle =
            isDatabase && groupCoveredDatabaseTitles.has(databaseName)
              ? groupCoveredDatabaseTitles.get(databaseName)
              : undefined;
          // Distinguish "user explicitly selected this row" from
          // "implicitly included because the row belongs to a selected
          // group". Only the explicit case should tint the row blue —
          // group-implied entries are locked-on and need to read as
          // disabled, not as an active user choice.
          const userSelected = isDatabase && selectedSet.has(databaseName);
          const groupImplied = matchedGroupTitle !== undefined;
          const checkDisabled = switchingConnection || groupImplied;
          return (
            <TreeRow
              style={style}
              node={data}
              depth={node.level}
              isOpen={!!node.isOpen}
              hasChildren={!!data.children && data.children.length > 0}
              query={query}
              rowTinted={userSelected}
              checkboxChecked={userSelected || groupImplied}
              checkDisabled={checkDisabled}
              checkTooltip={
                matchedGroupTitle !== undefined ? matchedGroupTitle : undefined
              }
              onClick={(e) => {
                e.stopPropagation();
                if (isDatabase) {
                  onConnect(data);
                } else {
                  node.toggle();
                }
              }}
              onToggleChecked={(next) => {
                // Belt-and-suspenders: even if BaseUI Checkbox somehow
                // fires `onCheckedChange` while disabled, drop the
                // toggle so a group-implied row can't be unchecked.
                if (checkDisabled) return;
                if (isDatabase) {
                  onToggleDatabase(databaseName, next);
                }
              }}
              onContextMenu={(e) => onContextMenu(data, e)}
            />
          );
        }}
      />
      {treeByEnv.fetchDataState.nextPageToken &&
        treeByEnv.expandedState.expandedKeys.includes(environmentName) &&
        (!treeIsEmpty(treeByEnv.tree) || showMissingQueryDatabases) && (
          <LoadMoreButton
            loading={treeByEnv.fetchDataState.loading}
            onLoadMore={() =>
              treeByEnv.fetchDatabases(filter).then(() => {
                treeByEnv.buildTree(showMissingQueryDatabases);
              })
            }
          />
        )}
    </div>
  );
}

function TreeRow({
  style,
  node,
  isOpen,
  hasChildren,
  query,
  rowTinted,
  checkboxChecked,
  checkDisabled,
  checkTooltip,
  onClick,
  onToggleChecked,
  onContextMenu,
}: {
  style: React.CSSProperties;
  node: SQLEditorTreeNode;
  /** Whether this row is currently expanded (for non-leaf nodes). */
  isOpen: boolean;
  /** Whether the node has children (controls chevron visibility). */
  hasChildren: boolean;
  depth: number;
  query: string;
  /** Whether the row should render the active-selection tint. Only set
   *  when the user explicitly selected the row — NOT for group-implied
   *  membership, which should read as locked-on (gray) not active. */
  rowTinted: boolean;
  /** Whether the inline batch-mode checkbox renders as checked. True for
   *  user-explicit selection AND for group-implied membership. */
  checkboxChecked: boolean;
  /** True when the checkbox should be locked, e.g. mid-connection-switch
   *  or when membership is forced by a selected group. */
  checkDisabled: boolean;
  /** Tooltip shown on the disabled checkbox — typically the group title
   *  that pulls this database into the batch query implicitly. */
  checkTooltip?: string;
  onClick: (e: React.MouseEvent) => void;
  onToggleChecked: (checked: boolean) => void;
  onContextMenu: (e: React.MouseEvent) => void;
}) {
  const hoverState = useHoverState();
  const rowRef = useRef<HTMLDivElement | null>(null);
  // Tracks whether the most recent mousedown started on an interactive
  // descendant (Checkbox, RequestQueryButton). The row's onMouseUp uses
  // this to skip the row-activation when the user is interacting with a
  // child control — even if the cursor drifts between mousedown and
  // mouseup so the synthesized click never reaches the button (the
  // failure mode `stopPropagation` on the wrapper alone doesn't cover).
  const interactiveMouseDownRef = useRef(false);

  const handleMouseEnter = (e: React.MouseEvent) => {
    if (node.meta.type !== "database") return;
    const el = e.currentTarget as HTMLElement;
    const rect = el.getBoundingClientRect();
    hoverState.setPosition({ x: rect.left, y: rect.bottom });
    // Re-use the `0` override when we're already showing something — the
    // panel stays without blinking when sliding between rows.
    if (hoverState.state) {
      hoverState.update({ node }, "before", 0);
    } else {
      hoverState.update({ node }, "before");
    }
  };

  const handleMouseLeave = () => {
    hoverState.update(undefined, "after");
  };

  return (
    <div
      ref={rowRef}
      style={style}
      className={cn(
        "bb-conn-pane-row flex items-center gap-x-1 w-full pr-2 cursor-pointer rounded-sm",
        "hover:bg-control-bg/70",
        rowTinted && "bg-accent/10"
      )}
      data-node-key={node.key}
      onMouseDown={(e) => {
        const target = e.target as HTMLElement;
        // Match `data-row-interactive` first because tooltip / permission
        // wrappers (Base UI's Tooltip.Trigger renders a `<span>`) can be
        // the actual mousedown target when the wrapped control is
        // disabled — `target.closest('button')` from that span walks UP
        // and won't find the disabled `<button>` descendant. Wrappers
        // owning interactive controls in this row mark themselves with
        // `data-row-interactive` to opt out of row activation regardless
        // of internal nesting.
        interactiveMouseDownRef.current = !!target.closest(
          '[data-row-interactive], button, [role="checkbox"], a, input'
        );
      }}
      // Use `onMouseUp` instead of `onClick` for the row activation. Safari
      // drops the synthesized `click` event when `mousedown` and `mouseup`
      // resolve to different inner elements (the chevron / engine icon /
      // text span). `mouseup` always fires regardless, so we drive the
      // selection from there, gated to the primary button.
      //
      // We also skip when mousedown started on an interactive descendant
      // — otherwise pressing a button/checkbox and slightly drifting the
      // cursor before release would connect-and-close the panel before
      // the button's click ever fires.
      onMouseUp={(e) => {
        if (e.button !== 0) return;
        if (interactiveMouseDownRef.current) {
          interactiveMouseDownRef.current = false;
          return;
        }
        onClick(e);
      }}
      onMouseEnter={handleMouseEnter}
      onMouseLeave={handleMouseLeave}
      onContextMenu={(e) => {
        e.preventDefault();
        onContextMenu(e);
      }}
    >
      {/* Expand / collapse chevron — shown for non-leaf rows. Renders an
          empty spacer for leaf rows so labels stay vertically aligned. */}
      <span className="shrink-0 inline-flex size-4 items-center justify-center text-control-light">
        {hasChildren ? (
          isOpen ? (
            <ChevronDown className="size-3.5" />
          ) : (
            <ChevronRight className="size-3.5" />
          )
        ) : null}
      </span>
      <Label
        node={node}
        keyword={query}
        checked={checkboxChecked}
        checkDisabled={checkDisabled}
        checkTooltip={checkTooltip}
        onCheckedChange={onToggleChecked}
      />
    </div>
  );
}

function LoadMoreButton({
  loading,
  onLoadMore,
}: {
  loading: boolean;
  onLoadMore: () => Promise<void>;
}) {
  const { t } = useTranslation();
  return (
    <div className="w-full flex items-center justify-start pl-4">
      <button
        type="button"
        className="text-sm text-accent hover:underline disabled:opacity-50"
        disabled={loading}
        onClick={() => void onLoadMore()}
      >
        {t("common.load-more")}
      </button>
    </div>
  );
}

// ---- utilities ---------------------------------------------------------

const ROW_HEIGHT = 28;

function toTreeDataNode(
  node: SQLEditorTreeNode
): TreeDataNode<SQLEditorTreeNode> {
  return {
    id: node.key,
    data: node,
    children: node.children?.map(toTreeDataNode),
  };
}

function treeIsEmpty(nodes: SQLEditorTreeNode[]): boolean {
  if (nodes.length === 0) return true;
  const [root] = nodes;
  const children = root.children;
  return !children || children.length === 0;
}

/**
 * Walks the tree and returns every node key. Used to seed `expandedKeys`
 * the first time an environment subtree is rendered so the user sees a
 * fully-expanded list by default — matching the Vue ConnectionPane.
 */
function collectAllNodeKeys(nodes: SQLEditorTreeNode[]): string[] {
  const out: string[] = [];
  const visit = (node: SQLEditorTreeNode) => {
    out.push(node.key);
    for (const child of node.children ?? []) visit(child);
  };
  for (const root of nodes) visit(root);
  return out;
}

/**
 * 1:1 port of `getQueryableDatabase` from the Vue ConnectionPane. Despite
 * the name, the original returns the *first* database in the group — it
 * does not filter for `isDatabaseV1Queryable`. The downstream
 * `setConnection` call attempts the connection regardless; the
 * "no-queriable-database" notification is only surfaced when the function
 * returns `undefined` (i.e. no databases at all in any picked group).
 */
async function getQueryableDatabase(
  ctx: BatchQueryContext,
  databaseStore: ReturnType<typeof useDatabaseV1Store>,
  dbGroupStore: ReturnType<typeof useDBGroupStore>
) {
  if (ctx.databases.length > 0) {
    return databaseStore.getDatabaseByName(ctx.databases[0]);
  }
  for (const groupName of ctx.databaseGroups ?? []) {
    const group = dbGroupStore.getDBGroupByName(groupName);
    if (!isValidDatabaseGroupName(group.name)) continue;
    const databases = await databaseStore.batchGetOrFetchDatabases(
      group.matchedDatabases.map((d) => d.name)
    );
    if (databases.length > 0) return databases[0];
  }
  return undefined;
}

function storageKey(email: string) {
  return `bb.sql-editor.show-missing-query-db.${email || "anonymous"}`;
}

function readShowMissingFromStorage(email: string): boolean {
  try {
    const raw = localStorage.getItem(storageKey(email));
    if (raw === null) return true;
    return JSON.parse(raw) as boolean;
  } catch {
    return true;
  }
}

function writeShowMissingToStorage(email: string, value: boolean) {
  try {
    localStorage.setItem(storageKey(email), JSON.stringify(value));
  } catch {
    /* ignore */
  }
}
