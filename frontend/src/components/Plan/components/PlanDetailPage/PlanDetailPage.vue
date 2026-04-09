<template>
  <div
    class="min-h-0 bg-gray-50 flex flex-col lg:grid"
    :style="desktopLayoutStyle"
  >
    <!-- Left column: timeline phases -->
    <div class="min-w-0">
      <div class="flex min-w-0 flex-col pl-2 pr-4 pb-6 pt-4 xl:pr-8 2xl:pr-12">
        <!-- Changes -->
        <PlanPhaseItem
          :label="$t('plan.navigator.changes')"
          :status="phases[0].status"
          :expanded="isExpanded('changes')"
          collapsible
          :badge="phases[0].badge"
          :line-class="phases[0].lineClass"
          @toggle="togglePhase('changes')"
          @select-phase="expandPhase('changes')"
        >
          <template #icon>
            <CodeIcon class="w-3 h-3 md:w-4 md:h-4 text-white" />
          </template>
          <template #collapsed>
            <p v-if="changesSummary" class="text-sm text-control truncate mt-0.5">
              {{ changesSummary }}
            </p>
          </template>
          <PlanSectionChanges :initial-spec-id="props.specId ?? ''" />
        </PlanPhaseItem>

        <!-- Review -->
        <PlanPhaseItem
          :label="$t('plan.navigator.review')"
          :status="phases[1].status"
          :expanded="isExpanded('review')"
          collapsible
          :badge="phases[1].badge"
          :line-class="phases[1].lineClass"
          @toggle="togglePhase('review')"
          @select-phase="expandPhase('review')"
        >
          <template #icon>
            <MessagesSquareIcon class="w-3 h-3 md:w-4 md:h-4 text-white" />
          </template>
          <template #future>
            <p class="text-sm text-control-placeholder mt-0.5">
              {{ $t("plan.phase.review-description") }}
            </p>
          </template>
          <template #collapsed>
            <p v-if="reviewSummary" class="text-sm text-control truncate mt-0.5">
              {{ reviewSummary }}
            </p>
          </template>
          <PlanSectionReview />
        </PlanPhaseItem>

        <!-- Deploy -->
        <PlanPhaseItem
          :label="$t('plan.navigator.deploy')"
          :status="phases[2].status"
          :expanded="isExpanded('deploy')"
          collapsible
          :badge="phases[2].badge"
          :is-last="true"
          @toggle="togglePhase('deploy')"
          @select-phase="expandPhase('deploy')"
        >
          <template #icon>
            <RocketIcon class="w-3 h-3 md:w-4 md:h-4 text-white" />
          </template>
          <template #future>
            <DeployFutureAction />
          </template>
          <template #collapsed>
            <p v-if="deploySummary" class="text-sm text-control truncate mt-0.5">
              {{ deploySummary }}
            </p>
          </template>
          <PlanSectionDeploy />
        </PlanPhaseItem>
      </div>
    </div>

    <!-- Right column: sidebar (lg+) -->
    <PlanDetailDesktopColumn v-if="showDesktopSidebar" tag="aside">
      <PlanMetadataSidebar class="p-4" />
    </PlanDetailDesktopColumn>

    <!-- Task detail panel: inline (xl+) -->
    <PlanDetailDesktopColumn v-if="showInlineDetailPanel">
      <template #header>
        <div class="flex items-center justify-between border-b px-4 py-2">
          <span class="textinfolabel">{{ $t("common.detail") }}</span>
          <NButton size="tiny" quaternary @click="closeDetailPanel">
            <template #icon>
              <XIcon class="w-4 h-4" />
            </template>
            {{ $t("common.close") }}
          </NButton>
        </div>
      </template>

      <TaskDetailPanel
        v-if="detailPanel"
        :task="detailPanel.task"
        @close="closeDetailPanel"
      />
    </PlanDetailDesktopColumn>

    <!-- Task detail panel: drawer (< xl) -->
    <NDrawer
      v-if="showTaskDrawer"
      :show="!!detailPanel"
      placement="right"
      :width="PLAN_DETAIL_TASK_DRAWER_WIDTH"
      @update:show="(val: boolean) => !val && closeDetailPanel()"
    >
      <NDrawerContent closable @close="closeDetailPanel">
        <TaskDetailPanel
          v-if="detailPanel"
          :task="detailPanel.task"
          @close="closeDetailPanel"
        />
      </NDrawerContent>
    </NDrawer>

    <!-- Sidebar drawer (< lg) -->
    <NDrawer
      v-if="showSidebarDrawer"
      :show="mobileSidebarOpen"
      placement="right"
      :width="PLAN_DETAIL_METADATA_DRAWER_WIDTH"
      @update:show="(val: boolean) => (mobileSidebarOpen = val)"
    >
      <NDrawerContent closable @close="mobileSidebarOpen = false">
        <PlanMetadataSidebar />
      </NDrawerContent>
    </NDrawer>
  </div>
