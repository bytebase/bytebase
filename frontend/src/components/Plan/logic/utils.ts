import { useProjectV1Store } from "@/store";
import type { ComposedProject } from "@/types";
import type { Plan } from "@/types/proto/v1/plan_service";
import { extractProjectResourceName } from "@/utils";

export const projectOfPlan = (plan: Plan): ComposedProject => {
  const project = `projects/${extractProjectResourceName(plan.name)}`;
  return useProjectV1Store().getProjectByName(project);
};
