<template>
  <div
    class="flex items-start gap-x-2 px-3 sm:px-4 py-3 cursor-pointer border-b border-gray-100 hover:bg-gray-50"
    @click="onRowClick($event)"
  >
    <!-- Checkbox -->
    <NCheckbox
      v-if="showSelection"
      class="shrink-0 mt-1"
      :checked="selected"
      @update:checked="$emit('toggle-selection')"
      @click.stop
    />

    <!-- Content + Approval: column on mobile, row on sm+ -->
    <div
      class="flex-1 min-w-0 flex flex-col sm:flex-row sm:items-start sm:gap-x-2"
    >
      <!-- Left: issue content -->
      <div class="flex-1 min-w-0">
        <!-- Line 1: status icon + title + labels -->
        <div class="flex items-center gap-x-1.5">
          <div class="h-6 flex justify-center items-center">
            <IssueStatusIcon class="shrink-0" :issue-status="issue.status" />
          </div>
          <a
            :href="issueUrl"
            class="font-medium text-main text-base truncate hover:underline"
            @click.stop
            v-html="highlightedTitle"
          ></a>
          <span
            v-for="label in labels"
            :key="label.value"
            class="inline-flex items-center gap-x-1 px-1.5 py-0.5 rounded text-xs whitespace-nowrap border shrink-0"
          >
            <span
              class="w-2.5 h-2.5 rounded-sm shrink-0"
              :style="{ backgroundColor: label.color }"
            ></span>
            {{ label.value }}
          </span>
        </div>
        <!-- Line 2: metadata -->
        <div
          class="flex items-center flex-wrap gap-x-1 text-xs text-control-light mt-1"
        >
          <span class="opacity-80">
            #{{ extractIssueUID(issue.name) }}
          </span>
          <span>&middot;</span>
          <HumanizeTs :ts="createTimeTs" />
          <span>&middot;</span>
          <router-link
            :to="{
              name: WORKSPACE_ROUTE_USER_PROFILE,
              params: { principalEmail: creator.email },
            }"
            class="hover:underline"
            @click.stop
          >
            {{ creator.title }}
          </router-link>
          <template v-if="showProject">
            <span>&middot;</span>
            <router-link
              :to="{
                name: PROJECT_V1_ROUTE_DETAIL,
                params: {
                  projectId: extractProjectResourceName(project.name),
                },
              }"
              class="hover:underline"
              @click.stop
            >
              {{ project.title }}
            </router-link>
          </template>
        </div>
        <!-- Expanded description for search highlights -->
        <div
          v-if="expanded"
          class="mt-2 max-h-80 overflow-auto whitespace-pre-wrap wrap-break-word break-all text-sm text-control-light"
          v-html="highlightedDescription"
        ></div>
      </div>

      <!-- Right: approval status tag -->
      <IssueApprovalStatus :issue="issue" />
    </div>
  </div>
</template>

<script lang="ts" setup>
import { NCheckbox } from "naive-ui";
import { computed } from "vue";
import { useRouter } from "vue-router";
import { PROJECT_V1_ROUTE_DETAIL } from "@/router/dashboard/projectV1";
import { WORKSPACE_ROUTE_USER_PROFILE } from "@/router/dashboard/workspaceRoutes";
import { useUserStore } from "@/store";
import { getTimeForPbTimestampProtoEs, unknownUser } from "@/types";
import type { Issue } from "@/types/proto-es/v1/issue_service_pb";
import {
  extractIssueUID,
  extractProjectResourceName,
  getHighlightHTMLByRegExp,
  getIssueRoute,
  projectOfIssue,
} from "@/utils";
import HumanizeTs from "../../misc/HumanizeTs.vue";
import IssueApprovalStatus from "./IssueApprovalStatus.vue";
import { getValidIssueLabels } from "./IssueLabelSelector.vue";
import IssueStatusIcon from "./IssueStatusIcon.vue";

const props = withDefaults(
  defineProps<{
    issue: Issue;
    selected?: boolean;
    showSelection?: boolean;
    showProject?: boolean;
    highlightText?: string;
  }>(),
  {
    selected: false,
    showSelection: true,
    showProject: false,
    highlightText: "",
  }
);

defineEmits<{
  (e: "toggle-selection"): void;
}>();

const router = useRouter();
const userStore = useUserStore();

// Navigation
const issueUrl = computed(() => {
  const issueRoute = getIssueRoute(props.issue);
  return router.resolve({
    name: issueRoute.name,
    params: issueRoute.params,
  }).fullPath;
});

const onRowClick = (e: MouseEvent) => {
  if (e.ctrlKey || e.metaKey) {
    window.open(issueUrl.value, "_blank");
  } else {
    router.push(issueUrl.value);
  }
};

// Metadata
const creator = computed(() => {
  return (
    userStore.getUserByIdentifier(props.issue.creator) ||
    unknownUser(props.issue.creator)
  );
});

const project = computed(() => projectOfIssue(props.issue));

const createTimeTs = computed(() => {
  return Math.floor(
    getTimeForPbTimestampProtoEs(props.issue.createTime, 0) / 1000
  );
});

// Labels
const labels = computed(() => {
  const proj = project.value;
  const validValues = getValidIssueLabels(props.issue.labels, proj.issueLabels);
  return proj.issueLabels.filter((label) => validValues.includes(label.value));
});

// Search highlighting
const highlightWords = computed(() => {
  if (!props.highlightText) return [];
  return props.highlightText.toLowerCase().split(" ");
});

const highlightedTitle = computed(() => {
  return getHighlightHTMLByRegExp(
    props.issue.title,
    highlightWords.value,
    false,
    "bg-yellow-100"
  );
});

const highlightedDescription = computed(() => {
  return getHighlightHTMLByRegExp(
    props.issue.description,
    highlightWords.value,
    false,
    "bg-yellow-100"
  );
});

const expanded = computed(() => {
  if (!props.highlightText || !props.issue.description) return false;
  const desc = props.issue.description.toLowerCase();
  return highlightWords.value.some((word) => desc.includes(word));
});
</script>
