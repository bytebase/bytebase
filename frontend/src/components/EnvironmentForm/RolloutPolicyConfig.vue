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
      </div>
      <div class="w-full inline-flex items-start gap-x-2">
        <Switch
          :value="isAutomaticRolloutChecked"
          :text="true"
          :disabled="disabled"
          @update:value="toggleAutomaticRollout($event)"
        />
        <div class="flex flex-col">
          <span class="textlabel">
            {{ t("policy.rollout.auto") }}
          </span>
          <div class="textinfolabel">
            {{ t("policy.rollout.auto-info") }}
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { create } from "@bufbuild/protobuf";
import { cloneDeep } from "lodash-es";
import { computed, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import type {
  Policy,
  RolloutPolicy,
} from "@/types/proto-es/v1/org_policy_service_pb";
import { RolloutPolicySchema } from "@/types/proto-es/v1/org_policy_service_pb";
import { RoleSelect, Switch } from "../v2";

const { t } = useI18n();

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

const isAutomaticRolloutChecked = computed(() => {
  return rolloutPolicy.value.automatic;
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
