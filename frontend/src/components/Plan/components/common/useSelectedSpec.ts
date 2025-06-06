import type { Ref } from "vue";
import type { Plan_Spec } from "@/types/proto/v1/plan_service";
import { usePlanContext } from "../../logic";

const useSelectedSpec = () => {
  const { selectedSpec } = usePlanContext();

  return selectedSpec as Ref<Plan_Spec>;
};

export default useSelectedSpec;
