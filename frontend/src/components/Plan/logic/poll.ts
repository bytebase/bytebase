import { create } from "@bufbuild/protobuf";
import { computed, watch } from "vue";
import { useProgressivePoll } from "@/composables/useProgressivePoll";
import {
  issueServiceClientConnect,
  planServiceClientConnect,
  rolloutServiceClientConnect,
} from "@/grpcweb";
import { useCurrentProjectV1 } from "@/store";
import { useIssueCommentStore } from "@/store";
import { usePlanStore } from "@/store/modules/v1/plan";
import { GetIssueRequestSchema } from "@/types/proto-es/v1/issue_service_pb";
import { ListPlanCheckRunsRequestSchema } from "@/types/proto-es/v1/plan_service_pb";
import { GetRolloutRequestSchema } from "@/types/proto-es/v1/rollout_service_pb";
import { ListIssueCommentsRequest } from "@/types/proto/v1/issue_service";
import { hasProjectPermissionV2 } from "@/utils";
import { convertNewIssueToOld } from "@/utils/v1/issue-conversions";
import { convertNewPlanCheckRunToOld } from "@/utils/v1/plan-conversions";
import { convertNewRolloutToOld } from "@/utils/v1/rollout-conversions";
import { usePlanContext } from "./context";

export const usePollPlan = () => {
  const { project } = useCurrentProjectV1();
  const { isCreating, plan, planCheckRunList, issue, rollout, events } =
    usePlanContext();
  const planStore = usePlanStore();
  const issueCommentStore = useIssueCommentStore();

  const shouldPollPlan = computed(() => {
    return !isCreating.value;
  });

  const refreshPlan = async () => {
    if (!shouldPollPlan.value) return;

    const updatedPlan = await planStore.fetchPlanByName(plan.value.name);
    plan.value = updatedPlan;

    if (
      !isCreating.value &&
      hasProjectPermissionV2(project.value, "bb.planCheckRuns.list")
    ) {
      const request = create(ListPlanCheckRunsRequestSchema, {
        parent: updatedPlan.name,
        latestOnly: true,
      });
      const response =
        await planServiceClientConnect.listPlanCheckRuns(request);
      planCheckRunList.value = response.planCheckRuns.map(
        convertNewPlanCheckRunToOld
      );
    }

    // Refresh rollout if it exists
    if (
      updatedPlan.rollout &&
      rollout &&
      hasProjectPermissionV2(project.value, "bb.rollouts.get")
    ) {
      try {
        const rolloutRequest = create(GetRolloutRequestSchema, {
          name: updatedPlan.rollout,
        });
        const newRollout =
          await rolloutServiceClientConnect.getRollout(rolloutRequest);
        rollout.value = convertNewRolloutToOld(newRollout);
      } catch {
        // Rollout might not exist or we don't have permission
      }
    }
  };

  const refreshIssue = async () => {
    if (!issue?.value || isCreating.value) return;

    const request = create(GetIssueRequestSchema, {
      name: issue.value.name,
    });
    const newIssue = await issueServiceClientConnect.getIssue(request);
    const updatedIssue = convertNewIssueToOld(newIssue);

    if (issue.value !== updatedIssue) {
      issue.value = updatedIssue;
    }
  };

  const refreshIssueComments = async () => {
    if (!issue?.value) return;

    await issueCommentStore.listIssueComments(
      ListIssueCommentsRequest.fromPartial({
        parent: issue.value.name,
        pageSize: 100,
      })
    );
  };

  // TODO: split this into a atomic resource poller.
  const poller = useProgressivePoll(
    () => {
      return Promise.all([
        refreshPlan(),
        refreshIssue(),
        refreshIssueComments(),
      ]);
    },
    {
      interval: {
        min: 500,
        max: 10000,
        growth: 2,
        jitter: 500,
      },
    }
  );

  // Register event handlers
  events.on("status-changed", async ({ eager }) => {
    if (eager) {
      await Promise.all([
        refreshPlan(),
        refreshIssue(),
        refreshIssueComments(),
      ]);
      poller.restart();
    }
  });

  events.on("perform-issue-review-action", async () => {
    // After review action, refresh both issue and comments
    await Promise.all([refreshIssue(), refreshIssueComments()]);
    events.emit("status-changed", { eager: true });
  });

  events.on("perform-issue-status-action", async () => {
    // After status action, refresh issue
    await refreshIssue();
    events.emit("status-changed", { eager: true });
  });

  watch(
    shouldPollPlan,
    () => {
      if (shouldPollPlan.value) {
        poller.start();
      } else {
        poller.stop();
      }
    },
    {
      immediate: true,
    }
  );
};
