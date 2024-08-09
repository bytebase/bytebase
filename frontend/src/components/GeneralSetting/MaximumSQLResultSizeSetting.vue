<template>
  <div class="mb-7 mt-4 lg:mt-0">
    <p class="font-medium flex flex-row justify-start items-center">
      <span class="mr-2">
        {{ $t("settings.general.workspace.maximum-sql-result.self") }}
      </span>
    </p>
    <p class="text-sm text-gray-400 mt-1">
      {{ $t("settings.general.workspace.maximum-sql-result.description") }}
    </p>
    <div class="mt-3 w-full flex flex-row justify-start items-center gap-4">
      <NInputNumber
        v-model:value="maximumSQLResultLimit"
        :disabled="!allowEdit"
        :min="1"
        :precision="0"
      >
        <template #suffix> MB </template>
      </NInputNumber>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { useDebounceFn } from "@vueuse/core";
import Long from "long";
import { NInputNumber } from "naive-ui";
import { watch, ref } from "vue";
import { useI18n } from "vue-i18n";
import { pushNotification } from "@/store";
import { useSettingV1Store } from "@/store/modules/v1/setting";

defineProps<{
  allowEdit: boolean;
}>();

const { t } = useI18n();
const settingV1Store = useSettingV1Store();

const initialState = () => {
  const limit =
    settingV1Store.getSettingByName("bb.workspace.maximum-sql-result-size")
      ?.value?.maximumSqlResultSizeSetting?.limit ??
    Long.fromNumber(100 * 1024 * 1024);

  return Math.round(limit.toNumber() / 1024 / 1024);
};

// limit in MB
const maximumSQLResultLimit = ref<number>(initialState());

const handleSettingChange = useDebounceFn(async () => {
  if (maximumSQLResultLimit.value <= 0) {
    return;
  }
  await settingV1Store.upsertSetting({
    name: "bb.workspace.maximum-sql-result-size",
    value: {
      maximumSqlResultSizeSetting: {
        limit: Long.fromNumber(maximumSQLResultLimit.value * 1024 * 1024),
      },
    },
  });
  pushNotification({
    module: "bytebase",
    style: "SUCCESS",
    title: t("settings.general.workspace.config-updated"),
  });
}, 2000);

watch(
  () => maximumSQLResultLimit.value,
  () => {
    handleSettingChange();
  }
);
</script>
