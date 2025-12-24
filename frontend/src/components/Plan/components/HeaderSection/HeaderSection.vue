<template>
  <div class="py-2 px-2 sm:px-4">
    <div class="flex flex-row items-center justify-between gap-2">
      <NTag v-if="showDraftTag" round>
        <template #icon>
          <CircleDotDashedIcon class="w-4 h-4" />
        </template>
        {{ $t("common.draft") }}
      </NTag>
      <NTag v-else-if="showClosedTag" round type="default">
        <template #icon>
          <BanIcon class="w-4 h-4" />
        </template>
        {{ $t("common.closed") }}
      </NTag>
      <NTag v-else-if="showDoneTag" round type="success">
        <template #icon>
          <CheckCircle2Icon class="w-4 h-4" />
        </template>
        {{ $t("common.done") }}
      </NTag>
      <TitleInput />
      <div class="flex flex-row items-center justify-end">
        <Actions />
        <!-- Mobile hamburger menu -->
        <NButton
          v-if="hasSidebar && sidebarMode === 'MOBILE'"
          class="px-1!"
          quaternary
          size="medium"
          @click="handleMobileSidebarOpen"
        >
          <MenuIcon class="w-5 h-5" />
        </NButton>
      </div>
    </div>
    <DescriptionSection v-if="showDescriptionSection" />
  </div>
</template>

<script lang="ts" setup>
import {
  BanIcon,
  CheckCircle2Icon,
  CircleDotDashedIcon,
  MenuIcon,
} from "lucide-vue-next";
import { NButton, NTag } from "naive-ui";
import { computed } from "vue";
import { useRoute } from "vue-router";
import { PROJECT_V1_ROUTE_ISSUE_DETAIL_V1 } from "@/router/dashboard/projectV1";
import { State } from "@/types/proto-es/v1/common_pb";
import { IssueStatus } from "@/types/proto-es/v1/issue_service_pb";
import { isValidPlanName } from "@/utils";
import { usePlanContext } from "../../logic";
import { useSidebarContext } from "../../logic/sidebar";
import Actions from "./Actions";
import DescriptionSection from "./DescriptionSection.vue";
import TitleInput from "./TitleInput.vue";

const route = useRoute();
const { isCreating, plan, issue } = usePlanContext();

const hasSidebar = computed(() => {
  // Check if we're in a layout with sidebar (Issue overview tab only)
  // The sidebar only exists when there's an issue AND we're on the overview tab
  if (!plan.value.issue) return false;

  // Check current route to see if we're on the overview tab
  // This would be the ISSUE_DETAIL_V1 route, not the PLAN_DETAIL or ROLLOUT routes
  return route.name === PROJECT_V1_ROUTE_ISSUE_DETAIL_V1;
});

const { mode: sidebarMode, mobileSidebarOpen } = useSidebarContext();

const handleMobileSidebarOpen = () => {
  mobileSidebarOpen.value = true;
};

const showDraftTag = computed(() => {
  return (
    !isCreating.value &&
    isValidPlanName(plan.value.name) &&
    !plan.value.issue &&
    !plan.value.rollout
  );
});

const showClosedTag = computed(() => {
  return (
    plan.value.state === State.DELETED ||
    issue.value?.status === IssueStatus.CANCELED
  );
});

const showDoneTag = computed(() => {
  return issue.value?.status === IssueStatus.DONE;
});

const showDescriptionSection = computed(() => {
  // Only show when there's no issue yet (draft plan)
  return (
    (isValidPlanName(plan.value.name) || isCreating.value) && !plan.value.issue
  );
});
</script>
