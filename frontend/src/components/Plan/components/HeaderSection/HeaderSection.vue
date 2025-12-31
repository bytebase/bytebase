<template>
  <div class="py-2 px-2 sm:px-4">
    <div class="flex flex-row items-center justify-between gap-2">
      <NTag v-if="showClosedTag" round type="default">
        <template #icon>
          <BanIcon class="w-4 h-4" />
        </template>
        {{ $t("common.closed") }}
      </NTag>
      <NTag v-else-if="showDraftTag" round>
        <template #icon>
          <CircleDotDashedIcon class="w-4 h-4" />
        </template>
        {{ $t("common.draft") }}
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
import {
  PROJECT_V1_ROUTE_ISSUE_DETAIL_V1,
  PROJECT_V1_ROUTE_PLAN_DETAIL_SPEC_DETAIL,
  PROJECT_V1_ROUTE_PLAN_DETAIL_SPECS,
} from "@/router/dashboard/projectV1";
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

const isPlanDetailPage = computed(() => {
  return (
    route.name === PROJECT_V1_ROUTE_PLAN_DETAIL_SPECS ||
    route.name === PROJECT_V1_ROUTE_PLAN_DETAIL_SPEC_DETAIL
  );
});

const isIssueDetailPage = computed(() => {
  return route.name === PROJECT_V1_ROUTE_ISSUE_DETAIL_V1;
});

const hasSidebar = computed(() => {
  // Check if we're in a layout with sidebar (Issue overview tab only)
  // The sidebar only exists when there's an issue AND we're on the overview tab
  if (!plan.value.issue) return false;

  return isIssueDetailPage.value;
});

const { mode: sidebarMode, mobileSidebarOpen } = useSidebarContext();

const handleMobileSidebarOpen = () => {
  mobileSidebarOpen.value = true;
};

const showDraftTag = computed(() => {
  // Only show draft tag on plan detail page
  if (!isPlanDetailPage.value) return false;
  return (
    !isCreating.value &&
    isValidPlanName(plan.value.name) &&
    !plan.value.issue &&
    !plan.value.hasRollout
  );
});

const showClosedTag = computed(() => {
  // On plan detail page: show when plan is deleted
  if (isPlanDetailPage.value) {
    return plan.value.state === State.DELETED;
  }
  // On issue detail page: show when issue is canceled
  if (isIssueDetailPage.value) {
    return issue.value?.status === IssueStatus.CANCELED;
  }
  return false;
});

const showDoneTag = computed(() => {
  // Only show done tag on issue detail page
  if (!isIssueDetailPage.value) return false;
  return issue.value?.status === IssueStatus.DONE;
});

const showDescriptionSection = computed(() => {
  // Only show when there's no issue yet (draft plan)
  return (
    (isValidPlanName(plan.value.name) || isCreating.value) && !plan.value.issue
  );
});
</script>
