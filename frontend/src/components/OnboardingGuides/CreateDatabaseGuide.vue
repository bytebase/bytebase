<template>
  <template v-if="guideIndex >= 0 && guideIndex < guideStepList.length">
    <GuideDialog
      v-for="guide in shownGuideList"
      :key="guide.title"
      :title="guide.title"
      :position="guide.position"
      :loading="guide.showLoading || false"
      :target-element-selector="guide.targetElementSelector"
    ></GuideDialog>
  </template>
  <Transition>
    <CreateDatabaseGuideFinished v-if="guideIndex === guideStepList.length" />
  </Transition>
</template>

<script lang="ts" setup>
import { computed, onMounted, ref, watch } from "vue";
import { useRoute, useRouter } from "vue-router";
import { lastTask } from "@/utils";
import {
  useDatabaseStore,
  useInstanceList,
  useIssueStore,
  useProjectStore,
} from "@/store";
import GuideDialog from "@/plugins/demo/components/GuideDialog.vue";
import CreateDatabaseGuideFinished from "./CreateDatabaseGuideFinished.vue";
import { useI18n } from "vue-i18n";

const { t } = useI18n();
const route = useRoute();
const router = useRouter();

type GuideData = {
  title: string;
  description: string;
  position: string;
  targetElementSelector: string;
  route: string;
  expectedUrl: any;
  showLoading?: boolean;
};

const guideStepList = computed(() => {
  return [
    // guides for `add instance`
    [
      {
        title: t("onboarding-guide.create-database-guide.let-add-a-instance"),
        description: "",
        position: "bottom-center",
        targetElementSelector: "[data-label='bb-quick-action-add-instance']",
        route: "/",
        expectedUrl: /.*/,
      },
    ],
    // guides for `add project`
    [
      {
        title: t(
          "onboarding-guide.create-database-guide.go-to-project-list-page"
        ),
        description: "",
        position: "bottom-center",
        targetElementSelector: "[data-label='bb-header-project-button']",
        expectedUrl: /^(?!\/project)/,
      },
      {
        title: t("onboarding-guide.create-database-guide.click-new-project"),
        description: "",
        position: "bottom-center",
        targetElementSelector: "[data-label='bb-quick-action-new-project']",
        expectedUrl: /^\/project/,
      },
    ],
    // guides for `add issue`
    [
      {
        title: t("onboarding-guide.create-database-guide.click-new-database"),
        description: "",
        position: "bottom-center",
        targetElementSelector: "[data-label='bb-quick-action-new-db']",
        expectedUrl: /.*/,
      },
    ],
    // guides for `approve issue`
    [
      {
        title: t("onboarding-guide.create-database-guide.click-approve"),
        description: "",
        position: "bottom-right",
        targetElementSelector:
          "[data-label='bb-issue-status-transition-button']",
        expectedUrl: /.*/,
      },
    ],
    // guides for `loading`
    [
      {
        title: t("onboarding-guide.create-database-guide.wait-issue-finished"),
        description: "",
        position: "center",
        targetElementSelector: "#app",
        expectedUrl: /.*/,
        showLoading: true,
      },
    ],
    // guides for `back to home`
    [
      {
        title: t("onboarding-guide.create-database-guide.back-to-home"),
        description: "",
        position: "bottom",
        targetElementSelector:
          "[data-label='bb-dashboard-sidebar-home-button']",
        expectedUrl: /^\/.+/,
      },
    ],
  ] as GuideData[][];
});

const instanceList = useInstanceList();
const projectList = computed(() => useProjectStore().projectList);
const issueList = computed(() => useIssueStore().issueList);

const shownGuideList = computed(() => {
  const guideList = guideStepList.value[guideIndex.value];

  return guideList.filter((guide) => {
    if (guide.expectedUrl) {
      return route.path.match(guide.expectedUrl);
    }
    return true;
  });
});

const guideIndex = ref(-1);

onMounted(() => {
  if (route.name !== "workspace.home") {
    router.push({
      name: "workspace.home",
    });
  }
});

watch(
  [route, instanceList, projectList, issueList],
  async () => {
    let tempGuideIndex = 0;
    if (instanceList.value.length > 0) {
      tempGuideIndex = 1;
    }
    if (projectList.value.length > 1) {
      tempGuideIndex = 2;
    }
    if (issueList.value.length > 0) {
      const issue = issueList.value[0];
      const task = lastTask(issue.pipeline);
      if (task.status === "PENDING_APPROVAL") {
        tempGuideIndex = 3;
      } else if (task.status === "PENDING" || task.status === "RUNNING") {
        tempGuideIndex = 4;
      } else {
        tempGuideIndex = 5;
      }
    }

    const databaseList = await useDatabaseStore().fetchDatabaseList();
    if (databaseList.length > 0 && route.name === "workspace.home") {
      tempGuideIndex = 6;
    }

    guideIndex.value = tempGuideIndex;
  },
  {
    immediate: true,
  }
);

watch(guideIndex, () => {
  document.body
    .querySelectorAll(".bb-guide-target-element")
    .forEach((el) => el.classList.remove("bb-guide-target-element"));
});
</script>

<style scoped>
.v-enter-active,
.v-leave-active {
  transition: opacity 0.3s ease-in;
}

.v-enter-from,
.v-leave-to {
  opacity: 0;
}
</style>
