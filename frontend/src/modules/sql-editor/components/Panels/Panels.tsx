import { ShieldAlert } from "lucide-react";
import { useEffect, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { Alert } from "@/components/ui/alert";
import { useAppDatabaseMetadata } from "@/hooks/useAppDatabaseMetadata";
import { DatabaseChooser } from "@/modules/sql-editor/components/DatabaseChooser";
import { DataExplorerPanel } from "@/modules/sql-editor/components/DataExplorer/DataExplorerPanel";
import { DiagramPanel } from "@/modules/sql-editor/components/DiagramPanel";
import { ExternalTablesPanel } from "@/modules/sql-editor/components/ExternalTablesPanel";
import { FunctionsPanel } from "@/modules/sql-editor/components/FunctionsPanel";
import { InfoPanel } from "@/modules/sql-editor/components/InfoPanel";
import { PackagesPanel } from "@/modules/sql-editor/components/PackagesPanel";
import { ProceduresPanel } from "@/modules/sql-editor/components/ProceduresPanel";
import { SchemaSelectToolbar } from "@/modules/sql-editor/components/SchemaSelectToolbar";
import { SequencesPanel } from "@/modules/sql-editor/components/SequencesPanel";
import { StandardPanel } from "@/modules/sql-editor/components/StandardPanel/StandardPanel";
import { TablesPanel } from "@/modules/sql-editor/components/TablesPanel";
import { TerminalPanel } from "@/modules/sql-editor/components/TerminalPanel/TerminalPanel";
import { TriggersPanel } from "@/modules/sql-editor/components/TriggersPanel";
import { ViewsPanel } from "@/modules/sql-editor/components/ViewsPanel";
import { useConnectionOfCurrentSQLEditorTab } from "@/modules/sql-editor/hooks/useSQLEditorState";
import {
  useCurrentSQLEditorTab,
  useSQLEditorTabState,
} from "@/modules/sql-editor/store/tab";
import { isValidDatabaseName } from "@/types";
import { extractDatabaseResourceName } from "@/utils";
import { useViewStateNav } from "./common/useViewStateNav";

export function Panels() {
  const { t } = useTranslation();
  const { database } = useConnectionOfCurrentSQLEditorTab();

  const tab = useCurrentSQLEditorTab();
  const tabMode = useSQLEditorTabState(
    (s) => s.tabsById.get(s.currentTabId)?.mode
  );
  const databaseName = database.name;
  const view = useSQLEditorTabState(
    (s) => s.tabsById.get(s.currentTabId)?.viewState?.view
  );
  const effectiveView = tabMode === "DATA_EXPLORER" ? "CODE" : view;
  const { setSchema } = useViewStateNav();

  // Parent drives the metadata fetch for the whole Panels subtree;
  // children call `useAppDatabaseMetadata(..., { autoFetch: false })`.
  const databaseMetadata = useAppDatabaseMetadata(databaseName ?? "", {
    silent: true,
  });

  // Pin the active schema to a sensible default whenever the tab,
  // database metadata, or current schema changes.
  const tabId = tab?.id;
  const currentSchema = tab?.viewState?.schema;
  useEffect(() => {
    if (!tabId) return;
    if (
      !isValidDatabaseName(
        extractDatabaseResourceName(databaseMetadata.name).database
      )
    ) {
      return;
    }
    if (
      !currentSchema ||
      databaseMetadata.schemas.findIndex((s) => s.name === currentSchema) < 0
    ) {
      const next = databaseMetadata.schemas[0]?.name;
      if (next !== undefined) setSchema(next);
    }
  }, [tabId, databaseMetadata, currentSchema, setSchema]);

  const codePanel = useMemo(() => {
    if (!tab || tabMode === "SAVED_QUERY") {
      return <StandardPanel key={`standard-${tab?.id ?? "default"}`} />;
    }
    if (tabMode === "ADMIN") {
      return <TerminalPanel key={`terminal-${tab.id}`} />;
    }
    if (tabMode === "DATA_EXPLORER") {
      return <DataExplorerPanel key={`data-explorer-${tab.id}`} />;
    }
    return (
      <Alert variant="error" className="m-2" key={`no-permission-${tab.id}`}>
        <ShieldAlert className="size-5 shrink-0 mt-0.5" />
        <div>{t("database.access-denied")}</div>
      </Alert>
    );
  }, [tab, tabMode, t]);

  const subPanel = effectiveView
    ? renderSubPanel(effectiveView, tab?.id)
    : null;

  return (
    <div className="flex-1 flex items-stretch overflow-hidden">
      <div className="flex-1 overflow-y-hidden overflow-x-auto">
        {(!effectiveView || effectiveView === "CODE") && codePanel}
        {effectiveView && effectiveView !== "CODE" && (
          <div className="h-full flex flex-col">
            <div className="py-2 px-2 w-full flex flex-row gap-x-2 justify-between items-center">
              <div className="flex items-center justify-start gap-2">
                {/*
                  Wrap the chooser so it's the lone child of its own
                  flex container. The chooser uses
                  `[&:not(:last-child)]:border-r-0` /
                  `first:rounded-l-xs last:rounded-r-xs` to share borders
                  with adjacent buttons in a button-group context. Here
                  the next sibling is a separate Select primitive (not
                  part of the group), so without this wrapper the chooser
                  loses its right border and only gets left-rounded
                  corners.
                */}
                <div className="inline-flex">
                  <DatabaseChooser disabled />
                </div>
                <SchemaSelectToolbar />
              </div>
            </div>
            <div className="flex-1 min-h-0">{subPanel}</div>
          </div>
        )}
      </div>
    </div>
  );
}

function renderSubPanel(view: string, tabId: string | undefined) {
  const k = (suffix: string) => `${suffix}-${tabId ?? "default"}`;
  switch (view) {
    case "INFO":
      return <InfoPanel key={k("info")} />;
    case "TABLES":
      return <TablesPanel key={k("tables")} />;
    case "VIEWS":
      return <ViewsPanel key={k("views")} />;
    case "FUNCTIONS":
      return <FunctionsPanel key={k("functions")} />;
    case "PROCEDURES":
      return <ProceduresPanel key={k("procedures")} />;
    case "SEQUENCES":
      return <SequencesPanel key={k("sequences")} />;
    case "PACKAGES":
      return <PackagesPanel key={k("packages")} />;
    case "TRIGGERS":
      return <TriggersPanel key={k("triggers")} />;
    case "EXTERNAL_TABLES":
      return <ExternalTablesPanel key={k("external-tables")} />;
    case "DIAGRAM":
      return <DiagramPanel key={k("diagram")} />;
    default:
      return null;
  }
}
