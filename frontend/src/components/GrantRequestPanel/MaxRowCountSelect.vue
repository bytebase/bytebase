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
import { NButton, NInputNumber, NPopselect, type SelectOption } from "naive-ui";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { usePolicyV1Store } from "@/store";
import { minmax } from "@/utils";

defineProps<{
  value?: number;
}>();

const emit = defineEmits<{
  (event: "update:value", value: number): void;
}>();

const { t } = useI18n();
const policyStore = usePolicyV1Store();

const rowCountOptions = computed(() => {
  const list = [1, 100, 500, 1000, 5000, 10000, 100000].filter(
    (num) => num <= policyStore.maximumResultRows
  );
  if (
    policyStore.maximumResultRows !== Number.MAX_VALUE &&
    !list.includes(policyStore.maximumResultRows)
  ) {
    list.push(policyStore.maximumResultRows);
  }
  return list;
});

const options = computed((): SelectOption[] => {
  return rowCountOptions.value.map((n) => ({
    label: t("common.rows.n-rows", { n }),
    value: n,
  }));
});

const handleInput = (value: number | null) => {
  const normalizedValue = minmax(
    value ?? 0,
    first(rowCountOptions.value)!,
    last(rowCountOptions.value)!
  );
  emit("update:value", normalizedValue);
};
</script>
