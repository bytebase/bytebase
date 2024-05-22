<template>
  <ContextMenuButton
    v-if="actionList.length > 0"
    :action-list="actionList"
    :disabled="hasRunningPlanCheck"
    preference-key="plan.task.run-checks"
    default-action-key="RUN-CHECKS"
    @click="$emit('run-checks')"
  >
    <template #icon>
      <BBSpin v-if="hasRunningPlanCheck" :size="20" />
      <heroicons-outline:play v-else class="w-4 h-4" />
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
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { planCheckRunListForSpec } from "@/components/Plan/logic";
import { usePlanContext } from "@/components/Plan/logic";
import type { ContextMenuButtonAction } from "@/components/v2";
import { ContextMenuButton } from "@/components/v2";
import { PlanCheckRun_Status } from "@/types/proto/v1/plan_service";

defineEmits<{
  (event: "run-checks"): void;
}>();

const { t } = useI18n();
const { isCreating, plan, selectedSpec } = usePlanContext();

const allowRunCheckForPlan = computed(() => {
  if (isCreating.value) {
    return false;
  }
  return true;
});

const actionList = computed(() => {
  if (!allowRunCheckForPlan.value) return [];

  const actionList: ContextMenuButtonAction[] = [];
  actionList.push({
    key: "RUN-CHECKS",
    text: t("task.run-checks"),
    params: {},
  });
  return actionList;
});

const hasRunningPlanCheck = computed((): boolean => {
  if (isCreating.value) return false;

  const planCheckRunList = planCheckRunListForSpec(
    plan.value,
    selectedSpec.value
  );
  return planCheckRunList.some(
    (checkRun) => checkRun.status === PlanCheckRun_Status.RUNNING
  );
});
</script>
