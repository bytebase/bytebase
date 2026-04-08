import { useEffect, useRef } from "react";
import { createApp, h } from "vue";
import DatabaseOverviewInfo from "@/components/Database/DatabaseOverviewInfo.vue";
import i18n from "@/plugins/i18n";
import NaiveUI from "@/plugins/naive-ui";
import { router } from "@/router";
import { pinia } from "@/store";
import type { Database } from "@/types/proto-es/v1/database_service_pb";

export function DatabaseOverviewInfoBridge({
  database,
  className,
}: {
  database: Database;
  className?: string;
}) {
  const containerRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!containerRef.current) {
      return;
    }

    const app = createApp({
      render() {
        return h(DatabaseOverviewInfo as never, {
          database,
        });
      },
    });
    app.use(router).use(pinia).use(i18n).use(NaiveUI);
    app.mount(containerRef.current);

    return () => {
      app.unmount();
    };
  }, [database]);

  return <div className={className} ref={containerRef} />;
}
