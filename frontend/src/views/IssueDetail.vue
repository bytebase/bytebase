<template>
  <IssueDetailLayout
    v-if="issue"
    :issue="issue"
    :create="create"
    @status-changed="onStatusChanged"
  />
  <div v-else class="w-full h-full flex justify-center items-center">
    <NSpin />
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
  POST_CHANGE_POLL_INTERVAL,
  Project,
  unknown,
  UNKNOWN_ID,
} from "@/types";
import { hasFeature, useProjectStore } from "@/store";
import { useInitializeIssue, usePollIssue } from "@/plugins/issue/logic";

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

const state = reactive<LocalState>({
  showFeatureModal: false,
});

const issueSlug = computed(() => props.issueSlug);

const { create, issue } = useInitializeIssue(issueSlug);

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
  pollIssue(eager ? POST_CHANGE_POLL_INTERVAL : NORMAL_POLL_INTERVAL);
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
</script>
