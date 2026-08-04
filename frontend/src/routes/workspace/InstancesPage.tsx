import { useCallback } from "react";
import { router } from "@/app/router";
import { INSTANCE_ROUTE_CREATE } from "@/app/router/handles";
import { markListScrollRestorationEntry } from "@/app/router/NavigationScrollRestoration";
import { InstanceDashboard } from "@/components/instance/InstanceDashboard";
import { WorkspacePageLayout } from "@/components/WorkspacePageLayout";
import type { Instance } from "@/types/proto-es/v1/instance_service_pb";

export function InstancesPage() {
  const handleOpen = useCallback(
    (instance: Instance, event: React.MouseEvent) => {
      const url = `/${instance.name}`;
      if (event.ctrlKey || event.metaKey) {
        window.open(url, "_blank");
        return;
      }
      markListScrollRestorationEntry();
      router.push(url);
    },
    []
  );

  return (
    <WorkspacePageLayout padding="flush">
      <InstanceDashboard
        layout="workspace"
        onCreate={() => router.push({ name: INSTANCE_ROUTE_CREATE })}
        onOpen={handleOpen}
      />
    </WorkspacePageLayout>
  );
}
