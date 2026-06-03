import { Trans, useTranslation } from "react-i18next";
import { v4 as uuidv4 } from "uuid";
import { Alert } from "@/react/components/ui/alert";
import { Button } from "@/react/components/ui/button";
import { useSQLEditorAllowAdmin } from "@/react/hooks/useSQLEditorBridge";
import { applyPlanTitleToQuery } from "@/react/lib/plan/title";
import { router } from "@/react/router";
import { PROJECT_V1_ROUTE_PLAN_DETAIL_SPEC_DETAIL } from "@/react/router/handles";
import { useAppStore } from "@/react/stores/app";
import { useSQLEditorEditorState } from "@/react/stores/sqlEditor/editor";
import { getSQLEditorTabsState } from "@/react/stores/sqlEditor/tab";
import { unknownProject } from "@/types";
import type { Database } from "@/types/proto-es/v1/database_service_pb";
import {
  extractDatabaseResourceName,
  extractProjectResourceName,
  getDatabaseEnvironment,
} from "@/utils";
import { putBlob } from "@/utils/blob-storage";
import { AdminModeButton } from "./AdminModeButton";

type Props = {
  readonly database?: Database;
  readonly onClose: () => void;
};

/**
 * Replaces frontend/src/views/sql-editor/EditorCommon/ExecuteHint.vue.
 * Body of the "DDL/DML requires a plan" confirmation surface.
 */
export function ExecuteHint({ database, onClose }: Props) {
  const { t } = useTranslation();
  const project = useSQLEditorEditorState((s) => s.project);
  const allowAdmin = useSQLEditorAllowAdmin(project);

  const environment = database ? getDatabaseEnvironment(database) : undefined;

  const gotoCreateIssue = async () => {
    const tabsState = getSQLEditorTabsState();
    const currentTab = tabsState.tabsById.get(tabsState.currentTabId);
    const connectedDatabase = currentTab?.connection.database ?? "";
    if (!connectedDatabase) {
      useAppStore.getState().notify({
        module: "bytebase",
        style: "CRITICAL",
        title: t("sql-editor.no-database-selected"),
      });
      return;
    }

    onClose();

    const db = await useAppStore
      .getState()
      .getOrFetchDatabaseByName(connectedDatabase);
    // Fall back to `unknownProject()` (enforceIssueTitle=true) if the project
    // can't be resolved so the plan page opens with a blank title.
    const projectInfo =
      (await useAppStore.getState().fetchProject(db.project)) ??
      unknownProject();

    const tab = tabsState.tabsById.get(tabsState.currentTabId);
    const statement = tab?.selectedStatement || tab?.statement || "";
    const sqlStorageKey = `bb.issues.sql.${uuidv4()}`;
    void putBlob(sqlStorageKey, statement);
    const { databaseName } = extractDatabaseResourceName(db.name);

    const query: Record<string, string> = {
      template: "bb.plan.change-database",
      databaseList: db.name,
      sqlStorageKey,
    };
    applyPlanTitleToQuery(
      query,
      projectInfo,
      () => `[${databaseName}] ${t("issue.title.change-from-sql-editor")}`
    );

    const route = router.resolve({
      name: PROJECT_V1_ROUTE_PLAN_DETAIL_SPEC_DETAIL,
      params: {
        projectId: extractProjectResourceName(db.project),
        planId: "create",
        specId: "placeholder",
      },
      query,
    });
    window.open(route.fullPath, "_blank");
  };

  return (
    <div className="w-[28rem]">
      <Alert
        variant="info"
        description={
          <section className="flex flex-col gap-y-2">
            <p>{t("sql-editor.only-select-allowed")}</p>
            {database && environment && (
              <p>
                <Trans
                  t={t}
                  i18nKey="sql-editor.ddl-dml-requires-data-change-plan"
                  components={{
                    environment: (
                      <span className="font-medium">{environment.title}</span>
                    ),
                  }}
                />
              </p>
            )}
          </section>
        }
      />

      <div className="mt-4 flex justify-between">
        {allowAdmin && (
          <div className="flex justify-start items-center gap-x-2">
            <AdminModeButton onEnter={onClose} />
          </div>
        )}
        <div className="flex flex-1 justify-end items-center gap-x-2">
          <Button variant="outline" onClick={onClose}>
            {t("common.close")}
          </Button>
          <Button onClick={() => void gotoCreateIssue()}>
            {t("plan.create-plan")}
          </Button>
        </div>
      </div>
    </div>
  );
}
