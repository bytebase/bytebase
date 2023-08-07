<template>
  <BBModal
    :title="$t('task.check-result.title', { name: selectedTask.title })"
    class="!w-[56rem]"
    header-class="whitespace-pre-wrap break-all gap-x-1"
    @close="$emit('close')"
  >
    <div class="space-y-4">
      <PlanCheckBadgeBar
        :plan-check-run-list="planCheckRunList"
        :selected-type="selectedType"
      />
      <!-- <div>
        <TaskCheckBadgeBar
          :task-check-run-list="task.taskCheckRunList"
          :allow-selection="true"
          :sticky-selection="true"
          :selected-task-check-type="state.selectedTaskCheckType"
          @select-task-check-type="viewCheckRunDetail"
        />
      </div>
      <BBTabFilter
        class="pt-4"
        :tab-item-list="tabItemList"
        :selected-index="state.selectedTabIndex"
        @select-index="
        (index: number) => {
          state.selectedTabIndex = index;
        }
      "
      />
      <TaskCheckRunPanel
        v-if="selectedTaskCheckRun"
        :task-check-run="selectedTaskCheckRun"
        :task="task"
      />
      <div class="pt-4 flex justify-end">
        <button
          type="button"
          class="btn-primary py-2 px-4"
          @click.prevent="$emit('close')"
        >
          {{ $t("common.close") }}
        </button>
      </div> -->
      <TabFilter
        v-if="selectedPlanCheckRunUID"
        v-model:value="selectedPlanCheckRunUID"
        :items="tabItemList"
      />

      <PlanCheckDetail
        v-if="selectedPlanCheckRun"
        :plan-check-run="selectedPlanCheckRun"
        :task="selectedTask"
      />
    </div>
  </BBModal>
</template>

<script setup lang="ts">
import { computed, ref, watch } from "vue";
import { first, orderBy } from "lodash-es";
import { useI18n } from "vue-i18n";

import {
  PlanCheckRun,
  PlanCheckRun_Type,
} from "@/types/proto/v1/rollout_service";
import { TabFilter, TabFilterItem } from "@/components/v2";
import { useIssueContext } from "@/components/IssueV1/logic";
import PlanCheckBadgeBar from "./PlanCheckBadgeBar.vue";
import PlanCheckDetail from "./PlanCheckDetail.vue";

const props = defineProps<{
  planCheckRunList: PlanCheckRun[];
  selectedType: PlanCheckRun_Type;
}>();

defineEmits<{
  (event: "close"): void;
}>();

const { t } = useI18n();
const { selectedTask } = useIssueContext();

const selectedPlanCheckRunList = computed(() => {
  return orderBy(
    props.planCheckRunList.filter(
      (checkRun) => checkRun.type === props.selectedType
    ),
    (checkRun) => parseInt(checkRun.uid, 10),
    "desc"
  );
});

const selectedPlanCheckRunUID = ref(first(selectedPlanCheckRunList.value)?.uid);

const selectedPlanCheckRun = computed(() => {
  const uid = selectedPlanCheckRunUID.value;
  if (!uid) return undefined;
  return selectedPlanCheckRunList.value.find(
    (checkRun) => checkRun.uid === uid
  );
});

const tabItemList = computed(() => {
  return selectedPlanCheckRunList.value.map<TabFilterItem<string>>(
    (checkRun, i) => {
      const label = i === 0 ? t("common.latest") : checkRun.uid;
      return {
        label,
        value: checkRun.uid,
      };
    }
  );
});

watch(selectedPlanCheckRunList, (list) => {
  selectedPlanCheckRunUID.value = first(list)?.uid;
});
</script>
