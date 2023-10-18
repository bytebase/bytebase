<template>
  <div class="rollout-policy-config flex flex-col gap-y-2">
    <div class="flex flex-col gap-y-2">
      <NRadio
        :checked="rolloutPolicy.automatic"
        @update:checked="selectAutomaticRollout"
      >
        <div class="flex flex-col gap-y-2">
          <div class="textlabel">
            {{ $t("policy.rollout.auto") }}
          </div>
          <div class="textinfolabel">
            {{ $t("policy.rollout.auto-info") }}
          </div>
        </div>
      </NRadio>
    </div>
    <div class="flex flex-col gap-y-2">
      <NCheckbox
        :checked="manualRolloutByFixedRolesCheckStatus.checked"
        :indeterminate="manualRolloutByFixedRolesCheckStatus.indeterminate"
        @update:checked="toggleAllFixedRoles"
      >
        <div class="flex flex-col gap-y-2">
          <div class="textlabel">
            {{ $t("policy.rollout.manual-by-fixed-roles") }}
          </div>
        </div>
      </NCheckbox>
      <div class="flex flex-col gap-y-2 pl-8">
        <NCheckbox
          :checked="isDBAOrWorkspaceOwnerChecked"
          @update:checked="toggleDBAOrWorkspaceOwner"
        >
          <div class="textlabel">
            {{ $t("policy.rollout.manual-by-dba-or-workspace-owner") }}
          </div>
        </NCheckbox>
        <NCheckbox
          :checked="isProjectOwnerChecked"
          @update:checked="toggleProjectRoles($event, [PresetRoleType.OWNER])"
        >
          <div class="textlabel">
            {{ $t("policy.rollout.manual-by-project-owner") }}
          </div>
        </NCheckbox>
        <NCheckbox
          :checked="isProjectReleaserChecked"
          @update:checked="
            toggleProjectRoles($event, [PresetRoleType.RELEASER])
          "
        >
          <div class="textlabel">
            {{ $t("policy.rollout.manual-by-project-releaser") }}
          </div>
        </NCheckbox>
        <NCheckbox
          :checked="isIssueCreatorChecked"
          @update:checked="toggleIssueRoles($event, [VirtualRoleType.CREATOR])"
        >
          <div class="textlabel">
            {{ $t("policy.rollout.manual-by-issue-creator") }}
          </div>
        </NCheckbox>
      </div>
    </div>
    <div class="flex flex-col gap-y-2">
      <NCheckbox
        :checked="isIssueLastApproverChecked"
        @update:checked="
          toggleIssueRoles($event, [VirtualRoleType.LAST_APPROVER])
        "
      >
        <div class="flex flex-col gap-y-2">
          <div class="textlabel">
            {{ $t("policy.rollout.manual-by-last-approver") }}
          </div>
        </div>
      </NCheckbox>
    </div>
  </div>
</template>

<script setup lang="ts">
import { cloneDeep } from "lodash-es";
import { NCheckbox, NRadio } from "naive-ui";
import { ref, watch } from "vue";
import { computed } from "vue";
import { PresetRoleType, VirtualRoleType } from "@/types";
import { Policy, RolloutPolicy } from "@/types/proto/v1/org_policy_service";

const props = defineProps<{
  policy: Policy;
}>();
const emit = defineEmits<{
  (event: "update:policy", policy: Policy): void;
}>();

const rolloutPolicy = ref(cloneDeep(props.policy.rolloutPolicy!));

const isDBAOrWorkspaceOwnerChecked = computed(() => {
  const { workspaceRoles } = rolloutPolicy.value;
  return (
    workspaceRoles.includes(VirtualRoleType.OWNER) &&
    workspaceRoles.includes(VirtualRoleType.DBA)
  );
});
const isProjectOwnerChecked = computed(() => {
  return rolloutPolicy.value.projectRoles.includes(PresetRoleType.OWNER);
});
const isProjectReleaserChecked = computed(() => {
  return rolloutPolicy.value.projectRoles.includes(PresetRoleType.RELEASER);
});
const isIssueCreatorChecked = computed(() => {
  return rolloutPolicy.value.issueRoles.includes(VirtualRoleType.CREATOR);
});
const isIssueLastApproverChecked = computed(() => {
  return rolloutPolicy.value.issueRoles.includes(VirtualRoleType.LAST_APPROVER);
});

