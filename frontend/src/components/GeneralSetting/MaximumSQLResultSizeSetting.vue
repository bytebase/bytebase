<template>
  <div class="flex flex-col gap-y-6">
    <div>
      <p class="font-medium flex flex-row justify-start items-center">
        <span class="mr-2">
          {{ $t("settings.general.workspace.maximum-sql-result.size.self") }}
        </span>
        <FeatureBadge :feature="PlanFeature.FEATURE_QUERY_POLICY" />
        <NTooltip v-if="tooltip">
          <template #trigger>
            <CircleQuestionMarkIcon class="w-4 textinfolabel" />
          </template>
          <span>
            {{ tooltip }}
          </span>
        </NTooltip>
      </p>
      <p class="text-sm text-gray-400 mt-1">
        {{
          $t("settings.general.workspace.maximum-sql-result.size.description")
        }}
        <span class="font-semibold! textinfolabel">
          {{
            $t("settings.general.workspace.maximum-sql-result.size.default", {
              limit: DEFAULT_MAX_RESULT_SIZE_IN_MB,
            })
          }}
        </span>
      </p>
      <div class="mt-3 w-full flex flex-row justify-start items-center gap-4">
        <NInputNumber
          :value="queryRestriction.maximumResultSize"
          :disabled="!allowEdit"
          class="w-60"
          :min="1"
          :precision="0"
          @update:value="
            handleInput(
              $event,
              (val: number) => (queryRestriction.maximumResultSize = val)
            )
          "
        >
          <template #suffix> MB </template>
        </NInputNumber>
      </div>
    </div>
    <div>
      <p class="font-medium flex flex-row justify-start items-center">
        <span class="mr-2">
          {{ $t("settings.general.workspace.maximum-sql-result.rows.self") }}
        </span>
        <FeatureBadge :feature="PlanFeature.FEATURE_QUERY_POLICY" />
        <NTooltip v-if="tooltip">
          <template #trigger>
            <CircleQuestionMarkIcon class="w-4 textinfolabel" />
          </template>
          <span>
            {{ tooltip }}
          </span>
        </NTooltip>
      </p>
      <p class="text-sm text-gray-400 mt-1">
        {{
          $t("settings.general.workspace.maximum-sql-result.rows.description")
        }}
        <span class="font-semibold! textinfolabel">
          {{ $t("settings.general.workspace.no-limit") }}
        </span>
      </p>
      <div class="mt-3 w-full flex flex-row justify-start items-center gap-4">
        <NInputNumber
          :value="queryRestriction.maximumResultRows"
          :disabled="!allowEdit"
          class="w-60"
          :min="-1"
          :precision="0"
          @update:value="
            handleInput(
              $event,
              (val: number) => (queryRestriction.maximumResultRows = val)
            )
          "
        >
          <template #suffix>{{
            $t("settings.general.workspace.maximum-sql-result.rows.rows")
          }}</template>
        </NInputNumber>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { create } from "@bufbuild/protobuf";
import { isEqual } from "lodash-es";
import { CircleQuestionMarkIcon } from "lucide-vue-next";
import { NInputNumber, NTooltip } from "naive-ui";
import { computed, ref } from "vue";
import { DEFAULT_MAX_RESULT_SIZE_IN_MB, usePolicyV1Store } from "@/store";
import {
  PolicyResourceType,
  PolicyType,
  type QueryDataPolicy,
  QueryDataPolicySchema,
} from "@/types/proto-es/v1/org_policy_service_pb";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import { FeatureBadge } from "../FeatureGuard";

const props = defineProps<{
  allowEdit: boolean;
  resource: string;
  policy: QueryDataPolicy;
  tooltip?: string;
}>();

const policyV1Store = usePolicyV1Store();

const initialState = () => {
  return {
    maximumResultSize: Math.round(
      Number(props.policy.maximumResultSize) / 1024 / 1024
    ),
    maximumResultRows: Number(props.policy.maximumResultRows),
  };
};

const queryRestriction = ref<{
  maximumResultSize: number; // limit in MB
  maximumResultRows: number;
}>(initialState());

const allowUpdate = computed(() => {
  return !isEqual(initialState(), queryRestriction.value);
});

const updateChange = async () => {
  await policyV1Store.upsertPolicy({
    parentPath: props.resource,
    policy: {
      type: PolicyType.DATA_QUERY,
      resourceType: PolicyResourceType.WORKSPACE,
      policy: {
        case: "queryDataPolicy",
        value: create(QueryDataPolicySchema, {
          ...props.policy,
          maximumResultSize: BigInt(
            queryRestriction.value.maximumResultSize * 1024 * 1024
          ),
          maximumResultRows: queryRestriction.value.maximumResultRows,
        }),
      },
    },
  });
};

const handleInput = (value: number | null, callback: (val: number) => void) => {
  if (value === null) return;
  if (value === undefined) return;
  callback(value);
};

defineExpose({
  isDirty: allowUpdate,
  update: updateChange,
  revert: () => (queryRestriction.value = initialState()),
});
</script>
