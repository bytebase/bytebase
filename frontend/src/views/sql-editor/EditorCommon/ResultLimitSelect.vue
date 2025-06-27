<template>
  <NPopselect
    v-model:value="resultRowsLimit"
    :options="options"
    trigger="click"
    placement="bottom-start"
    scrollable
  >
    <slot name="default" :result-rows-limit="resultRowsLimit">
      <NButton size="small">
        {{ $t("sql-editor.result-limit.n-rows", { n: resultRowsLimit }) }}
      </NButton>
    </slot>
    <template #action>
      <div class="flex items-center justify-between gap-1">
        <NInputNumber
          v-model:value="resultRowsLimit"
          :show-button="false"
          :min="0"
          :max="Math.min(maximumResultRows, 100000)"
          style="width: 5rem"
          size="small"
        />
        <span>{{ $t("sql-editor.result-limit.rows") }}</span>
      </div>
    </template>
  </NPopselect>
</template>

<script setup lang="ts">
import { last } from "lodash-es";
import { NButton, NInputNumber, NPopselect, type SelectOption } from "naive-ui";
import { storeToRefs } from "pinia";
import { computed } from "vue";
import { watchEffect } from "vue";
import { useI18n } from "vue-i18n";
import { useSQLEditorStore, useSettingV1Store } from "@/store";
import { Setting_SettingName } from "@/types/proto-es/v1/setting_service_pb";

const { t } = useI18n();
const { resultRowsLimit } = storeToRefs(useSQLEditorStore());
const settingV1Store = useSettingV1Store();

const maximumResultRows = computed(() => {
  const setting = settingV1Store.getSettingByName(
    Setting_SettingName.SQL_RESULT_SIZE_LIMIT
  );
  if (setting?.value?.value?.case === "sqlQueryRestrictionSetting") {
    return setting.value.value.value.maximumResultRows ?? Number.MAX_VALUE;
  }
  return Number.MAX_VALUE;
});

const options = computed((): SelectOption[] => {
  const list = [100, 500, 1000, 5000, 10000, 100000].filter(
    (num) => num <= maximumResultRows.value
  );
  if (!list.includes(maximumResultRows.value)) {
    list.push(maximumResultRows.value);
  }

  return list.map((n) => ({
    label: t("sql-editor.result-limit.n-rows", { n }),
    value: n,
  }));
});

watchEffect(() => {
  if (resultRowsLimit.value > maximumResultRows.value) {
    resultRowsLimit.value = last(options.value)!.value as number;
  }
});
</script>