const manualRolloutByFixedRolesCheckStatus = computed(() => {
  if (rolloutPolicy.value.automatic) {
    return {
      checked: false,
      indeterminate: false,
    };
  }
  const conditions = [
    isDBAOrWorkspaceOwnerChecked.value,
    isProjectOwnerChecked.value,
    isProjectReleaserChecked.value,
    isIssueCreatorChecked,
  ];
  const checkedCount = conditions.filter((checked) => checked).length;
  const checked = checkedCount === conditions.length;
  const indeterminate = checkedCount > 0 && checkedCount < conditions.length;
  return {
    checked,
    indeterminate,
  };
});

const update = (rp: RolloutPolicy) => {
  if (
    rp.issueRoles.length === 0 &&
    rp.projectRoles.length === 0 &&
    rp.workspaceRoles.length === 0
  ) {
    // normalize
    rp.automatic = true;
  }
  emit("update:policy", {
    ...props.policy,
    rolloutPolicy: rp,
  });
};
const selectAutomaticRollout = (checked: boolean) => {
  if (!checked) return;
  update(
    RolloutPolicy.fromPartial({
      automatic: true,
    })
  );
};
const toggleAllFixedRoles = (checked: boolean) => {
  if (checked) {
    update(
      RolloutPolicy.fromPartial({
        automatic: false,
        workspaceRoles: [VirtualRoleType.OWNER, VirtualRoleType.DBA],
        projectRoles: [PresetRoleType.OWNER, PresetRoleType.RELEASER],
        issueRoles: [VirtualRoleType.CREATOR],
      })
    );
  } else {
    selectAutomaticRollout(true);
  }
};
const toggleDBAOrWorkspaceOwner = (checked: boolean) => {
  const rp = rolloutPolicy.value;
  const workspaceRoles = new Set(rp.workspaceRoles);
  if (checked) {
    workspaceRoles.add(VirtualRoleType.OWNER);
    workspaceRoles.add(VirtualRoleType.DBA);
  } else {
    workspaceRoles.delete(VirtualRoleType.OWNER);
    workspaceRoles.delete(VirtualRoleType.DBA);
  }
  update(
    RolloutPolicy.fromPartial({
      automatic: false,
      workspaceRoles: Array.from(workspaceRoles),
      projectRoles: [...rp.projectRoles],
      issueRoles: [...rp.issueRoles],
    })
  );
};
const toggleProjectRoles = (checked: boolean, roles: string[]) => {
  const rp = rolloutPolicy.value;
  const projectRoles = new Set(rp.projectRoles);
  if (checked) {
    roles.forEach((role) => projectRoles.add(role));
  } else {
    roles.forEach((role) => projectRoles.delete(role));
  }
  update(
    RolloutPolicy.fromPartial({
      automatic: false,
      workspaceRoles: [...rp.workspaceRoles],
      projectRoles: Array.from(projectRoles),
      issueRoles: [...rp.issueRoles],
    })
  );
};
const toggleIssueRoles = (checked: boolean, roles: string[]) => {
  const rp = rolloutPolicy.value;
  const issueRoles = new Set(rp.issueRoles);
  if (checked) {
    roles.forEach((role) => issueRoles.add(role));
  } else {
    roles.forEach((role) => issueRoles.delete(role));
  }
  update(
    RolloutPolicy.fromPartial({
      automatic: false,
      workspaceRoles: [...rp.workspaceRoles],
      projectRoles: [...rp.projectRoles],
      issueRoles: Array.from(issueRoles),
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

<style lang="postcss" scoped>
.rollout-policy-config :deep(.n-radio),
.rollout-policy-config :deep(.n-checkbox) {
  --n-label-padding: 0 0 0 1rem !important;
}
</style>
