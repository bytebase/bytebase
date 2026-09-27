import { InstanceDetailView } from "@/components/instance/InstanceDetailView";
import { instanceNamePrefix } from "@/stores/modules/v1/common";

export function InstanceDetailPage({ instanceId }: { instanceId: string }) {
  return (
    <InstanceDetailView instanceName={`${instanceNamePrefix}${instanceId}`} />
  );
}
