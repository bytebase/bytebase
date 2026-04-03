<template>
  <div
    ref="containerRef"
    class="relative bg-gray-50 [overflow-x:clip]"
    :style="viewportVars"
  >
    <template v-if="ready">
      <PollerProvider>
        <div
          class="flex min-h-full flex-col"
          :style="{
            minHeight: `var(${PLAN_DETAIL_LAYOUT_MIN_HEIGHT_VAR})`,
          }"
        >
          <div
            ref="headerRef"
            class="shrink-0 border-b bg-white"
          >
            <PlanDetailHeader />
          </div>

          <PlanDetailPage :spec-id="props.specId" />
        </div>
      </PollerProvider>
    </template>
    <div
      v-else
      class="flex min-h-[16rem] w-full flex-col items-center justify-center"
    >
      <NSpin />
    </div>
  </div>
</template>

<script lang="ts" setup>
import { NSpin } from "naive-ui";
import { computed, ref, toRef, watch } from "vue";
import { useI18n } from "vue-i18n";
import {
  onBeforeRouteLeave,
  onBeforeRouteUpdate,
  useRouter,
} from "vue-router";
import {
  providePlanContext,
  useBasePlanContext,
  useInitializePlan,
} from "@/components/Plan";
import { provideSidebarContext } from "@/components/Plan/logic/sidebar";
import { useNavigationGuard } from "@/components/Plan/logic/useNavigationGuard";
import PollerProvider from "@/components/Plan/PollerProvider.vue";
import { useProjectByName } from "@/store";
import { projectNamePrefix } from "@/store/modules/v1/common";
import { getIssueRoute, setDocumentTitle } from "@/utils";
import PlanDetailHeader from "./PlanDetailHeader.vue";
import PlanDetailPage from "./PlanDetailPage.vue";
import {
  PLAN_DETAIL_LAYOUT_MIN_HEIGHT_VAR,
  usePlanDetailViewportVars,
} from "./usePlanDetailViewportVars";

const props = defineProps<{
  projectId: string;
  planId: string;
  specId?: string;
}>();

const { t } = useI18n();
const router = useRouter();
const {
  isCreating,
  plan,
  planCheckRuns,
  issue,
  rollout,
  taskRuns,
  isInitializing,
} = useInitializePlan(toRef(props, "projectId"), toRef(props, "planId"));

const planBaseContext = useBasePlanContext({ isCreating, plan, issue });
const containerRef = ref<HTMLElement>();
const headerRef = ref<HTMLElement>();
const { viewportVars } = usePlanDetailViewportVars(headerRef);

const ready = computed(() => {
  return !!plan.value && !isInitializing.value;
});

const { confirmNavigation } = useNavigationGuard();

providePlanContext({
  isCreating,
  plan,
  planCheckRuns,
  issue,
  rollout,
  taskRuns,
  ...planBaseContext,
});

provideSidebarContext(containerRef);

const projectName = computed(() => `${projectNamePrefix}${props.projectId}`);
const { project } = useProjectByName(projectName);

const shouldRedirectToIssueDetail = computed(() => {
  if (!issue.value?.name) {
    return false;
  }
  if (plan.value.specs.length === 0) {
    return false;
  }
  return plan.value.specs.every((spec) => {
    return (
      spec.config?.case === "createDatabaseConfig" ||
      spec.config?.case === "exportDataConfig"
    );
  });
});

watch(
  [isCreating, ready, () => plan.value?.title, () => project.value.title],
  () => {
    if (isCreating.value) {
      setDocumentTitle(t("plan.new-plan"), project.value.title);
    } else if (ready.value && plan.value?.title) {
      setDocumentTitle(plan.value.title, project.value.title);
    }
  },
  { immediate: true }
);

watch(
  [ready, shouldRedirectToIssueDetail, () => issue.value?.name],
  ([isReady, shouldRedirect, issueName]) => {
    if (!isReady) return;

    if (issueName && shouldRedirect) {
      void router.replace(getIssueRoute({ name: issueName }));
    }
  },
  { immediate: true }
);

const handleRouteNavigation = async (
  _to: unknown,
  _from: unknown,
  next: (proceed?: boolean) => void
) => {
  const canNavigate = await confirmNavigation();
  next(canNavigate);
};

onBeforeRouteLeave(handleRouteNavigation);
onBeforeRouteUpdate(handleRouteNavigation);
</script>
