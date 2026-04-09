import { useEffect, useRef } from "react";
import { h } from "vue";
import DatabaseOverviewInfo from "@/components/Database/DatabaseOverviewInfo.vue";
import { createLegacyVueApp } from "@/react/legacy/mountLegacyVueApp";
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

    const app = createLegacyVueApp({
      render() {
        return h(DatabaseOverviewInfo as never, {
          database,
        });
      },
    });
    app.mount(containerRef.current);

    return () => {
      app.unmount();
    };
  }, [database]);

  return <div className={className} ref={containerRef} />;
}
