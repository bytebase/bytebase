<template>
  <div v-if="payload.value === 'MANUAL_APPROVAL_ALWAYS'" class="my-4 space-y-4">
    <div class="flex space-x-4">
      <input
        v-model="state.assigneeGroup"
        name="WORKSPACE_OWNER_OR_DBA"
        tabindex="-1"
        type="radio"
        class="text-accent disabled:text-accent-disabled focus:ring-accent"
        value="WORKSPACE_OWNER_OR_DBA"
        :disabled="!allowEdit"
      />
      <div class="-mt-0.5">
        <div class="textlabel">
          {{ $t("policy.approval.assignee-group.workspace-owner-or-dba") }}
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
        value="PROJECT_OWNER"
        :disabled="!allowEdit"
      />
      <div class="-mt-0.5">
        <div class="textlabel">
          {{ $t("policy.approval.assignee-group.project-owner") }}
        </div>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { computed, PropType, reactive, watch } from "vue";
import {
  AssigneeGroup,
  AssigneeGroupValue,
  DefaultAssigneeGroup,
  PipelineApprovalPolicyPayload,
  Policy,
} from "@/types";

type LocalState = {
  assigneeGroup: AssigneeGroupValue;
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
  (event: "update", assigneeGroupList: AssigneeGroup[]): void;
}>();

const payload = computed(() => {
  return props.policy.payload as PipelineApprovalPolicyPayload;
});

const getAssigneeGroup = (): AssigneeGroupValue => {
  const { value, assigneeGroupList } = payload.value;
  if (value === "MANUAL_APPROVAL_NEVER") {
    return "WORKSPACE_OWNER_OR_DBA";
  }

  if (assigneeGroupList.length === 0) {
    return DefaultAssigneeGroup;
  }
  return assigneeGroupList[0].value;
};

const state = reactive<LocalState>({
  assigneeGroup: getAssigneeGroup(),
});

const getAssigneeGroupListByValue = (value: AssigneeGroupValue) => {
  const issueTypeList: Array<AssigneeGroup["issueType"]> = [
    "bb.issue.database.schema.update",
    "bb.issue.database.data.update",
    "bb.issue.database.schema.update.ghost",
  ];
  const assigneeGroupList = issueTypeList.map<AssigneeGroup>((issueType) => ({
    issueType,
    value,
  }));
  return assigneeGroupList;
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
  () => payload.value.value,
  (value) => {
    if (value === "MANUAL_APPROVAL_NEVER") {
      // Empty the array since it's meaningless when MANUAL_APPROVAL_NEVER
      emit("update", []);
    } else if (value === "MANUAL_APPROVAL_ALWAYS") {
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
