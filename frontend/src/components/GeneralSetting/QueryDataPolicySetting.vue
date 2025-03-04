<template>
  <div>
    <p class="font-medium flex flex-row justify-start items-center">
      <span class="mr-2">
        {{ $t("settings.general.workspace.query-data-policy.timeout.self") }}
      </span>
      <FeatureBadge feature="bb.feature.access-control" />
    </p>
    <p class="text-sm text-gray-400 mt-1">
      {{
        $t("settings.general.workspace.query-data-policy.timeout.description")
      }}
    </p>
    <div class="mt-3 w-full flex flex-row justify-start items-center gap-4">
      <NInputNumber
        :value="seconds"
        :disabled="!allowEdit"
        class="w-60"
        :min="0"
        :precision="0"
        @update:value="handleInput"
      >
        <template #suffix>Seconds</template>
      </NInputNumber>
    </div>
    <p class="text-sm textinfolabel mt-1" v-if="seconds <= 0">
      {{
        $t("settings.general.workspace.query-data-policy.timeout.no-time-limit")
      }}
    </p>
  </div>
</template>

<script lang="ts" setup>
import { NInputNumber } from "naive-ui";
import { ref, computed } from "vue";
import {
  featureToRef,
  usePolicyByParentAndType,
  usePolicyV1Store,
} from "@/store";
import { Duration } from "@/types/proto/google/protobuf/duration";
import {
  PolicyResourceType,
  PolicyType,
} from "@/types/proto/v1/org_policy_service";
import { hasWorkspacePermissionV2 } from "@/utils";
import { FeatureBadge } from "../FeatureGuard";

const policyV1Store = usePolicyV1Store();
const hasAccessControlFeature = featureToRef("bb.feature.access-control");

const { policy: queryDataPolicy } = usePolicyByParentAndType(
  computed(() => ({
    parentPath: "",
    policyType: PolicyType.DATA_QUERY,
  }))
);

const allowEdit = computed(
  () =>
    hasWorkspacePermissionV2("bb.policies.update") &&
    hasAccessControlFeature.value
);

const initialState = () => {
  return (
    queryDataPolicy.value?.queryDataPolicy?.timeout?.seconds.toNumber() ?? 0
  );
};

// limit in seconds.
const seconds = ref<number>(initialState());

const allowUpdate = computed(() => {
  return seconds.value !== initialState();
});

const updateChange = async () => {
  await policyV1Store.upsertPolicy({
    parentPath: "",
    policy: {
      type: PolicyType.DATA_QUERY,
      resourceType: PolicyResourceType.WORKSPACE,
      queryDataPolicy: {
        timeout: Duration.fromPartial({ seconds: seconds.value }),
      },
    },
  });
};

const handleInput = (value: number | null) => {
  if (value === null) return;
  if (value === undefined) return;
  seconds.value = value;
};

defineExpose({
  isDirty: allowUpdate,
  update: updateChange,
  revert: () => (seconds.value = initialState()),
});
</script>
