import { Trans, useTranslation } from "react-i18next";
import { v4 as uuidv4 } from "uuid";
import { Alert } from "@/react/components/ui/alert";
import { Button } from "@/react/components/ui/button";
import { useVueState } from "@/react/hooks/useVueState";
import { applyPlanTitleToQuery } from "@/react/lib/plan/title";
import { router } from "@/router";
import { PROJECT_V1_ROUTE_PLAN_DETAIL_SPEC_DETAIL } from "@/router/dashboard/projectV1";
import {
  pushNotification,
  useDatabaseV1Store,
  useProjectV1Store,
  useSQLEditorStore,
  useSQLEditorTabStore,
  useStorageStore,
} from "@/store";
import { unknownProject } from "@/types";
import type { Database } from "@/types/proto-es/v1/database_service_pb";
import {
  extractDatabaseResourceName,
  extractProjectResourceName,
  getDatabaseEnvironment,
} from "@/utils";
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
  const tabStore = useSQLEditorTabStore();
  const editorStore = useSQLEditorStore();
  const databaseStore = useDatabaseV1Store();
  const projectStore = useProjectV1Store();
  const storageStore = useStorageStore();

  const allowAdmin = useVueState(() => editorStore.allowAdmin);

  const environment = database ? getDatabaseEnvironment(database) : undefined;

  const gotoCreateIssue = async () => {
    const connectedDatabase = tabStore.currentTab?.connection.database ?? "";
    if (!connectedDatabase) {
      pushNotification({
        module: "bytebase",
        style: "CRITICAL",
        title: t("sql-editor.no-database-selected"),
      });
      return;
    }

    onClose();

    const db = await databaseStore.getOrFetchDatabaseByName(connectedDatabase);
    // Fall back to `unknownProject()` (enforceIssueTitle=true) if the project
    // fetch rejects so the plan page opens with a blank title.
    const project = await projectStore
      .getOrFetchProjectByName(db.project)
      .catch(() => unknownProject());

    const tab = tabStore.currentTab;
    const statement = tab?.selectedStatement || tab?.statement || "";
    const sqlStorageKey = `bb.issues.sql.${uuidv4()}`;
    storageStore.put(sqlStorageKey, statement);
    const { databaseName } = extractDatabaseResourceName(db.name);

    const query: Record<string, string> = {
      template: "bb.plan.change-database",
      databaseList: db.name,
      sqlStorageKey,
    };
    applyPlanTitleToQuery(
      query,
      project,
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
