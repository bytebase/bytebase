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
          {{ $t("role.issue-creator.self") }}
        </NCheckbox>
        <NCheckbox
          v-if="shouldShowLastApprover"
          :checked="isIssueLastApproverChecked"
          :disabled="disabled"
          @update:checked="
            toggleIssueRoles($event, VirtualRoleType.LAST_APPROVER)
          "
        >
          {{ $t("policy.rollout.last-issue-approver") }}
        </NCheckbox>
      </div>
      <NCheckbox
        :checked="isAutomaticRolloutChecked"
        :disabled="disabled"
        @update:checked="toggleAutomaticRollout($event)"
      >
        <div class="flex flex-col gap-y-1">
          <div>
            {{ $t("policy.rollout.auto") }}
          </div>
          <div class="textinfolabel">
            {{ $t("policy.rollout.auto-info") }}
          </div>
        </div>
      </NCheckbox>
    </div>

    <!-- Requirements Section -->
    <div v-if="isDev()" class="flex flex-col gap-y-2">
      <div class="textlabel font-medium">
        {{ $t("policy.rollout.checkers.self") }}
      </div>

      <!-- Required Issue Approval -->
      <NCheckbox
        :checked="isRequiredIssueApprovalChecked"
        :disabled="disabled"
        @update:checked="toggleRequiredIssueApproval($event)"
      >
        <div class="flex flex-col gap-y-1">
          <div>
            {{ $t("policy.rollout.checkers.required-issue-approval.self") }}
          </div>
          <div class="textinfolabel">
            {{
              $t("policy.rollout.checkers.required-issue-approval.description")
            }}
          </div>
        </div>
      </NCheckbox>

      <!-- Plan Check Enforcement -->
      <NCheckbox
        :checked="isPlanCheckEnforcementEnabled"
        :disabled="disabled"
        @update:checked="togglePlanCheckEnforcement($event)"
      >
        <div class="flex flex-col gap-y-1">
          <div>
            {{ $t("policy.rollout.checkers.plan-check-enforcement.self") }}
          </div>
          <div class="textinfolabel">
            {{
              $t("policy.rollout.checkers.plan-check-enforcement.description")
            }}
          </div>
        </div>
      </NCheckbox>

      <!-- Plan Check Enforcement Options (shown when enabled) -->
      <div
        v-if="isPlanCheckEnforcementEnabled"
        class="flex flex-col gap-y-2 ml-6"
      >
        <NRadioGroup
          :value="planCheckEnforcementLevel"
          :disabled="disabled"
          @update:value="updatePlanCheckEnforcementLevel($event)"
        >
          <div class="flex flex-col gap-y-2">
            <NRadio
              :value="RolloutPolicy_Checkers_PlanCheckEnforcement.ERROR_ONLY"
            >
              <div class="flex flex-col">
                <div>
                  {{
                    $t(
                      "policy.rollout.checkers.plan-check-enforcement.error-only.self"
                    )
                  }}
                </div>
                <div class="textinfolabel">
                  {{
                    $t(
                      "policy.rollout.checkers.plan-check-enforcement.error-only.description"
                    )
                  }}
                </div>
              </div>
            </NRadio>
            <NRadio :value="RolloutPolicy_Checkers_PlanCheckEnforcement.STRICT">
              <div class="flex flex-col">
                <div>
                  {{
                    $t(
                      "policy.rollout.checkers.plan-check-enforcement.strict.self"
                    )
                  }}
                </div>
                <div class="textinfolabel">
                  {{
                    $t(
                      "policy.rollout.checkers.plan-check-enforcement.strict.description"
                    )
                  }}
                </div>
              </div>
            </NRadio>
          </div>
        </NRadioGroup>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { create } from "@bufbuild/protobuf";
import { cloneDeep, uniq } from "lodash-es";
import { NCheckbox, NRadioGroup, NRadio } from "naive-ui";
import { ref, watch, computed } from "vue";
import { VirtualRoleType } from "@/types";
import type {
  Policy,
  RolloutPolicy,
} from "@/types/proto-es/v1/org_policy_service_pb";
import {
  RolloutPolicySchema,
  RolloutPolicy_CheckersSchema,
  RolloutPolicy_Checkers_RequiredStatusChecksSchema,
  RolloutPolicy_Checkers_PlanCheckEnforcement,
} from "@/types/proto-es/v1/org_policy_service_pb";
import { isDev } from "@/utils";
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

const isRequiredIssueApprovalChecked = computed(() => {
  return rolloutPolicy.value.checkers?.requiredIssueApproval ?? false;
});

const isPlanCheckEnforcementEnabled = computed(() => {
  const enforcement =
    rolloutPolicy.value.checkers?.requiredStatusChecks?.planCheckEnforcement;
  return (
    enforcement !== undefined &&
    enforcement !==
      RolloutPolicy_Checkers_PlanCheckEnforcement.PLAN_CHECK_ENFORCEMENT_UNSPECIFIED
  );
});

const planCheckEnforcementLevel = computed(() => {
  return (
    rolloutPolicy.value.checkers?.requiredStatusChecks?.planCheckEnforcement ??
    RolloutPolicy_Checkers_PlanCheckEnforcement.ERROR_ONLY
  );
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

// Checkers methods
const toggleRequiredIssueApproval = (checked: boolean) => {
  const currentCheckers = rolloutPolicy.value.checkers;
  const updatedCheckers = create(RolloutPolicy_CheckersSchema, {
    requiredIssueApproval: checked,
    requiredStatusChecks: currentCheckers?.requiredStatusChecks,
  });

  update(
    create(RolloutPolicySchema, {
      ...rolloutPolicy.value,
      checkers: updatedCheckers,
    })
  );
};

const togglePlanCheckEnforcement = (enabled: boolean) => {
  const currentCheckers = rolloutPolicy.value.checkers;
  let updatedStatusChecks;

  if (enabled) {
    // When enabling, default to ERROR_ONLY
    updatedStatusChecks = create(
      RolloutPolicy_Checkers_RequiredStatusChecksSchema,
      {
        planCheckEnforcement:
          RolloutPolicy_Checkers_PlanCheckEnforcement.ERROR_ONLY,
      }
    );
  } else {
    // When disabling, set to UNSPECIFIED (no enforcement)
    updatedStatusChecks = create(
      RolloutPolicy_Checkers_RequiredStatusChecksSchema,
      {
        planCheckEnforcement:
          RolloutPolicy_Checkers_PlanCheckEnforcement.PLAN_CHECK_ENFORCEMENT_UNSPECIFIED,
      }
    );
  }

  const updatedCheckers = create(RolloutPolicy_CheckersSchema, {
    requiredIssueApproval: currentCheckers?.requiredIssueApproval ?? false,
    requiredStatusChecks: updatedStatusChecks,
  });

  update(
    create(RolloutPolicySchema, {
      ...rolloutPolicy.value,
      checkers: updatedCheckers,
    })
  );
};

const updatePlanCheckEnforcementLevel = (
  enforcement: RolloutPolicy_Checkers_PlanCheckEnforcement
) => {
  const currentCheckers = rolloutPolicy.value.checkers;
  const updatedStatusChecks = create(
    RolloutPolicy_Checkers_RequiredStatusChecksSchema,
    {
      planCheckEnforcement: enforcement,
    }
  );
  const updatedCheckers = create(RolloutPolicy_CheckersSchema, {
    requiredIssueApproval: currentCheckers?.requiredIssueApproval ?? false,
    requiredStatusChecks: updatedStatusChecks,
  });

  update(
    create(RolloutPolicySchema, {
      ...rolloutPolicy.value,
      checkers: updatedCheckers,
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
