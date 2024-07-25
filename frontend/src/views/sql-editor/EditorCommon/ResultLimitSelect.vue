<template>
  <NPopselect
    v-model:value="resultRowsLimit"
    :options="options"
    trigger="click"
    placement="bottom-start"
    scrollable
  >
    <NButton size="small">
      {{ $t("sql-editor.result-limit.n-rows", { n: resultRowsLimit }) }}
    </NButton>
    <template #action>
      <div class="flex items-center justify-between gap-1">
        <NInputNumber
          v-model:value="resultRowsLimit"
          :show-button="false"
          :min="0"
          :max="100000"
          style="width: 5rem"
          size="small"
        />
        <span>{{ $t("sql-editor.result-limit.rows") }}</span>
      </div>
    </template>
  </NPopselect>
</template>

<script setup lang="ts">
import { NButton, NInputNumber, NPopselect, type SelectOption } from "naive-ui";
import { storeToRefs } from "pinia";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { useSQLEditorStore } from "@/store";

const { t } = useI18n();
const { resultRowsLimit } = storeToRefs(useSQLEditorStore());

const options = computed((): SelectOption[] => {
  return [100, 500, 1000, 5000, 10000, 100000].map((n) => ({
    label: t("sql-editor.result-limit.n-rows", { n }),
    value: n,
  }));
});
</script>
