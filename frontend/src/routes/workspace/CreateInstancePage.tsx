import { useCallback } from "react";
import { router } from "@/app/router";
import {
  INSTANCE_ROUTE_DASHBOARD,
  INSTANCE_ROUTE_DETAIL,
} from "@/app/router/handles";
import { CreateInstanceView } from "@/components/instance/CreateInstanceView";
import {
  PREPARE_DATABASE_PRODUCT_INTRO,
  PREPARE_DATABASE_TRANSFER_TIP,
  PRODUCT_INTRO_QUERY_KEY,
  PRODUCT_INTRO_TIP_QUERY_KEY,
} from "@/lib/productIntro";
import type { Instance } from "@/types/proto-es/v1/instance_service_pb";
import { extractInstanceResourceName } from "@/utils";

export function CreateInstancePage() {
  const onDismiss = useCallback(() => {
    router.push({ name: INSTANCE_ROUTE_DASHBOARD });
  }, []);

  const onCreated = useCallback((instance: Instance) => {
    const instanceId = extractInstanceResourceName(instance.name);
    router.push({
      name: INSTANCE_ROUTE_DETAIL,
      params: { instanceId },
      query: {
        syncingInstance: instanceId,
        [PRODUCT_INTRO_QUERY_KEY]: PREPARE_DATABASE_PRODUCT_INTRO,
        [PRODUCT_INTRO_TIP_QUERY_KEY]: PREPARE_DATABASE_TRANSFER_TIP,
      },
      hash: "databases",
    });
  }, []);

  return <CreateInstanceView onDismiss={onDismiss} onCreated={onCreated} />;
}
