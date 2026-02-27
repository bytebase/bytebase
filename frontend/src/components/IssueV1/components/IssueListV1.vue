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
          :checked="state.selectedIssueNameList.has(issue.name)"
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
                class="font-medium text-main text-base truncate"
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
              <span>{{ creatorName(issue) }}</span>
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
          <NTag
            v-if="issue.approvalStatus === Issue_ApprovalStatus.CHECKING"
            size="small"
            round
            class="shrink-0 mt-0.5"
          >
            {{ t("custom-approval.issue-review.generating-approval-flow") }}
          </NTag>
          <NTag
            v-else-if="
              issue.approvalStatus === Issue_ApprovalStatus.APPROVED
            "
            size="small"
            round
            type="success"
            class="shrink-0 mt-0.5"
          >
            {{ t("issue.table.approved") }}
          </NTag>
          <NTag
            v-else-if="
              issue.approvalStatus === Issue_ApprovalStatus.REJECTED
            "
            size="small"
            round
            type="warning"
            class="shrink-0 mt-0.5"
          >
            {{ t("custom-approval.approval-flow.issue-review.sent-back") }}
          </NTag>
          <div
            v-else-if="
              issue.approvalStatus === Issue_ApprovalStatus.PENDING &&
              totalSteps(issue) > 0
            "
            class="shrink-0 flex flex-row sm:flex-col items-center sm:items-end gap-x-1.5 sm:gap-x-0 mt-0.5"
          >
            <NTag size="small" round>
              {{
                t("issue.table.approval-progress", {
                  approved: issue.approvers.length,
                  total: totalSteps(issue),
                })
              }}
            </NTag>
            <span
              v-if="currentRoleName(issue)"
              class="text-xs text-control-light whitespace-nowrap sm:mt-0.5"
            >
              {{
                t("issue.table.waiting-role", {
                  role: currentRoleName(issue),
                })
              }}
            </span>
          </div>
          <NTag v-else size="small" round class="shrink-0 mt-0.5">
            {{ t("custom-approval.approval-flow.skip") }}
          </NTag>
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
import { NCheckbox, NSpin, NTag } from "naive-ui";
import { computed, reactive, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { useElementVisibilityInScrollParent } from "@/composables/useElementVisibilityInScrollParent";
import { PROJECT_V1_ROUTE_DETAIL } from "@/router/dashboard/projectV1";
import { useUserStore } from "@/store";
import { getTimeForPbTimestampProtoEs, unknownUser } from "@/types";
import type { Issue } from "@/types/proto-es/v1/issue_service_pb";
import { Issue_ApprovalStatus } from "@/types/proto-es/v1/issue_service_pb";
import {
  displayRoleTitle,
  extractIssueUID,
  extractProjectResourceName,
  getHighlightHTMLByRegExp,
  getIssueRoute,
  humanizeTs,
  projectOfIssue,
} from "@/utils";
import BatchIssueActionsV1 from "./BatchIssueActionsV1.vue";
import { getValidIssueLabels } from "./IssueLabelSelector.vue";
import IssueStatusIcon from "./IssueStatusIcon.vue";

interface LocalState {
  selectedIssueNameList: Set<string>;
}

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

const { t } = useI18n();
const router = useRouter();
const userStore = useUserStore();
const state = reactive<LocalState>({
  selectedIssueNameList: new Set(),
});

const listRef = ref<HTMLDivElement>();
const isListInViewport = useElementVisibilityInScrollParent(listRef);

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
    state.selectedIssueNameList.has(issue.name)
  );
});

const toggleSelection = (issue: Issue) => {
  if (state.selectedIssueNameList.has(issue.name)) {
    state.selectedIssueNameList.delete(issue.name);
  } else {
    state.selectedIssueNameList.add(issue.name);
  }
};

watch(
  () => props.issueList,
  (list) => {
    const oldIssueNames = Array.from(state.selectedIssueNameList.values());
    const newIssueNames = new Set(list.map((issue) => issue.name));
    oldIssueNames.forEach((name) => {
      if (!newIssueNames.has(name)) {
        state.selectedIssueNameList.delete(name);
      }
    });
  }
);

// Navigation
const issueUrl = (issue: Issue) => {
  const issueRoute = getIssueRoute(issue);
  const route = router.resolve({
    name: issueRoute.name,
    params: issueRoute.params,
  });
  return route.fullPath;
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

// Creator display
const creatorName = (issue: Issue) => {
  const creator =
    userStore.getUserByIdentifier(issue.creator) || unknownUser(issue.creator);
  return creator.title;
};

// Approval progress helpers
const totalSteps = (issue: Issue): number => {
  return issue.approvalTemplate?.flow?.roles?.length ?? 0;
};

const currentRoleName = (issue: Issue): string => {
  const currentRoleIndex = issue.approvers.length;
  const role = issue.approvalTemplate?.flow?.roles?.[currentRoleIndex];
  if (!role) return "";
  return displayRoleTitle(role);
};

// Search highlighting
interface IssueNameSection {
  text: string;
  highlight: boolean;
}

const highlights = computed(() => {
  if (!props.highlightText) {
    return [];
  }
  return props.highlightText.toLowerCase().split(" ");
});

const highlight = (content: string) => {
  return getHighlightHTMLByRegExp(
    content,
    highlights.value,
    false,
    "bg-yellow-100"
  );
};

const issueHighlightSections = (
  text: string,
  highlights: string[]
): IssueNameSection[] => {
  if (!text) {
    return [];
  }
  if (highlights.length === 0) {
    return [{ text, highlight: false }];
  }

  for (let i = 0; i < highlights.length; i++) {
    const highlight = highlights[i];
    const sections = text.toLowerCase().split(highlight);
    if (sections.length === 0) {
      continue;
    }

    const resp: IssueNameSection[] = [];
    let pos = 0;
    const nextHighlights = [
      ...highlights.slice(0, i),
      ...highlights.slice(i + 1),
    ];
    for (const section of sections) {
      if (section.length) {
        resp.push(
          ...issueHighlightSections(
            text.slice(pos, pos + section.length),
            nextHighlights
          )
        );
        pos += section.length;
      }
      if (i < sections.length - 1) {
        const t = text.slice(pos, pos + highlight.length);
        if (t) {
          resp.push({ text: t, highlight: true });
        }
        pos += highlight.length;
      }
    }
    return resp;
  }

  return [{ text, highlight: false }];
};

const isIssueExpanded = (issue: Issue): boolean => {
  if (!props.highlightText || !issue.description) {
    return false;
  }
  const sections = issueHighlightSections(issue.description, highlights.value);
  return sections.some((item) => item.highlight);
};
</script>
