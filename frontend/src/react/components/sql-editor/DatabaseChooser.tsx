import { ChevronRight, Database, SquareStack } from "lucide-react";
import { useTranslation } from "react-i18next";
import { EngineIcon } from "@/react/components/EngineIcon";
import { EnvironmentLabel } from "@/react/components/EnvironmentLabel";
import { Tooltip } from "@/react/components/ui/tooltip";
import { useVueState } from "@/react/hooks/useVueState";
import { cn } from "@/react/lib/utils";
import { useSQLEditorStore } from "@/react/stores/sqlEditor";
import {
  useConnectionOfCurrentSQLEditorTab,
  useSQLEditorStore as useSQLEditorPiniaStore,
  useSQLEditorTabStore,
} from "@/store";
import { isValidDatabaseName, isValidInstanceName } from "@/types";
import {
  extractDatabaseResourceName,
  getDatabaseEnvironment,
  getInstanceResource,
} from "@/utils";

type DatabaseChooserProps = {
  readonly disabled?: boolean;
};

/**
 * Replaces frontend/src/views/sql-editor/EditorCommon/DatabaseChooser.vue.
 * Breadcrumb-style chooser showing the current connection
 * (Environment > Instance > Database). Click opens the connection panel.
 */
export function DatabaseChooser({ disabled = false }: DatabaseChooserProps) {
  const { t } = useTranslation();
  const tabStore = useSQLEditorTabStore();
  const editorStore = useSQLEditorPiniaStore();
  const setShowConnectionPanel = useSQLEditorStore(
    (s) => s.setShowConnectionPanel
  );
  const connection = useConnectionOfCurrentSQLEditorTab();

  const currentTab = useVueState(() => tabStore.currentTab);
  const isInBatchMode = useVueState(() => tabStore.isInBatchMode);
  const projectContextReady = useVueState(
    () => editorStore.projectContextReady
  );
  const database = useVueState(() => connection.database.value);

  const instance = getInstanceResource(database);
  const environment = getDatabaseEnvironment(database);
  const databaseName = extractDatabaseResourceName(database.name).databaseName;

  const isConnected =
    !!currentTab &&
    isValidInstanceName(instance.name) &&
    isValidDatabaseName(database.name);

  const handleClick = () => {
    setShowConnectionPanel(true);
  };

  return (
    <button
      type="button"
      disabled={disabled || !projectContextReady}
      onClick={handleClick}
      className={cn(
        "inline-flex items-center justify-end gap-1 px-2 h-8 text-sm",
        "border border-accent text-accent",
        "hover:bg-accent/5 focus:bg-accent/5",
        "rounded-none first:rounded-l-xs last:rounded-r-xs",
        "[&:not(:last-child)]:border-r-0",
        "disabled:opacity-50 disabled:cursor-not-allowed",
        "transition-colors overflow-hidden"
      )}
    >
      {isConnected ? (
        <div className="flex flex-row items-center text-control truncate">
          {isInBatchMode && (
            <Tooltip content={t("sql-editor.batch-query.batch")}>
              <SquareStack className="size-4 mr-1 text-accent" />
            </Tooltip>
          )}
          <EnvironmentLabel environmentName={environment.name} />
          <ChevronRight className="size-4 shrink-0 text-control-light" />
          <div className="flex items-center gap-1">
            <EngineIcon engine={instance.engine} className="size-4" />
            <span className="truncate">{instance.title}</span>
          </div>
          <ChevronRight className="size-4 shrink-0 text-control-light" />
          <div className="flex items-center gap-1 truncate">
            <Database className="size-4 shrink-0" />
            <span className="truncate">{databaseName}</span>
          </div>
        </div>
      ) : (
        <span>{t("sql-editor.select-a-database-to-start")}</span>
      )}
    </button>
  );
}
