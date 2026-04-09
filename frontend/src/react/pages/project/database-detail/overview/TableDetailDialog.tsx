import { useEffect, useRef } from "react";
import { createApp, h, reactive } from "vue";
import { updateTableCatalog } from "@/components/ColumnDataTable/utils";
import OverlayStackManager from "@/components/misc/OverlayStackManager.vue";
import TableDetailDrawer from "@/components/TableDetailDrawer.vue";
import i18n from "@/plugins/i18n";
import NaiveUI from "@/plugins/naive-ui";
import { router } from "@/router";
import { pinia } from "@/store";
import type { DataClassificationSetting_DataClassificationConfig } from "@/types/proto-es/v1/setting_service_pb";

export interface VueTableDetailDrawerProps {
  show: boolean;
  databaseName: string;
  schemaName: string;
  tableName: string;
  classificationConfig?: DataClassificationSetting_DataClassificationConfig;
  onDismiss: () => void;
}

export function VueTableDetailDrawerMount({
  show,
  databaseName,
  schemaName,
  tableName,
  classificationConfig,
  onDismiss,
}: VueTableDetailDrawerProps) {
  const containerRef = useRef<HTMLDivElement>(null);
  const bridgeStateRef = useRef(
    reactive({
      show,
      databaseName,
      schemaName,
      tableName,
      classificationConfig,
      onDismiss,
    })
  );

  useEffect(() => {
    const state = bridgeStateRef.current;
    state.show = show;
    state.databaseName = databaseName;
    state.schemaName = schemaName;
    state.tableName = tableName;
    state.classificationConfig = classificationConfig;
    state.onDismiss = onDismiss;
  }, [
    show,
    databaseName,
    schemaName,
    tableName,
    classificationConfig,
    onDismiss,
  ]);

  useEffect(() => {
    if (!containerRef.current) {
      return;
    }

    const bridgeState = bridgeStateRef.current;

    const app = createApp({
      render() {
        return h(OverlayStackManager as never, null, {
          default: () =>
            h(TableDetailDrawer as never, {
              show: bridgeState.show,
              databaseName: bridgeState.databaseName,
              schemaName: bridgeState.schemaName,
              tableName: bridgeState.tableName,
              classificationConfig: bridgeState.classificationConfig,
              onDismiss: bridgeState.onDismiss,
              onApplyClassification: (table: string, id: string) => {
                void updateTableCatalog({
                  database: bridgeState.databaseName,
                  schema: bridgeState.schemaName,
                  table,
                  tableCatalog: { classification: id },
                });
              },
            }),
        });
      },
    });
    app.use(router).use(pinia).use(i18n).use(NaiveUI);
    app.mount(containerRef.current);

    return () => {
      app.unmount();
    };
  }, [databaseName]);

  return <div ref={containerRef} />;
}
