<template>
  <div class="w-full flex flex-row gap-4">
    <div class="flex items-center justify-between">
      <h3 class="textlabel">
        {{ $t("common.tasks") }}
        <span>({{ specList.length }})</span>
      </h3>
    </div>
    <div class="flex flex-row gap-2 items-center">
      <div
        class="bg-gray-50 pl-2 p-1 flex flex-row items-center rounded-full gap-1"
      >
        <span class="text-sm mr-1 text-gray-600">{{
          $t("issue.sql-check.sql-checks")
        }}</span>
        <template v-for="status in ADVICE_STATUS_FILTERS" :key="status">
          <NTag
            v-if="getSpecCount(status) > 0"
            :disabled="disabled"
            :size="'small'"
            round
            checkable
            :checked="adviceStatusList.includes(status)"
            @update:checked="
              (checked) => {
                emit(
                  'update:adviceStatusList',
                  checked
                    ? [...adviceStatusList, status]
                    : adviceStatusList.filter((s) => s !== status)
                );
              }
            "
          >
            <template #avatar>
              <AdviceStatusIcon :status="status" />
            </template>
            <span class="select-none">{{ getSpecCount(status) }}</span>
          </NTag>
        </template>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { NTag } from "naive-ui";
import { computed } from "vue";
import { Advice_Status } from "@/types/proto/v1/sql_service";
import { usePlanContext } from "../../logic";
import AdviceStatusIcon from "../SQLCheckSectionV1/AdviceStatusIcon.vue";
import { usePlanSQLCheckContext } from "../SQLCheckSectionV1/context";
import { filterSpec } from "./filter";

defineProps<{
  disabled: boolean;
  adviceStatusList: Advice_Status[];
}>();

const emit = defineEmits<{
  (event: "update:adviceStatusList", adviceStatusList: Advice_Status[]): void;
}>();

const ADVICE_STATUS_FILTERS: Advice_Status[] = [
  Advice_Status.UNRECOGNIZED,
  Advice_Status.SUCCESS,
  Advice_Status.WARNING,
  Advice_Status.ERROR,
];

const planContext = usePlanContext();
const sqlCheckContext = usePlanSQLCheckContext();

const { plan } = planContext;

const specList = computed(() => plan.value.steps.flatMap((step) => step.specs));

const getSpecCount = (adviceStatus?: Advice_Status) => {
  return specList.value.filter((spec) =>
    filterSpec(planContext, sqlCheckContext, spec, { adviceStatus })
  ).length;
};
</script>
