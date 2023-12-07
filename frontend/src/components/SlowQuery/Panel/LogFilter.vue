<template>
  <div v-if="ready" class="mb-2 space-y-2">
    <div
      class="flex flex-col md:flex-row items-end md:items-center gap-y-2 gap-x-2"
    >
      <AdvancedSearchBox
        v-if="supportOptionIdList.length > 0"
        :params="params"
        :autofocus="false"
        :placeholder="''"
        class="flex-1 hidden md:block"
        :support-option-id-list="supportOptionIdList"
        @update:params="$emit('update:params', $event)"
      />
      <NInputGroup style="width: auto">
        <NDatePicker
          :value="fromTime"
          :disabled="loading"
          :is-date-disabled="isDateDisabled"
          :placeholder="$t('slow-query.filter.from-date')"
          type="date"
          clearable
          format="yyyy-MM-dd z"
          style="width: 12rem"
          @update:value="changeFromTime($event)"
        />
        <NDatePicker
          :value="toTime"
          :disabled="loading"
          :is-date-disabled="isDateDisabled"
          :placeholder="$t('slow-query.filter.to-date')"
          type="date"
          clearable
          format="yyyy-MM-dd z"
          style="width: 12rem"
          @update:value="changeToTime($event)"
        />
      </NInputGroup>

      <div class="flex items-center justify-end space-x-2">
        <slot name="suffix" />
      </div>
    </div>
  </div>
</template>
<script lang="ts" setup>
import dayjs from "dayjs";
import { NDatePicker, NInputGroup } from "naive-ui";
import { useSlowQueryPolicyList } from "@/store";
import { SearchParams, SearchScopeId } from "@/utils";

const props = defineProps<{
  fromTime: number | undefined;
  toTime: number | undefined;
  params: SearchParams;
  supportOptionIdList: SearchScopeId[];
  loading?: boolean;
}>();

const emit = defineEmits<{
  (
    event: "update:time",
    params: { fromTime: number | undefined; toTime: number | undefined }
  ): void;
  (event: "update:params", params: SearchParams): void;
}>();

const { ready } = useSlowQueryPolicyList();

const changeTime = (
  fromTime: number | undefined,
  toTime: number | undefined
) => {
  if (fromTime && toTime && fromTime > toTime) {
    // Swap if from > to
    changeTime(toTime, fromTime);
    return;
  }
  if (fromTime) {
    // fromTime is the start of the day
    fromTime = dayjs(fromTime).startOf("day").valueOf();
  }
  if (toTime) {
    // toTime is the end of the day
    toTime = dayjs(toTime).endOf("day").valueOf();
  }
  emit("update:time", { fromTime, toTime });
};
const changeFromTime = (fromTime: number | undefined) => {
  const { toTime } = props;
  changeTime(fromTime, toTime);
};
const changeToTime = (toTime: number | undefined) => {
  const { fromTime } = props;
  changeTime(fromTime, toTime);
};

const isDateDisabled = (date: number) => {
  return date > dayjs().endOf("day").valueOf();
};
</script>
