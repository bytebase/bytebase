<template>
  <div class="flex flex-col gap-y-6">
    <div>
      <p class="font-medium flex flex-row justify-start items-center">
        <span class="mr-2">
          {{ $t("settings.general.workspace.maximum-sql-result.size.self") }}
        </span>
        <FeatureBadge :feature="PlanFeature.FEATURE_QUERY_POLICY" />
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
        <PermissionGuardWrapper
          v-slot="slotProps"
          :permissions="[
            'bb.settings.setWorkspaceProfile'
          ]"
        >
          <NInputNumber
            :value="queryRestriction.maximumResultSize"
            :disabled="!hasQueryPolicyFeature || slotProps.disabled"
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
        </PermissionGuardWrapper>
      </div>
    </div>
    <div>
      <p class="font-medium flex flex-row justify-start items-center">
        <span class="mr-2">
          {{ $t("settings.general.workspace.maximum-sql-result.rows.self") }}
        </span>
        <FeatureBadge :feature="PlanFeature.FEATURE_QUERY_POLICY" />
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
        <PermissionGuardWrapper
          v-slot="slotProps"
          :permissions="[
            'bb.policies.update'
          ]"
        >
          <NInputNumber
            :value="queryRestriction.maximumResultRows"
            :disabled="!hasQueryPolicyFeature || slotProps.disabled"
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
        </PermissionGuardWrapper>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { create } from "@bufbuild/protobuf";
import { FieldMaskSchema } from "@bufbuild/protobuf/wkt";
import { isEqual } from "lodash-es";
import { NInputNumber } from "naive-ui";
import { computed, ref } from "vue";
import PermissionGuardWrapper from "@/components/Permission/PermissionGuardWrapper.vue";
import {
  DEFAULT_MAX_RESULT_SIZE_IN_MB,
  featureToRef,
  usePolicyV1Store,
  useSettingV1Store,
} from "@/store";
import {
  PolicyResourceType,
  PolicyType,
  type QueryDataPolicy,
  QueryDataPolicySchema,
} from "@/types/proto-es/v1/org_policy_service_pb";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import { FeatureBadge } from "../FeatureGuard";

const props = defineProps<{
  resource: string;
  policy: QueryDataPolicy;
}>();

const policyV1Store = usePolicyV1Store();
const settingV1Store = useSettingV1Store();

const initialState = () => {
  let size = settingV1Store.workspaceProfile.dataExportResultSize;
  if (size <= 0) {
    size = BigInt(DEFAULT_MAX_RESULT_SIZE_IN_MB * 1024 * 1024);
  }
  return {
    maximumResultSize: Math.round(Number(size) / 1024 / 1024),
    maximumResultRows: Number(props.policy.maximumResultRows),
  };
};

const queryRestriction = ref<{
  maximumResultSize: number; // limit in MB
  maximumResultRows: number;
}>(initialState());

const hasQueryPolicyFeature = featureToRef(PlanFeature.FEATURE_QUERY_POLICY);

const allowUpdate = computed(() => {
  return !isEqual(initialState(), queryRestriction.value);
});

const updateChange = async () => {
  const init = initialState();
  if (init.maximumResultRows !== queryRestriction.value.maximumResultRows) {
    await policyV1Store.upsertPolicy({
      parentPath: props.resource,
      policy: {
        type: PolicyType.DATA_QUERY,
        resourceType: PolicyResourceType.WORKSPACE,
        policy: {
          case: "queryDataPolicy",
          value: create(QueryDataPolicySchema, {
            ...props.policy,
            maximumResultRows: queryRestriction.value.maximumResultRows,
          }),
        },
      },
    });
  }

  if (init.maximumResultSize !== queryRestriction.value.maximumResultSize) {
    await settingV1Store.updateWorkspaceProfile({
      payload: {
        dataExportResultSize: BigInt(
          queryRestriction.value.maximumResultSize * 1024 * 1024
        ),
      },
      updateMask: create(FieldMaskSchema, {
        paths: ["value.workspace_profile.data_export_result_size"],
      }),
    });
  }
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
