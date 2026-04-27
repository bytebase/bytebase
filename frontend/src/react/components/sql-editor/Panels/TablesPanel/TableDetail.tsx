import {
  ChevronLeft,
  Columns,
  KeyRound,
  Layers,
  Link as LinkIcon,
  Table as TableIcon,
  Zap,
} from "lucide-react";
import { useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@/react/components/ui/button";
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
  TableMetadata,
} from "@/types/proto-es/v1/database_service_pb";
import {
  extractKeyWithPosition,
  keyWithPosition,
} from "@/views/sql-editor/EditorCommon/utils";
import { CodeViewer } from "../common/CodeViewer";
import { PanelSearchBox } from "../common/PanelSearchBox";
import { ColumnsTable } from "../common/tables/ColumnsTable";
import { ForeignKeysTable } from "../common/tables/ForeignKeysTable";
import { IndexesTable } from "../common/tables/IndexesTable";
import { PartitionsTable } from "../common/tables/PartitionsTable";
import { useViewStateNav } from "../common/useViewStateNav";
import { TriggersTable } from "../TriggersPanel/TriggersTable";

type TabView =
  | "COLUMNS"
  | "INDEXES"
  | "FOREIGN-KEYS"
  | "TRIGGERS"
  | "PARTITIONS";

interface TableDetailProps {
  db: Database;
  database: DatabaseMetadata;
  schema: SchemaMetadata;
  table: TableMetadata;
}

export function TableDetail({ db, database, schema, table }: TableDetailProps) {
  const { t } = useTranslation();
  const { detail, setDetail, clearDetail } = useViewStateNav();
  const [tab, setTab] = useState<TabView>("COLUMNS");
  const [keyword, setKeyword] = useState("");

  // Sync the active tab with the detail anchor (column / index / FK /
  // partition / trigger). Mirrors the Vue watch in TableDetail.vue.
  useEffect(() => {
    if (!detail?.table) return;
    if (detail.column) {
      setTab("COLUMNS");
      return;
    }
    if (detail.index) {
      setTab("INDEXES");
      return;
    }
    if (detail.foreignKey) {
      setTab("FOREIGN-KEYS");
      return;
    }
    if (detail.partition) {
      setTab("PARTITIONS");
      return;
    }
    if (detail.trigger) {
      setTab("TRIGGERS");
      return;
    }
    setTab("COLUMNS");
  }, [
    detail?.table,
    detail?.column,
    detail?.index,
    detail?.foreignKey,
    detail?.partition,
    detail?.trigger,
  ]);

  const tabs = useMemo(() => {
    const out: { view: TabView; text: string; Icon: typeof Columns }[] = [
      { view: "COLUMNS", text: t("database.columns"), Icon: Columns },
    ];
    if (table.indexes.length > 0)
      out.push({
        view: "INDEXES",
        text: t("schema-editor.index.indexes"),
        Icon: KeyRound,
      });
    if (table.foreignKeys.length > 0)
      out.push({
        view: "FOREIGN-KEYS",
        text: t("database.foreign-keys"),
        Icon: LinkIcon,
      });
    if (table.triggers.length > 0)
      out.push({ view: "TRIGGERS", text: t("db.triggers"), Icon: Zap });
    if (table.partitions.length > 0)
      out.push({
        view: "PARTITIONS",
        text: t("schema-editor.table-partition.partitions"),
        Icon: Layers,
      });
    return out;
  }, [table, t]);

  // When a trigger row is clicked, the TRIGGERS tab swaps to a CodeViewer.
  const [triggerName, triggerPosition] = extractKeyWithPosition(
    detail?.trigger ?? ""
  );
  const trigger =
    detail?.trigger && detail.trigger !== "-1"
      ? table.triggers.find(
          (tr, i) => tr.name === triggerName && i === triggerPosition
        )
      : undefined;

  const deselectTrigger = () => {
    setDetail({ trigger: "-1" });
  };

  if (trigger) {
    return (
      <CodeViewer
        db={db}
        title={trigger.name}
        code={trigger.body}
        onBack={deselectTrigger}
        titlePrefix={
          <Button
            variant="ghost"
            className="h-8 px-1 text-sm"
            onClick={deselectTrigger}
          >
            <ChevronLeft className="size-5" />
            <Zap className="size-4 text-control" />
            <span className="truncate">{trigger.name}</span>
          </Button>
        }
      />
    );
  }

  return (
    <Tabs
      value={tab}
      onValueChange={(value) => setTab(value as TabView)}
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
            <TableIcon className="size-4 text-control" />
            <span className="truncate">{table.name}</span>
          </Button>
          {tabs.length > 1 ? (
            <TabsList className="gap-x-3">
              {tabs.map(({ view, text, Icon }) => (
                <TabsTrigger
                  key={view}
                  value={view}
                  className="flex items-center gap-x-1 pb-1.5"
                >
                  <Icon className="size-4" />
                  {text}
                </TabsTrigger>
              ))}
            </TabsList>
          ) : null}
        </div>
        <PanelSearchBox value={keyword} onChange={setKeyword} />
      </div>
      <TabsPanel value="COLUMNS" className="flex-1 min-h-0 mt-0">
        <ColumnsTable
          db={db}
          database={database}
          schema={schema}
          table={table}
          columns={table.columns}
          flavor="table"
          keyword={keyword}
        />
      </TabsPanel>
      {table.indexes.length > 0 ? (
        <TabsPanel value="INDEXES" className="flex-1 min-h-0 mt-0">
          <IndexesTable table={table} keyword={keyword} />
        </TabsPanel>
      ) : null}
      {table.foreignKeys.length > 0 ? (
        <TabsPanel value="FOREIGN-KEYS" className="flex-1 min-h-0 mt-0">
          <ForeignKeysTable db={db} table={table} keyword={keyword} />
        </TabsPanel>
      ) : null}
      {table.triggers.length > 0 ? (
        <TabsPanel value="TRIGGERS" className="flex-1 min-h-0 mt-0">
          <TriggersTable
            table={table}
            triggers={table.triggers}
            keyword={keyword}
            onSelect={({ trigger: target, position }) =>
              setDetail({
                table: table.name,
                trigger: keyWithPosition(target.name, position),
              })
            }
          />
        </TabsPanel>
      ) : null}
      {table.partitions.length > 0 ? (
        <TabsPanel value="PARTITIONS" className="flex-1 min-h-0 mt-0">
          <PartitionsTable table={table} keyword={keyword} />
        </TabsPanel>
      ) : null}
    </Tabs>
  );
}
