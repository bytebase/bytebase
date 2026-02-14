<template>
  <div ref="containerRef" class="relative overflow-x-hidden h-full">
    <template v-if="ready">
      <PollerProvider>
        <div class="h-full flex flex-col">
          <HeaderSection />

          <div class="flex-1 flex">
            <router-view v-slot="{ Component }">
              <keep-alive :max="3">
                <component :is="Component" />
              </keep-alive>
            </router-view>
          </div>
        </div>
      </PollerProvider>
    </template>
    <div v-else class="w-full h-full flex flex-col items-center justify-center">
      <NSpin />
    </div>
  </div>
</template>

<script lang="tsx" setup>
import { NSpin } from "naive-ui";
import { computed, ref, toRef, watch } from "vue";
import { useI18n } from "vue-i18n";
import { onBeforeRouteLeave, onBeforeRouteUpdate } from "vue-router";
import {
  providePlanContext,
  useBasePlanContext,
  useInitializePlan,
} from "@/components/Plan";
import { HeaderSection } from "@/components/Plan/components";
import { provideSidebarContext } from "@/components/Plan/logic/sidebar";
import { useNavigationGuard } from "@/components/Plan/logic/useNavigationGuard";
import PollerProvider from "@/components/Plan/PollerProvider.vue";
import { useProjectByName } from "@/store";
import { projectNamePrefix } from "@/store/modules/v1/common";
import { setDocumentTitle } from "@/utils";

const props = defineProps<{
  projectId: string;
  planId?: string;
  issueId?: string;
}>();

const { t } = useI18n();
const {
  isCreating,
  plan,
  planCheckRuns,
  issue,
  rollout,
  taskRuns,
  isInitializing,
} = useInitializePlan(
  toRef(props, "projectId"),
  toRef(props, "planId"),
  toRef(props, "issueId")
);
const planBaseContext = useBasePlanContext({
  isCreating,
  plan,
  issue,
});
const containerRef = ref<HTMLElement>();

const ready = computed(() => {
  // Ready when we have either an issue or a valid plan, and initialization is complete
  return (!!issue.value || !!plan.value) && !isInitializing.value;
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

watch(
  [
    isCreating,
    ready,
    () => issue.value?.title,
    () => plan.value?.title,
    () => project.value.title,
  ],
  () => {
    if (isCreating.value) {
      setDocumentTitle(t("plan.new-plan"), project.value.title);
    } else if (ready.value) {
      const entityTitle = issue.value?.title || plan.value?.title;
      if (entityTitle) {
        setDocumentTitle(entityTitle, project.value.title);
      }
    }
  },
  { immediate: true }
);

// Set up navigation guards to check for unsaved changes
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
