import { useEffect, useRef } from "react";
import { h } from "vue";
import SensitiveColumnTable from "@/components/SensitiveData/components/SensitiveColumnTable.vue";
import type { MaskData } from "@/components/SensitiveData/types";
import { createLegacyVueApp } from "@/react/legacy/mountLegacyVueApp";
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

    const app = createLegacyVueApp({
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
