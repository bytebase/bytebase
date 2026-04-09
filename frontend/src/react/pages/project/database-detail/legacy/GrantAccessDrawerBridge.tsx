import { useEffect, useRef } from "react";
import { createApp, h } from "vue";
import GrantAccessDrawer from "@/components/SensitiveData/GrantAccessDrawer.vue";
import type { SensitiveColumn } from "@/components/SensitiveData/types";
import i18n from "@/plugins/i18n";
import NaiveUI from "@/plugins/naive-ui";
import { router } from "@/router";
import { pinia } from "@/store";

export interface GrantAccessDrawerBridgeProps {
  open: boolean;
  columnList: SensitiveColumn[];
  projectName: string;
  onDismiss: () => void;
}

export function GrantAccessDrawerBridge({
  open,
  columnList,
  projectName,
  onDismiss,
}: GrantAccessDrawerBridgeProps) {
  const containerRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!open || !containerRef.current) {
      return;
    }

    const app = createApp({
      render() {
        return h(GrantAccessDrawer as never, {
          columnList,
          projectName,
          onDismiss,
        });
      },
    });

    app.use(router).use(pinia).use(i18n).use(NaiveUI);
    app.mount(containerRef.current);

    return () => {
      app.unmount();
    };
  }, [columnList, onDismiss, open, projectName]);

  if (!open) {
    return null;
  }

  return <div ref={containerRef} />;
}
