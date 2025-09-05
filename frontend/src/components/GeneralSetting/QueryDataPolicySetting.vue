<template>
  <div>
    <p class="font-medium flex flex-row justify-start items-center">
      <span class="mr-2">
        {{ $t("settings.general.workspace.query-data-policy.timeout.self") }}
      </span>
      <FeatureBadge :feature="PlanFeature.FEATURE_QUERY_POLICY" />
    </p>
    <p class="text-sm text-gray-400 mt-1">
      {{
        $t("settings.general.workspace.query-data-policy.timeout.description")
      }}
      <span class="!font-semibold textinfolabel">
        {{ $t("settings.general.workspace.maximum-sql-result.rows.limit") }}
      </span>
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
        <template #suffix>{{
          $t("settings.general.workspace.query-data-policy.seconds")
        }}</template>
      </NInputNumber>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { create } from "@bufbuild/protobuf";
import { DurationSchema } from "@bufbuild/protobuf/wkt";
import { NInputNumber } from "naive-ui";
import { computed, ref } from "vue";
import {
  featureToRef,
  usePolicyByParentAndType,
  usePolicyV1Store,
} from "@/store";
import {
  PolicyResourceType,
  PolicyType,
  QueryDataPolicySchema,
} from "@/types/proto-es/v1/org_policy_service_pb";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import { hasWorkspacePermissionV2 } from "@/utils";
import { FeatureBadge } from "../FeatureGuard";

const policyV1Store = usePolicyV1Store();
const hasQueryPolicyFeature = featureToRef(PlanFeature.FEATURE_QUERY_POLICY);

const { policy: queryDataPolicy } = usePolicyByParentAndType(
  computed(() => ({
    parentPath: "",
    policyType: PolicyType.DATA_QUERY,
  }))
);

const allowEdit = computed(
  () =>
    hasWorkspacePermissionV2("bb.policies.update") &&
    hasQueryPolicyFeature.value
);

const initialState = () => {
  if (
    queryDataPolicy.value?.policy.case === "queryDataPolicy" &&
    queryDataPolicy.value.policy.value.timeout
  ) {
    return Number(queryDataPolicy.value.policy.value.timeout.seconds);
  }
  return 0;
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
      policy: {
        case: "queryDataPolicy",
        value: create(QueryDataPolicySchema, {
          timeout: create(DurationSchema, { seconds: BigInt(seconds.value) }),
        }),
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
