import { NConfigProvider } from "naive-ui";
import { useEffect, useRef } from "react";
import { createApp, h } from "vue";
import { themeOverrides } from "@/../naive-ui.config";
import DatabaseSensitiveDataPanel from "@/components/Database/DatabaseSensitiveDataPanel.vue";
import OverlayStackManager from "@/components/misc/OverlayStackManager.vue";
import i18n from "@/plugins/i18n";
import NaiveUI from "@/plugins/naive-ui";
import { router } from "@/router";
import { pinia } from "@/store";
import type { Database } from "@/types/proto-es/v1/database_service_pb";

export function DatabaseCatalogPanel({ database }: { database: Database }) {
  const containerRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!containerRef.current) {
      return;
    }

    const app = createApp({
      render() {
        return h(
          NConfigProvider as never,
          { themeOverrides: themeOverrides.value },
          {
            default: () =>
              h(OverlayStackManager as never, null, {
                default: () =>
                  h(DatabaseSensitiveDataPanel as never, {
                    database,
                  }),
              }),
          }
        );
      },
    });
    app.use(router).use(pinia).use(i18n).use(NaiveUI);
    app.mount(containerRef.current);

    return () => {
      app.unmount();
    };
  }, [database.name, database.project, database.environment]);

  return <div ref={containerRef} />;
}
