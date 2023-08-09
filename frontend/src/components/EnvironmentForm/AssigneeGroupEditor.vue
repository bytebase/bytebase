<template>
  <div
    v-if="payload.defaultStrategy === ApprovalStrategy.MANUAL"
    class="my-4 space-y-4"
  >
    <div class="flex space-x-4">
      <input
        v-model="state.assigneeGroup"
        name="WORKSPACE_OWNER_OR_DBA"
        tabindex="-1"
        type="radio"
        class="text-accent disabled:text-accent-disabled focus:ring-accent"
        :value="ApprovalGroup.APPROVAL_GROUP_DBA"
        :disabled="!allowEdit"
      />
      <div class="-mt-0.5">
        <div class="textlabel">
          {{ $t("policy.rollout.assignee-group.workspace-owner-or-dba") }}
        </div>
      </div>
    </div>
    <div class="flex space-x-4">
      <input
        v-model="state.assigneeGroup"
        name="PROJECT_OWNER"
        tabindex="-1"
        type="radio"
        class="text-accent disabled:text-accent-disabled focus:ring-accent"
        :value="ApprovalGroup.APPROVAL_GROUP_PROJECT_OWNER"
        :disabled="!allowEdit"
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
import { computed, PropType, reactive, watch } from "vue";
import { DeploymentType } from "@/types/proto/v1/deployment";
import {
  Policy,
  ApprovalGroup,
  ApprovalStrategy,
  DeploymentApprovalStrategy,
} from "@/types/proto/v1/org_policy_service";

type LocalState = {
  assigneeGroup: ApprovalGroup;
};

const props = defineProps({
  allowEdit: {
    type: Boolean,
    required: true,
  },
  policy: {
    type: Object as PropType<Policy>,
    required: true,
  },
});

const emit = defineEmits<{
  (event: "update", assigneeGroupList: DeploymentApprovalStrategy[]): void;
}>();

const payload = computed(() => {
  return props.policy.deploymentApprovalPolicy!;
});

const getAssigneeGroup = (): ApprovalGroup => {
  if (payload.value.defaultStrategy === ApprovalStrategy.AUTOMATIC) {
    return ApprovalGroup.APPROVAL_GROUP_DBA;
  }

  if (payload.value.deploymentApprovalStrategies.length == 0) {
    return ApprovalGroup.APPROVAL_GROUP_DBA;
  }

  return payload.value.deploymentApprovalStrategies[0].approvalGroup;
};

const state = reactive<LocalState>({
  assigneeGroup: getAssigneeGroup(),
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

// Editing different AssigneeGroup for each issueType is not supported by now.
// So even if assigneeGroupList is an array, we apply the same logic on its
// items one-by-one when setting value.
watch(
  () => state.assigneeGroup,
  (value) => {
    emit("update", getAssigneeGroupListByValue(value));
  }
);

watch(
  () => payload.value?.defaultStrategy,
  (value) => {
    if (value === ApprovalStrategy.AUTOMATIC) {
      // Empty the array since it's meaningless when MANUAL_APPROVAL_NEVER
      emit("update", []);
    } else if (value === ApprovalStrategy.MANUAL) {
      // Sync the local state (DBA_OR_OWNER / PROJECT_OWNER) to the payload
      // when switching from "skip manual approval" -> "require manual approval"
      emit("update", getAssigneeGroupListByValue(state.assigneeGroup));
    }
  },
  { immediate: true }
);

watch(
  () => props.policy,
  () => {
    state.assigneeGroup = getAssigneeGroup();
  }
);
</script>
