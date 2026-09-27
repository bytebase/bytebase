import type { Plan_Spec } from "@/types/proto-es/v1/plan_service_pb";

export const getSelectedSpec = ({
  selectedSpecId,
  specs,
}: {
  selectedSpecId: string;
  specs: Plan_Spec[];
}): Plan_Spec | undefined => {
  return specs.find((spec) => spec.id === selectedSpecId) ?? specs[0];
};

// A plan is GitOps-style when any spec references a release artifact instead
// of an inline sheet. These plans bypass the review/issue workflow.
export const isReleaseBackedPlan = (specs: Plan_Spec[]): boolean => {
  return specs.some(
    (spec) =>
      spec.config?.case === "changeDatabaseConfig" &&
      !!spec.config.value.release
  );
};
