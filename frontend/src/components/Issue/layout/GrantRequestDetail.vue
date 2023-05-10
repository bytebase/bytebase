<!-- TODO(steven): Implement grant request issue detail -->
<template>
  <component :is="logicProviderType" ref="issueLogic">
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
              class="py-6 lg:pl-4 lg:w-72 xl:w-96 lg:border-l lg:border-block-border overflow-hidden"
            ></div>
            <div class="lg:hidden border-t border-block-border" />
            <div class="w-full lg:w-auto lg:flex-1 py-4 pr-4 overflow-x-hidden">
              <GrantRequestForm />
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
import { computed, onMounted, PropType, ref, watchEffect } from "vue";
import { useDialog } from "naive-ui";

import IssueBanner from "../IssueBanner.vue";
import IssueHighlightPanel from "../IssueHighlightPanel.vue";
import IssueDescriptionPanel from "../IssueDescriptionPanel.vue";
import IssueActivityPanel from "../IssueActivityPanel.vue";
import { Issue, IssueCreate, UNKNOWN_ID } from "@/types";
import { defaultTemplate, templateForType } from "@/plugins";
import { useProjectStore, useIssueStore } from "@/store";
import {
  provideIssueLogic,
  StandardModeProvider,
  IssueLogic,
  useBaseIssueLogic,
} from "../logic";
import GrantRequestForm from "../GrantRequestForm.vue";

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

const issueStore = useIssueStore();
const projectStore = useProjectStore();

const create = computed(() => props.create);
const issue = computed(() => props.issue);

const dialog = useDialog();

const { project, createIssue } = useBaseIssueLogic({ issue, create });

const issueLogic = ref<IssueLogic>();

// Determine which type of IssueLogicProvider should be used
const logicProviderType = computed(() => {
  return StandardModeProvider;
});

watchEffect(() => {
  if (props.create) {
    const projectId = (props.issue as IssueCreate).projectId;
    if (projectId !== UNKNOWN_ID) {
      projectStore.fetchProjectById(projectId);
    }
  }
});

const issueTemplate = computed(
  () => templateForType(props.issue.type) || defaultTemplate()
);

onMounted(() => {
  if (create.value) {
    // Set issue store issueStore.isCreatingIssue to false directly.
    issueStore.isCreatingIssue = false;
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