</template>

<script setup lang="ts">
import {
  CodeIcon,
  MessagesSquareIcon,
  RocketIcon,
  XIcon,
} from "lucide-vue-next";
import { NButton, NDrawer, NDrawerContent } from "naive-ui";
import { computed, ref, watch } from "vue";
import { useRoute, useRouter } from "vue-router";
import {
  PROJECT_V1_ROUTE_PLAN_DETAIL_SPEC_DETAIL,
  PROJECT_V1_ROUTE_PLAN_DETAIL_SPECS,
} from "@/router/dashboard/projectV1";
import {
  getRouteQueryString,
  PLAN_DETAIL_PHASE_DEPLOY,
} from "@/router/dashboard/projectV1RouteHelpers";
import type { Task } from "@/types/proto-es/v1/rollout_service_pb";
import { usePlanContext } from "../../logic";
import { useSidebarContext } from "../../logic/sidebar";
import DeployFutureAction from "./DeployFutureAction.vue";
import PlanDetailDesktopColumn from "./PlanDetailDesktopColumn.vue";
import PlanMetadataSidebar from "./PlanMetadataSidebar.vue";
import PlanPhaseItem from "./PlanPhaseItem.vue";
import PlanSectionChanges from "./PlanSectionChanges.vue";
import PlanSectionDeploy from "./PlanSectionDeploy.vue";
import PlanSectionReview from "./PlanSectionReview.vue";
import TaskDetailPanel from "./TaskDetailPanel.vue";
import { type PhaseType, useActivePhase } from "./useActivePhase";
import { usePhaseState } from "./usePhaseState";
import { usePhaseSummaries } from "./usePhaseSummaries";
import {
  PLAN_DETAIL_METADATA_DRAWER_WIDTH,
  PLAN_DETAIL_TASK_DRAWER_WIDTH,
  usePlanDetailDesktopLayout,
} from "./usePlanDetailDesktopLayout";

const props = defineProps<{
  specId?: string;
}>();

const route = useRoute();
const router = useRouter();
const { isCreating, plan, issue, rollout } = usePlanContext();
const {
  mode: sidebarMode,
  containerWidth,
  desktopSidebarWidth,
  mobileSidebarOpen,
} = useSidebarContext();
const { isExpanded, togglePhase, expandPhase, syncExpandedPhases } =
  useActivePhase();

const routeDrivenExpandedPhases = computed<PhaseType[]>(() => {
  if (
    route.name === PROJECT_V1_ROUTE_PLAN_DETAIL_SPECS ||
    route.name === PROJECT_V1_ROUTE_PLAN_DETAIL_SPEC_DETAIL
  ) {
    return ["changes"];
  }

  const routePhase = getRouteQueryString(route.query.phase);
  const routeStageId = getRouteQueryString(route.query.stageId);
  const routeTaskId = getRouteQueryString(route.query.taskId);
  if (routePhase === PLAN_DETAIL_PHASE_DEPLOY || routeStageId || routeTaskId) {
    return ["deploy"];
  }

  return [];
});

watch(
  [routeDrivenExpandedPhases] as const,
  ([phases]) => {
    syncExpandedPhases(phases);
  },
  { immediate: true }
);

const { phases } = usePhaseState(isCreating, issue, rollout);
const { changesSummary, reviewSummary, deploySummary } = usePhaseSummaries(
  plan,
  issue,
  rollout
);

// Task detail panel — driven by ?taskId query param
const detailPanel = ref<{ task: Task } | null>(null);
const hasDetailPanel = computed(() => !!detailPanel.value);
const {
  desktopLayoutStyle,
  showDesktopSidebar,
  showInlineDetailPanel,
  showTaskDrawer,
  showSidebarDrawer,
} = usePlanDetailDesktopLayout({
  sidebarMode,
  containerWidth,
  desktopSidebarWidth,
  hasDetailPanel,
});

// Reactively open/close panel based on query params
watch(
  [() => getRouteQueryString(route.query.taskId), rollout] as const,
  ([taskId, r]) => {
    if (!taskId) {
      detailPanel.value = null;
      return;
    }
    if (!taskId || !r) return;
    for (const stage of r.stages) {
      const task = stage.tasks.find((t) => t.name.endsWith(`/${taskId}`));
      if (task) {
        detailPanel.value = { task };
        return;
      }
    }
    detailPanel.value = null;
  },
  { immediate: true }
);

const closeDetailPanel = () => {
  detailPanel.value = null;
  const routePhase = getRouteQueryString(route.query.phase);
  const stageId = getRouteQueryString(route.query.stageId);
  if (route.query.taskId) {
    router.replace({
      query: {
        ...(routePhase ? { phase: routePhase } : {}),
        ...(stageId ? { stageId } : {}),
      },
    });
  }
};
</script>
