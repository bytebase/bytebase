<template>
  <div class="flex flex-col items-start gap-y-2">
    <div class="flex flex-col gap-y-2">
      <div class="flex flex-col gap-y-2">
        <RoleSelect
          v-model:value="rolloutPolicy.roles"
          :disabled="disabled"
          multiple
          @update:value="updateRoles(rolloutPolicy.roles)"
        />
        <NCheckbox
          v-if="shouldShowIssueCreator"
          :checked="isIssueCreatorChecked"
          :disabled="disabled"
          @update:checked="toggleIssueRoles($event, VirtualRoleType.CREATOR)"
        >
          <div class="textlabel">
            {{ $t("role.issue-creator.self") }}
          </div>
        </NCheckbox>
        <NCheckbox
          v-if="shouldShowLastApprover"
          :checked="isIssueLastApproverChecked"
          :disabled="disabled"
          @update:checked="
            toggleIssueRoles($event, VirtualRoleType.LAST_APPROVER)
          "
        >
          <div class="textlabel">
            {{ $t("policy.rollout.last-issue-approver") }}
          </div>
        </NCheckbox>
      </div>
      <NCheckbox
        :checked="isAutomaticRolloutChecked"
        :disabled="disabled"
        @update:checked="toggleAutomaticRollout($event)"
      >
        <div class="flex flex-col gap-y-1">
          <div class="textlabel">
            {{ $t("policy.rollout.auto") }}
          </div>
          <div class="textinfolabel">
            {{ $t("policy.rollout.auto-info") }}
          </div>
        </div>
      </NCheckbox>
    </div>
  </div>
</template>

<script setup lang="ts">
import { create } from "@bufbuild/protobuf";
import { cloneDeep, uniq } from "lodash-es";
import { NCheckbox } from "naive-ui";
import { ref, watch, computed } from "vue";
import { VirtualRoleType } from "@/types";
import type {
  Policy,
  RolloutPolicy,
} from "@/types/proto-es/v1/org_policy_service_pb";
import { RolloutPolicySchema } from "@/types/proto-es/v1/org_policy_service_pb";
import { RoleSelect } from "../v2";

const props = defineProps<{
  policy: Policy;
  disabled?: boolean;
}>();

const emit = defineEmits<{
  (event: "update:policy", policy: Policy): void;
}>();

const rolloutPolicy = ref<RolloutPolicy>(
  cloneDeep(
    props.policy.policy?.case === "rolloutPolicy"
      ? props.policy.policy.value
      : create(RolloutPolicySchema)
  )
);

// Check if deprecated options are configured in the original policy
const originalPolicy = computed(() =>
  props.policy.policy?.case === "rolloutPolicy"
    ? props.policy.policy.value
    : undefined
);

const shouldShowIssueCreator = computed(() => {
  // Show if the original policy has this role configured
  return (
    originalPolicy.value?.issueRoles?.includes(VirtualRoleType.CREATOR) ?? false
  );
});

const shouldShowLastApprover = computed(() => {
  // Show if the original policy has this role configured
  return (
    originalPolicy.value?.issueRoles?.includes(VirtualRoleType.LAST_APPROVER) ??
    false
  );
});

const isAutomaticRolloutChecked = computed(() => {
  return rolloutPolicy.value.automatic;
});
const isIssueCreatorChecked = computed(() => {
  return rolloutPolicy.value.issueRoles.includes(VirtualRoleType.CREATOR);
});
const isIssueLastApproverChecked = computed(() => {
  return rolloutPolicy.value.issueRoles.includes(VirtualRoleType.LAST_APPROVER);
});

const update = (rp: RolloutPolicy) => {
  emit("update:policy", {
    ...props.policy,
    policy: {
      case: "rolloutPolicy",
      value: rp,
    },
  });
};
const toggleAutomaticRollout = (selected: boolean) => {
  update(
    create(RolloutPolicySchema, {
      ...rolloutPolicy.value,
      automatic: selected,
    })
  );
};
const toggleIssueRoles = (checked: boolean, role: string) => {
  const issueRoles = [...rolloutPolicy.value.issueRoles];
  if (checked) {
    issueRoles.push(role);
  } else {
    const index = issueRoles.indexOf(role);
    if (index !== -1) {
      issueRoles.splice(index, 1);
    }
  }
  update(
    create(RolloutPolicySchema, {
      ...rolloutPolicy.value,
      issueRoles: uniq(issueRoles),
    })
  );
};
const updateRoles = (roles: string[]) => {
  update(
    create(RolloutPolicySchema, {
      ...rolloutPolicy.value,
      roles: roles,
    })
  );
};

watch(
  () =>
    props.policy.policy?.case === "rolloutPolicy"
      ? props.policy.policy.value
      : undefined,
  (p) => {
    if (p) {
      rolloutPolicy.value = cloneDeep(p);
    }
  },
  { immediate: true, deep: true }
);
</script>
