import { useEffect, useRef } from "react";
import { createApp, h } from "vue";
import InstanceAssignment from "@/components/InstanceAssignment.vue";
import i18n from "@/plugins/i18n";
import NaiveUI from "@/plugins/naive-ui";
import { router } from "@/router";
import { pinia } from "@/store";

export function InstanceAssignmentBridge({
  open,
  selectedInstanceList,
  onDismiss,
}: {
  open: boolean;
  selectedInstanceList?: string[];
  onDismiss: () => void;
}) {
  const containerRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!open || !containerRef.current) {
      return;
    }

    const app = createApp({
      render() {
        return h(InstanceAssignment as never, {
          show: open,
          selectedInstanceList,
          onDismiss,
        });
      },
    });
    app.use(router).use(pinia).use(i18n).use(NaiveUI);
    app.mount(containerRef.current);

    return () => {
      app.unmount();
    };
  }, [onDismiss, open, selectedInstanceList]);

  if (!open) {
    return null;
  }

  return <div ref={containerRef} />;
}
