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
          :max="rowCountOptions[rowCountOptions.length - 1]"
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

const props = defineProps<{
  maximum: number;
}>();

const { t } = useI18n();
const { resultRowsLimit } = storeToRefs(useSQLEditorStore());

const rowCountOptions = computed(() => {
  const list = [100, 500, 1000, 5000, 10000, 100000].filter(
    (num) => num <= props.maximum
  );
  if (props.maximum !== Number.MAX_VALUE && !list.includes(props.maximum)) {
    list.push(props.maximum);
  }
  return list;
});

const options = computed((): SelectOption[] => {
  return rowCountOptions.value.map((n) => ({
    label: t("sql-editor.result-limit.n-rows", { n }),
    value: n,
  }));
});
</script>
