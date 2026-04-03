<template>
  <div ref="containerRef" class="relative overflow-x-hidden h-full">
    <template v-if="ready">
      <PollerProvider>
        <div class="h-full flex flex-col">
          <HeaderSection />

          <div class="flex-1 flex border-t border-gray-100">
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
  issueId?: string;
}>();

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
  undefined,
  toRef(props, "issueId")
);
const planBaseContext = useBasePlanContext({
  isCreating,
  plan,
  issue,
});
const containerRef = ref<HTMLElement>();

const ready = computed(() => {
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
    if (ready.value) {
      const entityTitle = issue.value?.title || plan.value?.title;
      if (entityTitle) {
        setDocumentTitle(entityTitle, project.value.title);
      }
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
