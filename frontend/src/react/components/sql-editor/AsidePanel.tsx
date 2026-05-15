import { useCallback } from "react";
import { useTranslation } from "react-i18next";
import { ProjectSelect } from "@/react/components/ProjectSelect";
import { useVueState } from "@/react/hooks/useVueState";
import { useSQLEditorStore } from "@/react/stores/sqlEditor";
import { useSQLEditorVueState } from "@/react/stores/sqlEditor/editor-vue-state";
import { useSQLEditorTabStore } from "@/react/stores/sqlEditor/tab-vue-state";
import { router } from "@/router";
import { PROJECT_V1_ROUTE_DASHBOARD } from "@/router/dashboard/workspaceRoutes";
import { useActuatorV1Store } from "@/store";
import { defaultProject, isValidProjectName } from "@/types";
import { hasProjectPermissionV2, hasWorkspacePermissionV2 } from "@/utils";
import { AccessPane } from "./AccessPane";
import { ActionBar } from "./AsidePanel/ActionBar";
import { GutterBar } from "./GutterBar";
import { HistoryPane } from "./HistoryPane";
import { SchemaPane } from "./SchemaPane/SchemaPane";
import { WorksheetPane } from "./WorksheetPane";

/**
 * Replaces `frontend/src/views/sql-editor/AsidePanel/AsidePanel.vue`.
 *
 * Three-column shell:
 *   1. GutterBar (vertical icon rail) — fixed.
 *   2. ActionBar — only when `asidePanelTab === "SCHEMA"` and the tab is
 *      connected to a database. Vertical button column for view drill-downs.
 *   3. Main column — project picker on top + active pane (Worksheet /
 *      Schema / History / Access) below.
 *
 * Schema-viewer modal stays in `SQLEditorHomePage.vue` (Vue parent) since
 * the embedded `TableSchemaViewer` is Vue-only; the React side triggers
 * it via the `show-schema-viewer` event on `sqlEditorEvents`.
 */
export function AsidePanel() {
  const { t } = useTranslation();
  const editorStore = useSQLEditorVueState();
  const actuatorStore = useActuatorV1Store();
  const tabStore = useSQLEditorTabStore();
  const maybeSwitchProject = useSQLEditorStore((s) => s.maybeSwitchProject);

  const asidePanelTab = useSQLEditorStore((s) => s.asidePanelTab);
  const isDisconnected = useVueState(() => tabStore.isDisconnected);
  const project = useVueState(() => editorStore.project);
  const projectContextReady = useVueState(
    () => editorStore.projectContextReady
  );

  const allowAccessDefaultProject = useVueState(() => {
    const name = actuatorStore.serverInfo?.defaultProject ?? "";
    return hasProjectPermissionV2(defaultProject(name), "bb.projects.get");
  });
  const allowCreateProject = useVueState(() =>
    hasWorkspacePermissionV2("bb.projects.create")
  );

  const handleSwitchProject = useCallback(
    (name: string) => {
      if (!name || !isValidProjectName(name)) {
        editorStore.setProject("");
      } else {
        void maybeSwitchProject(name);
      }
    },
    [editorStore, maybeSwitchProject]
  );

  // Vue's `<template #empty>` — rich empty state when the user is not
  // a member of any project. Shows a "go to create" link when the user
  // has `bb.projects.create`, or an "ask the admin" hint otherwise.
  // Vue Router resolves the dashboard route to its href; rendering a
  // plain anchor lets the page navigate via the router's history side
  // effects without dragging react-router-dom into the bundle.
  const projectsHref = router.resolve({
    name: PROJECT_V1_ROUTE_DASHBOARD,
    hash: "#new",
  }).href;
  const emptyContent = (
    <div className="text-sm text-control-placeholder flex flex-col gap-1">
      <p>
        {t("sql-editor.no-project.not-member-of-any-projects")}{" "}
        {allowCreateProject ? (
          <a
            href={projectsHref}
            className="text-accent hover:underline"
            onClick={(e) => {
              e.preventDefault();
              router.push({
                name: PROJECT_V1_ROUTE_DASHBOARD,
                hash: "#new",
              });
            }}
          >
            {t("sql-editor.no-project.go-to-create")}
          </a>
        ) : null}
      </p>
      {!allowCreateProject ? (
        <p>{t("sql-editor.no-project.contact-the-admin-to-grant-access")}</p>
      ) : null}
    </div>
  );

  return (
    <div className="h-full flex flex-row overflow-hidden">
      <div className="h-full border-r shrink-0">
        <GutterBar />
      </div>
      {asidePanelTab === "SCHEMA" && !isDisconnected ? (
        <div className="h-full border-r shrink-0">
          <ActionBar />
        </div>
      ) : null}
      <div className="h-full flex-1 flex flex-col overflow-hidden">
        <div className="flex flex-row items-center gap-x-1 px-1 py-1 border-b">
          <ProjectSelect
            className="w-full project-select"
            value={project ?? ""}
            onChange={handleSwitchProject}
            disabled={!projectContextReady}
            excludeDefault={!allowAccessDefaultProject}
            emptyContent={emptyContent}
          />
        </div>

        <div className="flex-1 flex flex-row overflow-hidden">
          <div className="h-full flex-1 flex flex-col pt-1 overflow-hidden">
            {asidePanelTab === "WORKSHEET" ? <WorksheetPane /> : null}
            {asidePanelTab === "SCHEMA" ? <SchemaPane /> : null}
            {asidePanelTab === "HISTORY" ? <HistoryPane /> : null}
            {asidePanelTab === "ACCESS" ? <AccessPane /> : null}
          </div>
        </div>
      </div>
    </div>
  );
}
