<template>
  <component :is="GrantRequestModeProvider" ref="issueLogic">
    <div
      id="issue-detail-top"
      class="flex-1 overflow-auto focus:outline-none"
      tabindex="0"
    >
      <IssueBanner v-if="!create" />

      <!-- Highlight Panel -->
      <div class="bg-white px-4 pb-4">
        <IssueHighlightPanel />
      </div>

      <!-- Main Content -->
      <main
        class="flex-1 relative overflow-y-auto focus:outline-none lg:border-t lg:border-block-border"
        tabindex="-1"
      >
        <div class="flex max-w-3xl mx-auto px-6 lg:max-w-full">
          <div
            class="flex flex-col flex-1 lg:flex-row-reverse lg:col-span-2 overflow-x-hidden"
          >
            <div
              v-if="!create"
              class="py-6 lg:pl-4 lg:w-72 xl:w-96 lg:border-l lg:border-block-border overflow-hidden"
            >
              <GrantRequestIssueSidebar />
            </div>
            <div class="lg:hidden border-t border-block-border" />
            <div class="w-full lg:w-auto lg:flex-1 py-4 pr-4 overflow-x-hidden">
              <GrantRequestExporterForm v-if="requestRole === 'EXPORTER'" />
              <GrantRequestQuerierForm v-if="requestRole === 'QUERIER'" />
              <IssueDescriptionPanel />
              <section
                v-if="!create"
                aria-labelledby="activity-title"
                class="mt-4"
              >
                <IssueActivityPanel />
              </section>
            </div>
          </div>
        </div>
      </main>
    </div>
  </component>
</template>

<script lang="ts" setup>
import { useDialog } from "naive-ui";
import { computed, onMounted, PropType, ref, watchEffect, watch } from "vue";
import { defaultTemplate, templateForType } from "@/plugins";
import { useIssueReviewContext } from "@/plugins/issue/logic/review/context";
import { useProjectIamPolicyStore, useProjectV1Store } from "@/store";
import {
  GrantRequestContext,
  GrantRequestPayload,
  Issue,
  IssueCreate,
  UNKNOWN_ID,
} from "@/types";
import GrantRequestIssueSidebar from "../GrantRequestIssueSidebar.vue";
import IssueActivityPanel from "../IssueActivityPanel.vue";
import IssueBanner from "../IssueBanner.vue";
import IssueDescriptionPanel from "../IssueDescriptionPanel.vue";
import IssueHighlightPanel from "../IssueHighlightPanel.vue";
import GrantRequestExporterForm from "../form/GrantRequestExporterForm.vue";
import GrantRequestQuerierForm from "../form/GrantRequestQuerierForm.vue";
import {
  provideIssueLogic,
  IssueLogic,
  GrantRequestModeProvider,
} from "../logic";
import { useGrantRequestIssueLogic } from "../logic/grantRequest";

const props = defineProps({
  create: {
    type: Boolean,
    required: true,
  },
  issue: {
    type: Object as PropType<Issue | IssueCreate>,
    required: true,
  },
});

const emit = defineEmits<{
  (e: "status-changed", eager: boolean): void;
}>();

const projectV1Store = useProjectV1Store();

const create = computed(() => props.create);
const issue = computed(() => props.issue);

const dialog = useDialog();
const { project, createIssue } = useGrantRequestIssueLogic({ issue, create });
const reviewContext = useIssueReviewContext();
const issueLogic = ref<IssueLogic>();

watchEffect(() => {
  if (props.create) {
    const projectId = String((props.issue as IssueCreate).projectId);
    if (projectId !== String(UNKNOWN_ID)) {
      projectV1Store.getOrFetchProjectByUID(projectId);
    }
  }
});

const issueTemplate = computed(
  () => templateForType(props.issue.type) || defaultTemplate()
);

const requestRole = computed(() => {
  if (create.value) {
    return ((issue.value as IssueCreate).createContext as GrantRequestContext)
      .role;
  } else {
    const payload = ((issue.value as Issue).payload as any)
      .grantRequest as GrantRequestPayload;
    return payload.role.replace(/^roles\//, "");
  }
});

onMounted(() => {
  if (requestRole.value !== "EXPORTER" && requestRole.value !== "QUERIER") {
    console.error("Invalid request role", requestRole.value);
    return;
  }
  // Always scroll to top, the scrollBehavior doesn't seem to work.
  // The hypothesis is that because the scroll bar is in the nested
  // route, thus setting the scrollBehavior in the global router
  // won't work.
  // BUT when we have a location.hash #activity(\d+) we won't scroll to the top,
  // since #activity(\d+) is used as an activity anchor
  if (!location.hash.match(/^#activity(\d+)/)) {
    document.getElementById("issue-detail-top")!.scrollIntoView();
  }
});

const onStatusChanged = (eager: boolean) => emit("status-changed", eager);

watch(
  () => reviewContext.done.value,
  () => {
    // After the grant request issue's review is done, we need to fetch the latest project IAM policy.
    if (reviewContext.done.value) {
      useProjectIamPolicyStore().fetchProjectIamPolicy(
        project.value.name,
        true /* Skip cache */
      );
    }
  }
);

provideIssueLogic(
  {
    create,
    issue,
    project,
    template: issueTemplate,
    onStatusChanged,
    createIssue,
    dialog,
  },
  true
);
</script>
