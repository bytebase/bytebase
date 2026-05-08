import { ChevronLeft, Code, Columns, Eye, FileSymlink } from "lucide-react";
import { useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@/react/components/ui/button";
import { Checkbox } from "@/react/components/ui/checkbox";
import {
  Tabs,
  TabsList,
  TabsPanel,
  TabsTrigger,
} from "@/react/components/ui/tabs";
import type {
  Database,
  DatabaseMetadata,
  SchemaMetadata,
  ViewMetadata,
} from "@/types/proto-es/v1/database_service_pb";
import { STORAGE_KEY_SQL_EDITOR_CODE_VIEWER_FORMAT } from "@/utils";
import { OpenAIButton } from "../../OpenAIButton";
import { DefinitionViewer } from "../common/DefinitionViewer";
import { PanelSearchBox } from "../common/PanelSearchBox";
import { ColumnsTable } from "../common/tables/ColumnsTable";
import { DependencyColumnsTable } from "../common/tables/DependencyColumnsTable";
import { useViewStateNav } from "../common/useViewStateNav";

type Mode = "DEFINITION" | "COLUMNS" | "DEPENDENCY-COLUMNS";

interface ViewDetailProps {
  db: Database;
  database: DatabaseMetadata;
  schema: SchemaMetadata;
  view: ViewMetadata;
}

export function ViewDetail({ db, database, schema, view }: ViewDetailProps) {
  const { t } = useTranslation();
  const { detail, clearDetail } = useViewStateNav();
  const [mode, setMode] = useState<Mode>("COLUMNS");
  const [keyword, setKeyword] = useState("");
  const [selectedStatement, setSelectedStatement] = useState("");
  const [format, setFormat] = useState<boolean>(() => {
    if (typeof window === "undefined") return false;
    return (
      localStorage.getItem(STORAGE_KEY_SQL_EDITOR_CODE_VIEWER_FORMAT) === "true"
    );
  });

  useEffect(() => {
    localStorage.setItem(
      STORAGE_KEY_SQL_EDITOR_CODE_VIEWER_FORMAT,
      String(format)
    );
  }, [format]);

  // Sync the active tab with the detail anchor (column / dependency-column).
  useEffect(() => {
    if (!detail?.view) return;
    if (detail.column) {
      setMode("COLUMNS");
      return;
    }
    if (detail.dependencyColumn) {
      setMode("DEPENDENCY-COLUMNS");
      return;
    }
    setMode("DEFINITION");
  }, [detail?.view, detail?.column, detail?.dependencyColumn]);

  const tabs = useMemo(() => {
    const out: { mode: Mode; text: string; Icon: typeof Code }[] = [
      { mode: "DEFINITION", text: t("common.definition"), Icon: Code },
    ];
    if (view.columns.length > 0)
      out.push({ mode: "COLUMNS", text: t("database.columns"), Icon: Columns });
    if (view.dependencyColumns.length > 0)
      out.push({
        mode: "DEPENDENCY-COLUMNS",
        text: t("schema-editor.index.dependency-columns"),
        Icon: FileSymlink,
      });
    return out;
  }, [view, t]);

  return (
    <Tabs
      value={mode}
      onValueChange={(value) => setMode(value as Mode)}
      className="h-full flex flex-col overflow-hidden"
    >
      <div className="flex items-center justify-between gap-x-3 px-2 py-1 border-b border-block-border shrink-0">
        <div className="flex items-center gap-x-3 min-w-0">
          <Button
            variant="ghost"
            className="h-8 px-1 text-sm shrink-0"
            onClick={() => clearDetail()}
          >
            <ChevronLeft className="size-5" />
            <Eye className="size-4 text-control" />
            <span className="truncate">{view.name}</span>
          </Button>
          {tabs.length > 1 ? (
            <TabsList className="gap-x-3">
              {tabs.map(({ mode: m, text, Icon }) => (
                <TabsTrigger
                  key={m}
                  value={m}
                  className="flex items-center gap-x-1 pb-1.5"
                >
                  <Icon className="size-4" />
                  {text}
                </TabsTrigger>
              ))}
            </TabsList>
          ) : null}
        </div>
        {mode === "DEFINITION" ? (
          <div className="flex items-center gap-2">
            <label className="flex items-center gap-x-1 text-sm text-control cursor-pointer select-none">
              <Checkbox
                checked={format}
                onCheckedChange={(checked) => setFormat(checked)}
              />
              {t("sql-editor.format")}
            </label>
            <OpenAIButton
              size="sm"
              actions={["explain-code"]}
              statement={selectedStatement || view.definition}
            />
          </div>
        ) : (
          <PanelSearchBox value={keyword} onChange={setKeyword} />
        )}
      </div>
      <TabsPanel value="DEFINITION" className="flex-1 min-h-0 mt-0">
        <DefinitionViewer
          db={db}
          code={view.definition}
          format={format}
          onSelectContent={setSelectedStatement}
        />
      </TabsPanel>
      {view.columns.length > 0 ? (
        <TabsPanel value="COLUMNS" className="flex-1 min-h-0 mt-0">
          <ColumnsTable
            db={db}
            database={database}
            schema={schema}
            columns={view.columns}
            flavor="view"
            keyword={keyword}
          />
        </TabsPanel>
      ) : null}
      {view.dependencyColumns.length > 0 ? (
        <TabsPanel value="DEPENDENCY-COLUMNS" className="flex-1 min-h-0 mt-0">
          <DependencyColumnsTable db={db} view={view} keyword={keyword} />
        </TabsPanel>
      ) : null}
    </Tabs>
  );
}
