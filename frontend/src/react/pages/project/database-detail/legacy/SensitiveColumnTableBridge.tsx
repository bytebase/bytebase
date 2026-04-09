import { useEffect, useRef } from "react";
import { createApp, h } from "vue";
import SensitiveColumnTable from "@/components/SensitiveData/components/SensitiveColumnTable.vue";
import type { MaskData } from "@/components/SensitiveData/types";
import i18n from "@/plugins/i18n";
import NaiveUI from "@/plugins/naive-ui";
import { router } from "@/router";
import { pinia } from "@/store";
import type { Database } from "@/types/proto-es/v1/database_service_pb";

export interface SensitiveColumnTableBridgeProps {
  database: Database;
  rowClickable: boolean;
  rowSelectable: boolean;
  showOperation: boolean;
  columnList: MaskData[];
  checkedColumnList: MaskData[];
  onCheckedColumnListChange: (list: MaskData[]) => void;
  onDelete: (item: MaskData) => void;
}

export function SensitiveColumnTableBridge({
  database,
  rowClickable,
  rowSelectable,
  showOperation,
  columnList,
  checkedColumnList,
  onCheckedColumnListChange,
  onDelete,
}: SensitiveColumnTableBridgeProps) {
  const containerRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!containerRef.current) {
      return;
    }

    const app = createApp({
      render() {
        return h(SensitiveColumnTable as never, {
          database,
          rowClickable,
          rowSelectable,
          showOperation,
          columnList,
          checkedColumnList,
          "onUpdate:checkedColumnList": onCheckedColumnListChange,
          onDelete,
        });
      },
    });

    app.use(router).use(pinia).use(i18n).use(NaiveUI);
    app.mount(containerRef.current);

    return () => {
      app.unmount();
    };
  }, [
    checkedColumnList,
    columnList,
    database,
    onCheckedColumnListChange,
    onDelete,
    rowClickable,
    rowSelectable,
    showOperation,
  ]);

  return <div ref={containerRef} />;
}
