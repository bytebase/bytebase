<template>
  <div class="w-full h-full relative">
    <IssueDetailLayout
      v-if="issue"
      :issue="issue"
      :create="create"
      @status-changed="onStatusChanged"
    />
    <div
      v-if="showLoading"
      class="w-full h-full absolute inset-0 flex justify-center items-center bg-white/50"
    >
      <NSpin />
    </div>
  </div>
  <FeatureModal
    v-if="state.showFeatureModal"
    feature="bb.feature.multi-tenancy"
    @cancel="state.showFeatureModal = false"
  />
</template>

<script lang="ts" setup>
import { computed, reactive, watch } from "vue";
import { useRoute, _RouteLocationBase } from "vue-router";
import { NSpin } from "naive-ui";
import { IssueDetailLayout } from "@/components/Issue";
import {
  IssueType,
  NORMAL_POLL_INTERVAL,
  MINIMUM_POLL_INTERVAL,
  Project,
  unknown,
  UNKNOWN_ID,
  Issue,
} from "@/types";
import { hasFeature, useIssueStore, useProjectStore } from "@/store";
import { useInitializeIssue, usePollIssue } from "@/plugins/issue/logic";
import { useTitle } from "@vueuse/core";
import { useI18n } from "vue-i18n";

interface LocalState {
  showFeatureModal: boolean;
}

const props = defineProps({
  issueSlug: {
    required: true,
    type: String,
  },
});

const route = useRoute();
const { t } = useI18n();

const state = reactive<LocalState>({
  showFeatureModal: false,
});
const issueStore = useIssueStore();

const issueSlug = computed(() => props.issueSlug);

const { create, issue } = useInitializeIssue(issueSlug);

const showLoading = computed(() => {
  if (!issue.value) return true;
  return issueStore.isCreatingIssue;
});

const pollIssue = usePollIssue(issueSlug, issue);

watch(issueSlug, async () => {
  if (!create.value) return;
  const type = route.query.template as IssueType;
  const tenantIssueTypes: IssueType[] = [
    "bb.issue.database.schema.update",
    "bb.issue.database.data.update",
  ];
  if (tenantIssueTypes.includes(type)) {
    const project = await findProject();
    if (
      project.tenantMode === "TENANT" &&
      !hasFeature("bb.feature.multi-tenancy")
    ) {
      state.showFeatureModal = true;
    }
  }
});

const onStatusChanged = (eager: boolean) => {
  pollIssue(eager ? MINIMUM_POLL_INTERVAL : NORMAL_POLL_INTERVAL);
};

const findProject = async (): Promise<Project> => {
  const projectId = route.query.project
    ? parseInt(route.query.project as string)
    : UNKNOWN_ID;
  let project = unknown("PROJECT");

  if (projectId !== UNKNOWN_ID) {
    const projectStore = useProjectStore();
    project = await projectStore.fetchProjectById(projectId);
  }

  return project;
};

const documentTitle = computed(() => {
  if (create.value) {
    return t("issue.new-issue");
  } else {
    const issueEntity = issue.value as Issue | undefined;

    if (issueEntity) {
      return issueEntity.name;
    }
    return t("common.loading");
  }
});
useTitle(documentTitle);
</script>
