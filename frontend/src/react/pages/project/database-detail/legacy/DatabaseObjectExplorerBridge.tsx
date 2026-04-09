import { useEffect, useRef } from "react";
import { createApp, h } from "vue";
import i18n from "@/plugins/i18n";
import NaiveUI from "@/plugins/naive-ui";
import { router } from "@/router";
import { pinia } from "@/store";
import type { Database } from "@/types/proto-es/v1/database_service_pb";
import DatabaseObjectExplorer from "./DatabaseObjectExplorer.vue";

export interface DatabaseObjectExplorerBridgeProps {
  database: Database;
  loading: boolean;
  selectedSchemaName: string;
  tableSearchKeyword: string;
  externalTableSearchKeyword: string;
  onSelectedSchemaNameChange: (value: string) => void;
  onTableSearchKeywordChange: (value: string) => void;
  onExternalTableSearchKeywordChange: (value: string) => void;
}

export function DatabaseObjectExplorerBridge({
  database,
  loading,
  selectedSchemaName,
  tableSearchKeyword,
  externalTableSearchKeyword,
  onSelectedSchemaNameChange,
  onTableSearchKeywordChange,
  onExternalTableSearchKeywordChange,
}: DatabaseObjectExplorerBridgeProps) {
  const containerRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!containerRef.current) {
      return;
    }

    const app = createApp({
      render() {
        return h(DatabaseObjectExplorer as never, {
          database,
          loading,
          selectedSchemaName,
          tableSearchKeyword,
          externalTableSearchKeyword,
          "onUpdate:selected-schema-name": onSelectedSchemaNameChange,
          "onUpdate:table-search-keyword": onTableSearchKeywordChange,
          "onUpdate:external-table-search-keyword":
            onExternalTableSearchKeywordChange,
        });
      },
    });
    app.use(router).use(pinia).use(i18n).use(NaiveUI);
    app.mount(containerRef.current);

    return () => {
      app.unmount();
    };
  }, [
    database,
    externalTableSearchKeyword,
    loading,
    onExternalTableSearchKeywordChange,
    onSelectedSchemaNameChange,
    onTableSearchKeywordChange,
    selectedSchemaName,
    tableSearchKeyword,
  ]);

  return <div ref={containerRef} />;
}
