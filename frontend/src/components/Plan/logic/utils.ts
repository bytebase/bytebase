import { useProjectV1Store } from "@/store";
import type { ComposedProject } from "@/types";
import type { ComposedPlan } from "@/types/v1/issue/plan";

export const projectOfPlan = (plan: ComposedPlan): ComposedProject =>
  useProjectV1Store().getProjectByName(plan.project);
