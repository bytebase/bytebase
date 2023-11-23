<template>
  <NDatePicker
    :key="timeRange ? 'filled' : 'empty'"
    :value="timeRange"
    :is-date-disabled="isDateDisabled"
    type="daterange"
    clearable
    style="width: 14rem"
    class="time-range-picker"
    @update:value="handleUpdate"
  >
  </NDatePicker>
</template>

<script setup lang="ts">
import dayjs from "dayjs";
import { NDatePicker } from "naive-ui";
import { computed } from "vue";
import { SearchParams, getTsRangeFromSearchParams, upsertScope } from "@/utils";

const props = defineProps<{
  params: SearchParams;
}>();

const emit = defineEmits<{
  (event: "update:params", params: SearchParams): void;
}>();

const timeRange = computed(() => {
  return getTsRangeFromSearchParams(props.params, "created");
});

const isDateDisabled = (ts: number) => {
  const today = dayjs().add(1, "day").endOf("day").valueOf();
  if (ts > today) {
    return true;
  }
  return false;
};

const handleUpdate = (values: [number, number] | null) => {
  const updated = upsertScope(props.params, {
    id: "created",
    value: values ? values.join(",") : "",
  });
  emit("update:params", updated);
};
</script>

<style lang="postcss" scoped>
.time-range-picker :deep(.n-input) {
  --n-padding-left: 6px !important;
  --n-padding-right: 6px !important;
}
</style>
