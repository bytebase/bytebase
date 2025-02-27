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
          :checked="isIssueCreatorChecked"
          :disabled="disabled"
          style="--n-label-padding: 0 0 0 1rem"
          @update:checked="toggleIssueRoles($event, VirtualRoleType.CREATOR)"
        >
          <div class="textlabel">
            {{ $t("role.issue-creator.self") }}
          </div>
        </NCheckbox>
        <NCheckbox
          :checked="isIssueLastApproverChecked"
          :disabled="disabled"
          style="--n-label-padding: 0 0 0 1rem"
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
        style="--n-label-padding: 0 0 0 1rem"
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
import { cloneDeep, uniq } from "lodash-es";
import { NCheckbox } from "naive-ui";
import { ref, watch } from "vue";
import { computed } from "vue";
import { VirtualRoleType } from "@/types";
import type { Policy } from "@/types/proto/v1/org_policy_service";
import { RolloutPolicy } from "@/types/proto/v1/org_policy_service";
import { RoleSelect } from "../v2";

const props = defineProps<{
  policy: Policy;
  disabled?: boolean;
}>();

const emit = defineEmits<{
  (event: "update:policy", policy: Policy): void;
}>();

const rolloutPolicy = ref(cloneDeep(props.policy.rolloutPolicy!));

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
    rolloutPolicy: rp,
  });
};
const toggleAutomaticRollout = (selected: boolean) => {
  update(RolloutPolicy.fromJSON({
    ...rolloutPolicy.value,
    automatic: selected,
  }));
};
const toggleIssueRoles = (checked: boolean, role: string) => {
  const issueRoles = rolloutPolicy.value.issueRoles;
  if (checked) {
    issueRoles.push(role);
  } else {
    const index = issueRoles.indexOf(role);
    if (index !== -1) {
      issueRoles.splice(index, 1);
    }
  }
  update(
    RolloutPolicy.fromPartial({
      ...rolloutPolicy.value,
      issueRoles: uniq(issueRoles),
    })
  );
};
const updateRoles = (roles: string[]) => {
  update(
    RolloutPolicy.fromPartial({
      ...rolloutPolicy.value,
      roles: roles,
    })
  );
};

watch(
  () => props.policy.rolloutPolicy!,
  (p) => {
    rolloutPolicy.value = cloneDeep(p);
  },
  { immediate: true, deep: true }
);
</script>
