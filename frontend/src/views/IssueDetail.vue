<template>
  <div class="w-full h-full relative">
    <template v-if="issue">
      <DatabaseRelatedDetail
        v-if="isDatabaseRelatedIssue"
        :issue="issue"
        :create="create"
        @status-changed="onStatusChanged"
      />
      <GrantRequestDetail
        v-if="isGrantRequestIssue"
        :issue="issue"
        :create="create"
        @status-changed="onStatusChanged"
      />
    </template>
    <div
      v-if="showLoading"
      class="w-full h-full fixed md:absolute inset-0 flex justify-center items-center bg-white/50"
    >
      <NSpin />
    </div>
  </div>
  <FeatureModal
    feature="bb.feature.multi-tenancy"
    :open="state.showFeatureModal"
    @cancel="state.showFeatureModal = false"
  />
</template>

<script lang="ts" setup>
import { onMounted, computed, reactive, watch } from "vue";
import { useRoute, _RouteLocationBase } from "vue-router";
import { NSpin } from "naive-ui";
import { useI18n } from "vue-i18n";

import {
  IssueType,
  NORMAL_POLL_INTERVAL,
  MINIMUM_POLL_INTERVAL,
  UNKNOWN_ID,
  Issue,
  unknownProject,
} from "@/types";
import {
  hasFeature,
  useIssueStore,
  useProjectV1Store,
  useUIStateStore,
} from "@/store";
import {
  useInitializeIssue,
  provideIssueReview,
  usePollIssue,
  ReviewEvents,
} from "@/plugins/issue/logic";
import { useTitle } from "@vueuse/core";
import Emittery from "emittery";
import { isDatabaseRelatedIssueType, isGrantRequestIssueType } from "@/utils";
import DatabaseRelatedDetail from "@/components/Issue/layout/DatabaseRelatedDetail.vue";
import GrantRequestDetail from "@/components/Issue/layout/GrantRequestDetail.vue";
import { Project, TenantMode } from "@/types/proto/v1/project_service";

interface LocalState {
  showFeatureModal: boolean;
}

const props = defineProps({
  issueSlug: {
    required: true,
    type: String,
  },
});

const uiStateStore = useUIStateStore();
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

const reviewEvents = new Emittery<ReviewEvents>();

provideIssueReview(
  computed(() => {
    return create.value ? undefined : (issue.value as Issue);
  }),
  reviewEvents
);

const isGrantRequestIssue = computed(() => {
  return !!issue.value && isGrantRequestIssueType(issue.value.type);
});

const isDatabaseRelatedIssue = computed(() => {
  return !!issue.value && isDatabaseRelatedIssueType(issue.value.type);
});

onMounted(() => {
  if (!uiStateStore.getIntroStateByKey("issue.visit")) {
    uiStateStore.saveIntroStateByKey({
      key: "issue.visit",
      newState: true,
    });
  }
});

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
      project.tenantMode === TenantMode.TENANT_MODE_ENABLED &&
      !hasFeature("bb.feature.multi-tenancy")
    ) {
      state.showFeatureModal = true;
    }
  }
});

const onStatusChanged = (eager: boolean) => {
  pollIssue(eager ? MINIMUM_POLL_INTERVAL : NORMAL_POLL_INTERVAL);
  reviewEvents.emit("issue-status-changed", eager);
};

const findProject = async (): Promise<Project> => {
  const projectId = route.query.project
    ? (route.query.project as string)
    : String(UNKNOWN_ID);
  if (projectId !== String(UNKNOWN_ID)) {
    const projectV1Store = useProjectV1Store();
    const project = await projectV1Store.getOrFetchProjectByUID(projectId);
    return project;
  }
  return unknownProject();
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
