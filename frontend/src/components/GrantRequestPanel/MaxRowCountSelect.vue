<template>
  <NPopselect
    :value="value"
    :options="options"
    trigger="click"
    placement="bottom-start"
    scrollable
    @update:value="handleUpdateValue"
  >
    <NButton
      icon-placement="right"
      :quaternary="quaternary"
      class="bb-overlay-stack-ignore-esc"
      style="justify-content: start; --n-padding: 0 8px;"
    >
      {{ $t("sql-editor.result-limit.self") }}
      {{ $t("common.rows.n-rows", { n: value }) }}
      <template #icon>
        <ChevronRight />
      </template>
    </NButton>
    <template #action>
      <div class="flex items-center justify-between gap-1">
        <NInputNumber
          :value="value"
          :show-button="false"
          :min="0"
          :max="rowCountOptions[rowCountOptions.length - 1]"
          style="width: 5rem"
          @update:value="handleInput"
        />
        <span>{{ $t("common.rows.self") }}</span>
      </div>
    </template>
  </NPopselect>
</template>

<script setup lang="ts">
import { first, last } from "lodash-es";
import { ChevronRight } from "lucide-vue-next";
import { NButton, NInputNumber, NPopselect, type SelectOption } from "naive-ui";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { minmax } from "@/utils";

const props = defineProps<{
  value: number;
  quaternary?: boolean;
  maximumExportCount: number;
}>();

const emit = defineEmits<{
  (event: "update:value", value: number): void;
}>();

const { t } = useI18n();

const rowCountOptions = computed(() => {
  const list = [1, 100, 500, 1000, 5000, 10000, 100000].filter(
    (num) => num <= props.maximumExportCount
  );
  if (
    props.maximumExportCount !== Number.MAX_VALUE &&
    !list.includes(props.maximumExportCount)
  ) {
    list.push(props.maximumExportCount);
  }
  return list;
});

const options = computed((): SelectOption[] => {
  return rowCountOptions.value.map((n) => ({
    label: t("common.rows.n-rows", { n }),
    value: n,
  }));
});

const handleUpdateValue = (value: number) => {
  emit("update:value", value);
};

const handleInput = (value: number | null) => {
  const normalizedValue = minmax(
    value ?? 0,
    first(rowCountOptions.value)!,
    last(rowCountOptions.value)!
  );
  emit("update:value", normalizedValue);
};

defineExpose({
  maximum: computed(() => rowCountOptions.value.slice(-1)[0]),
});
</script>
