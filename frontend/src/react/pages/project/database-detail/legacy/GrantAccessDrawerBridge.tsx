import { useEffect, useRef } from "react";
import { h } from "vue";
import GrantAccessDrawer from "@/components/SensitiveData/GrantAccessDrawer.vue";
import type { SensitiveColumn } from "@/components/SensitiveData/types";
import { createLegacyVueApp } from "@/react/legacy/mountLegacyVueApp";

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

    const app = createLegacyVueApp({
      render() {
        return h(GrantAccessDrawer as never, {
          columnList,
          projectName,
          onDismiss,
        });
      },
    });
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
