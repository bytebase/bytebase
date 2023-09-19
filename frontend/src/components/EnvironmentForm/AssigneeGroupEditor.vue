<template>
  <div class="my-4 space-y-4">
    <div class="flex space-x-4">
      <input
        :checked="assigneeGroup === ApprovalGroup.APPROVAL_GROUP_DBA"
        name="WORKSPACE_OWNER_OR_DBA"
        tabindex="-1"
        type="radio"
        class="text-accent disabled:text-accent-disabled focus:ring-accent"
        :value="ApprovalGroup.APPROVAL_GROUP_DBA"
        :disabled="disabled"
        @input="
          $emit(
            'update',
            getAssigneeGroupListByValue(ApprovalGroup.APPROVAL_GROUP_DBA)
          )
        "
      />
      <div class="-mt-0.5">
        <div class="textlabel">
          {{ $t("policy.rollout.assignee-group.workspace-owner-or-dba") }}
        </div>
      </div>
    </div>
    <div class="flex space-x-4">
      <input
        :checked="assigneeGroup === ApprovalGroup.APPROVAL_GROUP_PROJECT_OWNER"
        name="PROJECT_OWNER"
        tabindex="-1"
        type="radio"
        class="text-accent disabled:text-accent-disabled focus:ring-accent"
        :value="ApprovalGroup.APPROVAL_GROUP_PROJECT_OWNER"
        :disabled="disabled"
        @input="
          $emit(
            'update',
            getAssigneeGroupListByValue(
              ApprovalGroup.APPROVAL_GROUP_PROJECT_OWNER
            )
          )
        "
      />
      <div class="-mt-0.5">
        <div class="textlabel">
          {{ $t("policy.rollout.assignee-group.project-owner") }}
        </div>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { computed, watch } from "vue";
import { DeploymentType } from "@/types/proto/v1/deployment";
import {
  Policy,
  ApprovalGroup,
  ApprovalStrategy,
  DeploymentApprovalStrategy,
} from "@/types/proto/v1/org_policy_service";

const props = defineProps<{
  disabled: boolean;
  policy: Policy;
}>();

const emit = defineEmits<{
  (event: "update", assigneeGroupList: DeploymentApprovalStrategy[]): void;
}>();

const payload = computed(() => {
  return props.policy.deploymentApprovalPolicy!;
});

const assigneeGroup = computed(() => {
  if (payload.value.defaultStrategy === ApprovalStrategy.AUTOMATIC) {
    return ApprovalGroup.ASSIGNEE_GROUP_UNSPECIFIED;
  }

  if (payload.value.deploymentApprovalStrategies.length == 0) {
    return ApprovalGroup.APPROVAL_GROUP_DBA;
  }

  return payload.value.deploymentApprovalStrategies[0].approvalGroup;
});

const getAssigneeGroupListByValue = (
  approvalGroup: ApprovalGroup
): DeploymentApprovalStrategy[] => {
  const issueTypeList: Array<DeploymentType> = [
    DeploymentType.DATABASE_DDL,
    DeploymentType.DATABASE_DML,
    DeploymentType.DATABASE_DDL_GHOST,
  ];
  return issueTypeList.map((deploymentType) => ({
    deploymentType,
    approvalGroup,
    approvalStrategy: payload.value.defaultStrategy,
  }));
};

watch(
  () => payload.value?.defaultStrategy,
  (value) => {
    if (value === ApprovalStrategy.AUTOMATIC) {
      // Empty the array since it's meaningless when MANUAL_APPROVAL_NEVER
      emit("update", []);
    } else if (value === ApprovalStrategy.MANUAL) {
      // Sync the local state (DBA_OR_OWNER / PROJECT_OWNER) to the payload
      // when switching from "skip manual approval" -> "require manual approval"
      emit("update", getAssigneeGroupListByValue(assigneeGroup.value));
    }
  },
  { immediate: true }
);
</script>
