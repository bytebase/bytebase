<template>
  <NDatePicker
    v-if="isMdOrAbove"
    :key="timeRange ? 'filled' : 'empty'"
    :value="timeRange"
    :is-date-disabled="isDateDisabled"
    type="datetimerange"
    clearable
    class="time-range-picker min-w-0"
    @update:value="emitUpdate"
  />
  <NPopover
    v-else
    trigger="click"
    placement="bottom-end"
    :show="showPopover"
    @update:show="showPopover = $event"
  >
    <template #trigger>
      <NButton quaternary size="small" :type="timeRange ? 'primary' : 'default'">
        <template #icon>
          <CalendarIcon class="w-4 h-4" />
        </template>
      </NButton>
    </template>
    <NDatePicker
      :key="timeRange ? 'filled-sm' : 'empty-sm'"
      :value="timeRange"
      :is-date-disabled="isDateDisabled"
      type="datetimerange"
      clearable
      panel
      @update:value="handleSmallScreenUpdate"
    />
  </NPopover>
</template>

<script setup lang="ts">
import dayjs from "dayjs";
import { CalendarIcon } from "lucide-vue-next";
import { NButton, NDatePicker, NPopover } from "naive-ui";
import { computed, ref } from "vue";
import { useWideScreen } from "@/composables/useWideScreen";
import type { SearchParams } from "@/utils";
import { getTsRangeFromSearchParams, upsertScope } from "@/utils";

const props = withDefaults(
  defineProps<{
    params: SearchParams;
    scope?: "created" | "updated";
  }>(),
  {
    scope: "created",
  }
);

const emit = defineEmits<{
  (event: "update:params", params: SearchParams): void;
}>();

const isMdOrAbove = useWideScreen();
const showPopover = ref(false);

const timeRange = computed(() => {
  return getTsRangeFromSearchParams(props.params, props.scope);
});

const isDateDisabled = (ts: number) => {
  return ts > dayjs().add(1, "day").endOf("day").valueOf();
};

const emitUpdate = (values: [number, number] | null) => {
  const from = values ? dayjs(values[0]).startOf("day") : null;
  const to = values ? dayjs(values[1]).endOf("day") : null;
  const updated = upsertScope({
    params: props.params,
    scopes: {
      id: props.scope,
      value: from && to ? `${from.valueOf()},${to.valueOf()}` : "",
    },
  });
  emit("update:params", updated);
};

const handleSmallScreenUpdate = (values: [number, number] | null) => {
  emitUpdate(values);
  showPopover.value = false;
};
</script>

<style lang="postcss" scoped>
.time-range-picker :deep(.n-input) {
  --n-padding-left: 6px !important;
  --n-padding-right: 6px !important;
}
</style>
