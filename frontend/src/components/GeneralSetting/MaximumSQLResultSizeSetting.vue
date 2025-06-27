<template>
  <div class="flex flex-col space-y-7">
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
      </p>
      <div class="mt-3 w-full flex flex-row justify-start items-center gap-4">
        <NInputNumber
          :value="queryRestriction.maximumResultSize"
          :disabled="!allowEdit || !hasQueryPolicyFeature"
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
      </p>
      <p class="text-sm text-gray-400 mt-1">
        {{
          $t("settings.general.workspace.maximum-sql-result.rows.description")
        }}
      </p>
      <div class="mt-3 w-full flex flex-row justify-start items-center gap-4">
        <NInputNumber
          :value="queryRestriction.maximumResultRows"
          :disabled="!allowEdit || !hasQueryPolicyFeature"
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
          <template #suffix> Rows </template>
        </NInputNumber>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { create } from "@bufbuild/protobuf";
import { isEqual } from "lodash-es";
import { NInputNumber } from "naive-ui";
import { ref, computed } from "vue";
import { useSettingV1Store, featureToRef } from "@/store";
import {
  Setting_SettingName,
  SQLQueryRestrictionSettingSchema,
  ValueSchema as SettingValueSchema,
} from "@/types/proto-es/v1/setting_service_pb";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import { FeatureBadge } from "../FeatureGuard";

defineProps<{
  allowEdit: boolean;
}>();

const settingV1Store = useSettingV1Store();
const timing = ref<ReturnType<typeof setTimeout>>();
const hasQueryPolicyFeature = featureToRef(PlanFeature.FEATURE_QUERY_POLICY);

const initialState = () => {
  const setting = settingV1Store.getSettingByName(
    Setting_SettingName.SQL_RESULT_SIZE_LIMIT
  );
  let maximumResultSize = BigInt(100 * 1024 * 1024); // default 100
  let maximumResultRows = -1; // default -1

  if (setting?.value?.value?.case === "sqlQueryRestrictionSetting") {
    maximumResultSize =
      setting.value.value.value.maximumResultSize ?? maximumResultSize;
    maximumResultRows =
      setting.value.value.value.maximumResultRows ?? maximumResultRows;
  }

  return {
    maximumResultSize: Math.round(Number(maximumResultSize) / 1024 / 1024),
    maximumResultRows: Number(maximumResultRows),
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
  clearTimeout(timing.value);
  await settingV1Store.upsertSetting({
    name: Setting_SettingName.SQL_RESULT_SIZE_LIMIT,
    value: create(SettingValueSchema, {
      value: {
        case: "sqlQueryRestrictionSetting",
        value: create(SQLQueryRestrictionSettingSchema, {
          maximumResultSize: BigInt(
            queryRestriction.value.maximumResultSize * 1024 * 1024
          ),
          maximumResultRows: queryRestriction.value.maximumResultRows,
        }),
      },
    }),
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
