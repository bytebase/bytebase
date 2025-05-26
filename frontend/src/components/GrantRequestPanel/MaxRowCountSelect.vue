<template>
  <NPopselect
    :value="value"
    :options="options"
    trigger="click"
    placement="bottom-start"
    scrollable
    @update:value="$emit('update:value', $event)"
  >
    <NButton class="bb-overlay-stack-ignore-esc">
      {{
        value
          ? $t("common.rows.n-rows", { n: value })
          : $t("issue.grant-request.select-export-rows")
      }}
    </NButton>
    <template #action>
      <div class="flex items-center justify-between gap-1">
        <NInputNumber
          :value="value"
          :show-button="false"
          :min="0"
          :max="100000"
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
import { NButton, NInputNumber, NPopselect, type SelectOption } from "naive-ui";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { minmax } from "@/utils";

defineProps<{
  value?: number;
}>();

const emit = defineEmits<{
  (event: "update:value", value: number): void;
}>();

const { t } = useI18n();
const rowCountOptions = [1, 100, 500, 1000, 5000, 10000, 100000];

const options = computed((): SelectOption[] => {
  return rowCountOptions.map((n) => ({
    label: t("common.rows.n-rows", { n }),
    value: n,
  }));
});

const handleInput = (value: number | null) => {
  const normalizedValue = minmax(
    value ?? 0,
    first(rowCountOptions)!,
    last(rowCountOptions)!
  );
  emit("update:value", normalizedValue);
};
</script>
