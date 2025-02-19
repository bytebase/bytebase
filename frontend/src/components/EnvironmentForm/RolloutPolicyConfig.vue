<template>
  <div class="flex flex-col items-start gap-y-2">
    <div class="flex flex-col gap-y-2">
      <NRadio
        :checked="isAutomaticRolloutChecked"
        :disabled="disabled"
        style="--n-label-padding: 0 0 0 1rem"
        @update:checked="toggleAutomaticRollout(true)"
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
        :checked="!isAutomaticRolloutChecked"
        :disabled="disabled"
        style="--n-label-padding: 0 0 0 1rem"
        @update:checked="toggleAutomaticRollout(false)"
      >
        <div class="flex flex-col gap-y-1">
          <div class="textlabel flex flex-row gap-x-1">
            <span>{{ $t("policy.rollout.manual-by-dedicated-roles") }}</span>
            <FeatureBadge feature="bb.feature.rollout-policy" />
            <FeatureBadgeForInstanceLicense
              :show="hasRolloutPolicyFeature"
              feature="bb.feature.rollout-policy"
              :tooltip="$t('subscription.instance-assignment.require-license')"
            />
          </div>
          <div class="textinfolabel">
            {{ $t("policy.rollout.manual-by-dedicated-roles-info") }}
          </div>
        </div>
      </NRadio>
      <div class="flex flex-col gap-y-2 pl-8" v-if="!isAutomaticRolloutChecked">
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
            {{ $t("policy.rollout.last-approver-from-custom-approval") }}
          </div>
        </NCheckbox>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { cloneDeep, uniq } from "lodash-es";
import { NCheckbox, NRadio } from "naive-ui";
import { ref, watch } from "vue";
import { computed } from "vue";
import { featureToRef } from "@/store";
import { VirtualRoleType } from "@/types";
import type { Policy } from "@/types/proto/v1/org_policy_service";
import { RolloutPolicy } from "@/types/proto/v1/org_policy_service";
import FeatureBadge from "../FeatureGuard/FeatureBadge.vue";
import FeatureBadgeForInstanceLicense from "../FeatureGuard/FeatureBadgeForInstanceLicense.vue";
import { RoleSelect } from "../v2";

const props = defineProps<{
  policy: Policy;
  disabled?: boolean;
}>();

const emit = defineEmits<{
  (event: "update:policy", policy: Policy): void;
}>();

const rolloutPolicy = ref(cloneDeep(props.policy.rolloutPolicy!));

const hasRolloutPolicyFeature = featureToRef("bb.feature.rollout-policy");

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
const toggleAutomaticRollout = (checked: boolean) => {
  update(
    RolloutPolicy.fromPartial(
      checked
        ? {
            automatic: true,
            roles: [],
            issueRoles: [],
          }
        : {
            automatic: false,
            roles: rolloutPolicy.value.roles,
            issueRoles: rolloutPolicy.value.issueRoles,
          }
    )
  );
};
const toggleIssueRoles = (checked: boolean, role: string) => {
  const roles = rolloutPolicy.value.issueRoles;
  if (checked) {
    roles.push(role);
  } else {
    const index = roles.indexOf(role);
    if (index !== -1) {
      roles.splice(index, 1);
    }
  }
  update(
    RolloutPolicy.fromPartial({
      automatic: false,
      roles: rolloutPolicy.value.roles,
      issueRoles: uniq(roles),
    })
  );
};
const updateRoles = (roles: string[]) => {
  update(
    RolloutPolicy.fromPartial({
      automatic: false,
      roles,
      issueRoles: rolloutPolicy.value.issueRoles,
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
