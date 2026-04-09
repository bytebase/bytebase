import { computed } from "vue";
import { useRoute } from "vue-router";
import { PROJECT_V1_ROUTE_ISSUE_DETAIL } from "@/router/dashboard/projectV1";
import { Issue_ApprovalStatus } from "@/types/proto-es/v1/issue_service_pb";
import { usePlanContext } from "./context";

/**
 * Composable that determines whether the "Plan" link should be shown on the
 * Issue page for database change plans.
 */
export const usePlanLink = () => {
  const route = useRoute();
  const { plan, issue } = usePlanContext();

  const isDatabaseChangePlan = computed(() => {
    return (
      plan.value.specs.length > 0 &&
      plan.value.specs.every(
        (spec) => spec.config.case === "changeDatabaseConfig"
      )
    );
  });

  const issueApproved = computed(() => {
    const issueValue = issue.value;
    if (!issueValue) return false;

    const hasNoApprovalRequired =
      (issueValue.approvalTemplate?.flow?.roles ?? []).length === 0;

    return (
      issueValue.approvalStatus === Issue_ApprovalStatus.APPROVED ||
      issueValue.approvalStatus === Issue_ApprovalStatus.SKIPPED ||
      hasNoApprovalRequired
    );
  });

  const shouldShow = computed(() => {
    if (route.name !== PROJECT_V1_ROUTE_ISSUE_DETAIL) return false;
    if (!isDatabaseChangePlan.value) return false;
    return plan.value.hasRollout || issueApproved.value;
  });

  return {
    shouldShow,
  };
};
