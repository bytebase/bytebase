<template>
  <div>
    <div
      v-show="showClosedBanner"
      class="h-8 w-full text-base font-medium bg-gray-400 text-white flex justify-center items-center shrink-0"
    >
      {{ $t("common.closed") }}
    </div>
    <div
      v-show="showSuccessBanner"
      class="h-8 w-full text-base font-medium bg-success text-white flex justify-center items-center shrink-0"
    >
      {{ $t("common.done") }}
    </div>
    <div
      v-show="showReadyForRollout"
      class="h-8 w-full text-base font-medium bg-accent text-white flex justify-center items-center shrink-0"
    >
      <NButton
        text
        class="!text-white hover:opacity-80"
        :icon-placement="'right'"
        @click="handleGoToRollout"
      >
        {{ t("issue.approval.ready-for-rollout") }}
        <template #icon>
          <ArrowRightIcon class="w-4 h-4" />
        </template>
      </NButton>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { ArrowRightIcon } from "lucide-vue-next";
import { NButton } from "naive-ui";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { usePlanContext } from "@/components/Plan";
import { PROJECT_V1_ROUTE_ROLLOUT_DETAIL } from "@/router/dashboard/projectV1";
import { State } from "@/types/proto-es/v1/common_pb";
import {
  IssueStatus,
  Issue_ApprovalStatus,
  Issue_Type,
} from "@/types/proto-es/v1/issue_service_pb";
import { Task_Status, Task_Type } from "@/types/proto-es/v1/rollout_service_pb";
import {
  activeTaskInRollout,
  extractProjectResourceName,
  extractRolloutUID,
} from "@/utils";

const props = defineProps<{
  currentTab: string;
}>();

const { t } = useI18n();
const router = useRouter();
const { plan, issue, rollout } = usePlanContext();

const showClosedBanner = computed(() => {
  return (
    plan.value.state === State.DELETED ||
    (issue.value && issue.value.status === IssueStatus.CANCELED)
  );
});

const showSuccessBanner = computed(() => {
  return issue.value && issue.value.status === IssueStatus.DONE;
});

const showReadyForRollout = computed(() => {
  // Only show for OPEN issues
  if (!issue.value || issue.value.status !== IssueStatus.OPEN) return false;

  // Only show for database change issues
  if (issue.value.type !== Issue_Type.DATABASE_CHANGE) return false;

  // Check if issue is approved
  if (
    issue.value.approvalStatus !== Issue_ApprovalStatus.APPROVED &&
    issue.value.approvalStatus !== Issue_ApprovalStatus.SKIPPED
  ) {
    return false;
  }

  // Hide if on rollout tab
  if (props.currentTab === "rollout") {
    return false;
  }

  if (!rollout.value) {
    return false;
  }

  // Don't show for rollouts with database creation/export tasks
  const hasDatabaseCreateOrExportTasks = rollout.value.stages.some((stage) =>
    stage.tasks.some(
      (task) =>
        task.type === Task_Type.DATABASE_CREATE ||
        task.type === Task_Type.DATABASE_EXPORT
    )
  );
  if (hasDatabaseCreateOrExportTasks) {
    return false;
  }

  // Check if there's an active task that needs action
  const activeTask = activeTaskInRollout(rollout.value);
  return (
    activeTask.status === Task_Status.NOT_STARTED ||
    activeTask.status === Task_Status.PENDING ||
    activeTask.status === Task_Status.RUNNING ||
    activeTask.status === Task_Status.FAILED ||
    activeTask.status === Task_Status.CANCELED
  );
});

const handleGoToRollout = () => {
  if (!rollout.value) return;

  router.push({
    name: PROJECT_V1_ROUTE_ROLLOUT_DETAIL,
    params: {
      projectId: extractProjectResourceName(plan.value.name),
      rolloutId: extractRolloutUID(rollout.value.name),
    },
  });
};
</script>
