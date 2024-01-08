<template>
  <div class="flex flex-col items-start gap-y-2">
    <div class="flex flex-col gap-y-2">
      <NRadio
        :checked="isAutomaticRolloutChecked"
        :disabled="disabled"
        style="--n-label-padding: 0 0 0 1rem"
        @update:checked="selectAutomaticRollout"
      >
        <div class="flex flex-col gap-y-1">
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
      <NRadio
        :checked="isManualRolloutByDedicatedRolesChecked"
        :disabled="disabled"
        style="--n-label-padding: 0 0 0 1rem"
        @update:checked="toggleAllDedicatedRoles"
      >
        <div class="flex flex-col gap-y-1">
          <div class="textlabel flex flex-row gap-x-1">
            <span>{{ $t("policy.rollout.manual-by-dedicated-roles") }}</span>
            <FeatureBadge feature="bb.feature.approval-policy" />
          </div>
          <div class="textinfolabel">
            {{ $t("policy.rollout.manual-by-dedicated-roles-info") }}
          </div>
        </div>
      </NRadio>
      <div class="flex flex-col gap-y-2 pl-8">
        <NCheckbox
          :checked="isWorkspaceOwnerChecked"
          :disabled="disabled"
          style="--n-label-padding: 0 0 0 1rem"
          @update:checked="
            toggleDedicatedRoles($event, 'workspace', [
              VirtualRoleType.WORKSPACE_ADMIN,
            ])
          "
        >
          <div class="textlabel">
            {{ $t("policy.rollout.manual-by-workspace-admin") }}
          </div>
        </NCheckbox>
        <NCheckbox
          :checked="isDBAChecked"
          :disabled="disabled"
          style="--n-label-padding: 0 0 0 1rem"
          @update:checked="
            toggleDedicatedRoles($event, 'workspace', [
              VirtualRoleType.WORKSPACE_DBA,
            ])
          "
        >
          <div class="textlabel">
            {{ $t("policy.rollout.manual-by-dba") }}
          </div>
        </NCheckbox>
        <NCheckbox
          :checked="isProjectOwnerChecked"
          :disabled="disabled"
          style="--n-label-padding: 0 0 0 1rem"
          @update:checked="
            toggleDedicatedRoles($event, 'project', [
              PresetRoleType.PROJECT_OWNER,
            ])
          "
        >
          <div class="textlabel">
            {{ $t("policy.rollout.manual-by-project-owner") }}
          </div>
        </NCheckbox>
        <NCheckbox
          :checked="isProjectReleaserChecked"
          :disabled="disabled"
          style="--n-label-padding: 0 0 0 1rem"
          @update:checked="
            toggleDedicatedRoles($event, 'project', [
              PresetRoleType.PROJECT_RELEASER,
            ])
          "
        >
          <div class="textlabel">
            {{ $t("policy.rollout.manual-by-project-releaser") }}
          </div>
        </NCheckbox>
        <NCheckbox
          :checked="isIssueCreatorChecked"
          :disabled="disabled"
          style="--n-label-padding: 0 0 0 1rem"
          @update:checked="
            toggleDedicatedRoles($event, 'issue', [VirtualRoleType.CREATOR])
          "
        >
          <div class="textlabel">
            {{ $t("policy.rollout.manual-by-issue-creator") }}
          </div>
        </NCheckbox>
        <div class="flex flex-col gap-y-1">
          <NCheckbox
            :checked="customProjectRoles.checked"
            :disabled="disabled"
            style="--n-label-padding: 0 0 0 1rem"
            @update:checked="toggleCustomProjectRoles($event)"
          >
            <div class="textlabel">
              {{ $t("policy.rollout.manual-by-custom-project-roles") }}
            </div>
          </NCheckbox>
          <div v-if="customProjectRoles.checked" class="pl-8">
            <ProjectRoleSelect
              :roles="customProjectRoles.roles"
              :multiple="true"
              :filter="filterProjectRole"
              :status="
                customProjectRoles.roles.length === 0 ? 'error' : undefined
              "
              :tooltip-props="{
                placement: 'right',
              }"
              style="width: 24rem"
              @update:roles="handleUpdateCustomProjectRoles"
            />
          </div>
        </div>
      </div>
    </div>
    <div class="flex flex-col gap-y-2">
      <NRadio
        :checked="isIssueLastApproverChecked"
        :disabled="disabled"
        style="--n-label-padding: 0 0 0 1rem"
        @update:checked="toggleLastApprover"
      >
        <div class="textlabel flex flex-row gap-x-1">
          <span>{{ $t("policy.rollout.manual-by-last-approver") }}</span>
          <FeatureBadge feature="bb.feature.custom-approval" />
        </div>
      </NRadio>
    </div>
  </div>
</template>

<script setup lang="ts">
import { cloneDeep, pull } from "lodash-es";
import { NCheckbox, NRadio } from "naive-ui";
import { ref, watch } from "vue";
import { computed } from "vue";
import { PresetRoleType, VirtualRoleType } from "@/types";
import { Policy, RolloutPolicy } from "@/types/proto/v1/org_policy_service";
import { Role } from "@/types/proto/v1/role_service";
import FeatureBadge from "../FeatureGuard/FeatureBadge.vue";

const props = defineProps<{
  policy: Policy;
  disabled?: boolean;
}>();
const emit = defineEmits<{
  (event: "update:policy", policy: Policy): void;
}>();

const rolloutPolicy = ref(cloneDeep(props.policy.rolloutPolicy!));
const PresetProjectRoles = [
  PresetRoleType.PROJECT_OWNER,
  PresetRoleType.PROJECT_RELEASER,
];

