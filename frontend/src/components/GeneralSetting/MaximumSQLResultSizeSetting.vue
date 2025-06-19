<template>
  <div>
    <p class="font-medium flex flex-row justify-start items-center">
      <span class="mr-2">
        {{ $t("settings.general.workspace.maximum-sql-result.self") }}
      </span>
      <FeatureBadge :feature="PlanFeature.FEATURE_QUERY_POLICY" />
    </p>
    <p class="text-sm text-gray-400 mt-1">
      {{ $t("settings.general.workspace.maximum-sql-result.description") }}
    </p>
    <div class="mt-3 w-full flex flex-row justify-start items-center gap-4">
      <NInputNumber
        :value="maximumSQLResultLimit"
        :disabled="!allowEdit || !hasQueryPolicyFeature"
        class="w-60"
        :min="1"
        :precision="0"
        @update:value="handleInput"
      >
        <template #suffix> MB </template>
      </NInputNumber>
    </div>
  </div>
</template>

<script lang="ts" setup>
import Long from "long";
import { NInputNumber } from "naive-ui";
import { ref, computed } from "vue";
import { useSettingV1Store, featureToRef } from "@/store";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import { Setting_SettingName } from "@/types/proto/v1/setting_service";
import { FeatureBadge } from "../FeatureGuard";

defineProps<{
  allowEdit: boolean;
}>();

const settingV1Store = useSettingV1Store();
const timing = ref<ReturnType<typeof setTimeout>>();
const hasQueryPolicyFeature = featureToRef(PlanFeature.FEATURE_QUERY_POLICY);

const initialState = () => {
  const limit =
    settingV1Store.getSettingByName(Setting_SettingName.SQL_RESULT_SIZE_LIMIT)
      ?.value?.maximumSqlResultSizeSetting?.limit ??
    Long.fromNumber(100 * 1024 * 1024);

  return Math.round(limit.toNumber() / 1024 / 1024);
};

// limit in MB
const maximumSQLResultLimit = ref<number>(initialState());
const allowUpdate = computed(() => {
  return maximumSQLResultLimit.value !== initialState();
});

const updateChange = async () => {
  if (maximumSQLResultLimit.value <= 0) {
    return;
  }
  clearTimeout(timing.value);
  await settingV1Store.upsertSetting({
    name: Setting_SettingName.SQL_RESULT_SIZE_LIMIT,
    value: {
      maximumSqlResultSizeSetting: {
        limit: Long.fromNumber(maximumSQLResultLimit.value * 1024 * 1024),
      },
    },
  });
};

const handleInput = (value: number | null) => {
  if (value === null) return;
  if (value === undefined) return;
  maximumSQLResultLimit.value = value;
};

defineExpose({
  isDirty: allowUpdate,
  update: updateChange,
  revert: () => (maximumSQLResultLimit.value = initialState()),
});
</script>
