<template>
  <ContextMenuButton
    v-if="actionList.length > 0"
    :action-list="actionList"
    :disabled="hasRunningPlanCheck"
    preference-key="issue.task.run-checks"
    default-action-key="RUN-CHECKS"
    @click="handleRunChecks"
  >
    <template #icon>
      <BBSpin v-if="hasRunningPlanCheck" :size="16" />
      <PlayIcon v-else class="w-4 h-4" />
    </template>
    <template #default="{ action }">
      <template v-if="hasRunningPlanCheck">
        {{ $t("task.checking") }}
      </template>
      <template v-else>
        {{ action.text }}
      </template>
    </template>
  </ContextMenuButton>
</template>

<script setup lang="ts">
import { PlayIcon } from "lucide-vue-next";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { BBSpin } from "@/bbkit";
import type { ContextMenuButtonAction } from "@/components/v2";
import { ContextMenuButton } from "@/components/v2";
import {
  PlanCheckRun,
  PlanCheckRun_Status,
} from "@/types/proto/v1/plan_service";

const props = defineProps<{
  planCheckRunList: PlanCheckRun[];
}>();

const emit = defineEmits<{
  (event: "run-checks"): void;
}>();

const { t } = useI18n();

const actionList = computed(() => {
  const actionList: ContextMenuButtonAction[] = [];
  actionList.push({
    key: "RUN-CHECKS",
    text: t("task.run-checks"),
    params: {},
  });
  return actionList;
});

const hasRunningPlanCheck = computed((): boolean => {
  return props.planCheckRunList.some(
    (checkRun) => checkRun.status === PlanCheckRun_Status.RUNNING
  );
});

const handleRunChecks = () => {
  if (hasRunningPlanCheck.value) return;
  emit("run-checks");
};
</script>
