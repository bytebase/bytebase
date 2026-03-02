<template>
  <div
    v-if="title"
    class="text-left pl-4 pt-4 pb-2 text-base leading-6 font-medium text-gray-900"
  >
    {{ title }}
  </div>
  <div ref="listRef">
    <NSpin :show="loading">
      <div
        v-for="issue in sortedIssueList"
        :key="issue.name"
        class="flex items-start gap-x-2 px-3 sm:px-4 py-3 cursor-pointer border-b hover:bg-gray-50"
        @click="onRowClick(issue, $event)"
      >
        <!-- Checkbox -->
        <NCheckbox
          v-if="showSelection"
          class="shrink-0 mt-1"
          :checked="selectedIssueNames.has(issue.name)"
          @update:checked="toggleSelection(issue)"
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
                <IssueStatusIcon
                  class="shrink-0"
                  :issue-status="issue.status"
                />
              </div>
              <a
                :href="issueUrl(issue)"
                class="font-medium text-main text-base truncate hover:underline"
                @click.stop
                v-html="highlight(issue.title)"
              ></a>
              <span
                v-for="label in validLabels(issue)"
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
              <span>{{
                humanizeTs(
                  getTimeForPbTimestampProtoEs(issue.updateTime, 0) / 1000
                )
              }}</span>
              <span>&middot;</span>
              <router-link
                :to="{
                  name: WORKSPACE_ROUTE_USER_PROFILE,
                  params: {
                    principalEmail: creatorOfIssue(issue).email,
                  },
                }"
                class="hover:underline"
                @click.stop
              >
                {{ creatorOfIssue(issue).title }}
              </router-link>
              <template v-if="showProject">
                <span>&middot;</span>
                <router-link
                  :to="{
                    name: PROJECT_V1_ROUTE_DETAIL,
                    params: {
                      projectId: extractProjectResourceName(
                        projectOfIssue(issue).name
                      ),
                    },
                  }"
                  class="hover:underline"
                  @click.stop
                >
                  {{ projectOfIssue(issue).title }}
                </router-link>
              </template>
            </div>
            <!-- Expanded description for search highlights -->
            <div
              v-if="isIssueExpanded(issue)"
              class="mt-2 max-h-80 overflow-auto whitespace-pre-wrap wrap-break-word break-all text-sm text-control-light"
              v-html="highlight(issue.description)"
            ></div>
          </div>

          <!-- Right: approval status tag -->
          <IssueApprovalStatus :issue="issue" />
        </div>
      </div>
    </NSpin>
  </div>

  <div
    v-if="isListInViewport && selectedIssueList.length > 0"
    class="sticky bottom-0 w-full bg-white flex items-center gap-x-2 px-3 sm:px-4 py-2 border-y"
    :class="bordered && 'border-x'"
  >
    <BatchIssueActionsV1 :issue-list="selectedIssueList" />
  </div>
</template>

<script lang="ts" setup>
import { orderBy } from "lodash-es";
import { NCheckbox, NSpin } from "naive-ui";
import { computed, ref, watch } from "vue";
import { useRouter } from "vue-router";
import { useElementVisibilityInScrollParent } from "@/composables/useElementVisibilityInScrollParent";
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
  humanizeTs,
  projectOfIssue,
} from "@/utils";
import BatchIssueActionsV1 from "./BatchIssueActionsV1.vue";
import IssueApprovalStatus from "./IssueApprovalStatus.vue";
import { getValidIssueLabels } from "./IssueLabelSelector.vue";
import IssueStatusIcon from "./IssueStatusIcon.vue";

const props = withDefaults(
  defineProps<{
    issueList: Issue[];
    bordered?: boolean;
    title?: string;
    highlightText?: string;
    loading?: boolean;
    showProject?: boolean;
    showSelection?: boolean;
  }>(),
  {
    title: "",
    highlightText: "",
    loading: true,
    bordered: false,
    showSelection: true,
  }
);

const router = useRouter();
const userStore = useUserStore();

const listRef = ref<HTMLDivElement>();
const isListInViewport = useElementVisibilityInScrollParent(listRef);
const selectedIssueNames = ref(new Set<string>());

// Sorting: matching issues first when searching, then by ID descending
const sortedIssueList = computed(() => {
  if (!props.highlightText) {
    return props.issueList;
  }
  return orderBy(
    props.issueList,
    [
      (issue) =>
        `${issue.title} ${issue.description}`.includes(props.highlightText)
          ? 1
          : 0,
      (issue) => parseInt(extractIssueUID(issue.name)),
    ],
    ["desc", "desc"]
  );
});

// Selection
const selectedIssueList = computed(() => {
  return props.issueList.filter((issue) =>
    selectedIssueNames.value.has(issue.name)
  );
});

const toggleSelection = (issue: Issue) => {
  if (selectedIssueNames.value.has(issue.name)) {
    selectedIssueNames.value.delete(issue.name);
  } else {
    selectedIssueNames.value.add(issue.name);
  }
};

watch(
  () => props.issueList,
  (list) => {
    const newIssueNames = new Set(list.map((issue) => issue.name));
    for (const name of selectedIssueNames.value) {
      if (!newIssueNames.has(name)) {
        selectedIssueNames.value.delete(name);
      }
    }
  }
);

// Navigation
const issueUrl = (issue: Issue) => {
  const issueRoute = getIssueRoute(issue);
  return router.resolve({
    name: issueRoute.name,
    params: issueRoute.params,
  }).fullPath;
};

const onRowClick = (issue: Issue, e: MouseEvent) => {
  const url = issueUrl(issue);
  if (e.ctrlKey || e.metaKey) {
    window.open(url, "_blank");
  } else {
    router.push(url);
  }
};

// Labels
const validLabels = (issue: Issue) => {
  const project = projectOfIssue(issue);
  const validValues = getValidIssueLabels(issue.labels, project.issueLabels);
  return project.issueLabels.filter((label) =>
    validValues.includes(label.value)
  );
};

// Creator
const creatorOfIssue = (issue: Issue) => {
  return (
    userStore.getUserByIdentifier(issue.creator) || unknownUser(issue.creator)
  );
};

// Search highlighting
const highlightWords = computed(() => {
  if (!props.highlightText) return [];
  return props.highlightText.toLowerCase().split(" ");
});

const highlight = (content: string) => {
  return getHighlightHTMLByRegExp(
    content,
    highlightWords.value,
    false,
    "bg-yellow-100"
  );
};

const isIssueExpanded = (issue: Issue): boolean => {
  if (!props.highlightText || !issue.description) return false;
  const desc = issue.description.toLowerCase();
  return highlightWords.value.some((word) => desc.includes(word));
};
</script>