const isAutomaticRolloutChecked = computed(() => {
  return rolloutPolicy.value.automatic && !customProjectRoles.value.checked;
});
const isDBAChecked = computed(() => {
  return rolloutPolicy.value.workspaceRoles.includes(
    VirtualRoleType.WORKSPACE_DBA
  );
});
const isWorkspaceOwnerChecked = computed(() => {
  return rolloutPolicy.value.workspaceRoles.includes(
    VirtualRoleType.WORKSPACE_ADMIN
  );
});
const isProjectOwnerChecked = computed(() => {
  return rolloutPolicy.value.projectRoles.includes(
    PresetRoleType.PROJECT_OWNER
  );
});
const isProjectReleaserChecked = computed(() => {
  return rolloutPolicy.value.projectRoles.includes(
    PresetRoleType.PROJECT_RELEASER
  );
});
const isIssueCreatorChecked = computed(() => {
  return rolloutPolicy.value.issueRoles.includes(VirtualRoleType.CREATOR);
});

const extractCustomProjectRolesFromRolloutPolicy = (rp: RolloutPolicy) => {
  const customRoles = rp.projectRoles.filter((role) => {
    return !PresetProjectRoles.includes(role);
  });
  return {
    checked: customRoles.length > 0,
    roles: customRoles,
  };
};
const customProjectRoles = ref(
  extractCustomProjectRolesFromRolloutPolicy(rolloutPolicy.value)
);

const isIssueLastApproverChecked = computed(() => {
  return (
    rolloutPolicy.value.issueRoles.includes(VirtualRoleType.LAST_APPROVER) &&
    !customProjectRoles.value.checked
  );
});
const isManualRolloutByDedicatedRolesChecked = computed(() => {
  if (customProjectRoles.value.checked) {
    return true;
  }

  if (rolloutPolicy.value.automatic) {
    return false;
  }

  if (isIssueLastApproverChecked.value) {
    return false;
  }

  const conditions = [
    isDBAChecked.value,
    isWorkspaceOwnerChecked.value,
    isProjectOwnerChecked.value,
    isProjectReleaserChecked.value,
    isIssueCreatorChecked.value,
  ];
  return conditions.some((checked) => checked);
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
const toggleAllDedicatedRoles = (checked: boolean) => {
  if (!checked) return;
  update(
    RolloutPolicy.fromPartial({
      automatic: false,
      workspaceRoles: [
        VirtualRoleType.WORKSPACE_ADMIN,
        VirtualRoleType.WORKSPACE_DBA,
      ],
      projectRoles: [
        PresetRoleType.PROJECT_OWNER,
        PresetRoleType.PROJECT_RELEASER,
      ],
      issueRoles: [VirtualRoleType.CREATOR],
    })
  );
};
const toggleLastApprover = (checked: boolean) => {
  if (!checked) return;
  update(
    RolloutPolicy.fromPartial({
      automatic: false,
      workspaceRoles: [],
      projectRoles: [],
      issueRoles: [VirtualRoleType.LAST_APPROVER],
    })
  );
};
const toggleDedicatedRoles = (
  checked: boolean,
  type: "workspace" | "project" | "issue",
  roles: string[]
) => {
  const rp = rolloutPolicy.value;
  const key = `${type}Roles` as `${typeof type}Roles`;
  const set = new Set(rp[key]);
  if (checked) {
    roles.forEach((role) => set.add(role));
  } else {
    roles.forEach((role) => set.delete(role));
  }
  const patch = RolloutPolicy.fromPartial({
    ...rp,
    automatic: false,
    [key]: Array.from(set),
  });
  pull(patch.issueRoles, VirtualRoleType.LAST_APPROVER);
  update(patch);
};
const toggleCustomProjectRoles = (checked: boolean) => {
  if (checked) {
    customProjectRoles.value = {
      checked: true,
      roles: [],
    };
  } else {
    const rp = rolloutPolicy.value;
    // Remove custom project roles
    // remaining preset project roles
    const set = new Set(
      rp.projectRoles.filter((role) => PresetProjectRoles.includes(role))
    );
    const patch = RolloutPolicy.fromPartial({
      ...rp,
      automatic: false,
      projectRoles: Array.from(set),
    });
    pull(patch.issueRoles, VirtualRoleType.LAST_APPROVER);
    customProjectRoles.value = {
      checked: false,
      roles: [],
    };
    update(patch);
  }
};
const handleUpdateCustomProjectRoles = (roles: string[]) => {
  const rp = rolloutPolicy.value;
  const presetProjectRoleSet = new Set(
    rp.projectRoles.filter((role) => PresetProjectRoles.includes(role))
  );
  const customProjectRoleSet = new Set(roles);
  const patch = RolloutPolicy.fromPartial({
    ...rp,
    automatic: false,
    projectRoles: [
      ...Array.from(presetProjectRoleSet),
      ...Array.from(customProjectRoleSet),
    ],
  });
  pull(patch.issueRoles, VirtualRoleType.LAST_APPROVER);
  customProjectRoles.value.roles = roles;
  update(patch);
};

const filterProjectRole = (role: Role) => {
  return !PresetProjectRoles.includes(role.name);
};

watch(
  () => props.policy.rolloutPolicy!,
  (p) => {
    rolloutPolicy.value = cloneDeep(p);
    customProjectRoles.value = extractCustomProjectRolesFromRolloutPolicy(p);
  },
  { immediate: true, deep: true }
);
</script>
