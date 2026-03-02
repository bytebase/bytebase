<template>
  <div>
    <NSpin :show="loading">
      <IssueListItem
        v-for="issue in sortedIssueList"
        :key="issue.name"
        :issue="issue"
        :selected="selectedIssueNames.has(issue.name)"
        :show-selection="showSelection"
        :show-project="showProject"
        :highlight-text="highlightText"
        @toggle-selection="toggleSelection(issue)"
      />
    </NSpin>
  </div>

  <div
    v-if="selectedIssueList.length > 0"
    class="sticky bottom-0 w-full bg-white flex items-center gap-x-2 px-3 sm:px-4 py-2 border-y"
  >
    <NCheckbox
      :checked="allSelected"
      :indeterminate="!allSelected"
      @update:checked="toggleSelectAll"
    />
    <span class="text-sm text-control-light">
      {{ selectedIssueList.length }} / {{ issueList.length }}
    </span>
    <BatchIssueActionsV1 :issue-list="selectedIssueList" />
  </div>
</template>

<script lang="ts" setup>
import { orderBy } from "lodash-es";
import { NCheckbox, NSpin } from "naive-ui";
import { computed, ref, watch } from "vue";
import type { Issue } from "@/types/proto-es/v1/issue_service_pb";
import { extractIssueUID } from "@/utils";
import BatchIssueActionsV1 from "./BatchIssueActionsV1.vue";
import IssueListItem from "./IssueListItem.vue";

const props = withDefaults(
  defineProps<{
    issueList: Issue[];
    highlightText?: string;
    loading?: boolean;
    showProject?: boolean;
    showSelection?: boolean;
  }>(),
  {
    highlightText: "",
    loading: true,
    showSelection: true,
  }
);

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

const allSelected = computed(() => {
  return (
    props.issueList.length > 0 &&
    props.issueList.every((issue) => selectedIssueNames.value.has(issue.name))
  );
});

const toggleSelection = (issue: Issue) => {
  if (selectedIssueNames.value.has(issue.name)) {
    selectedIssueNames.value.delete(issue.name);
  } else {
    selectedIssueNames.value.add(issue.name);
  }
};

const toggleSelectAll = () => {
  if (allSelected.value) {
    selectedIssueNames.value.clear();
  } else {
    for (const issue of props.issueList) {
      selectedIssueNames.value.add(issue.name);
    }
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
</script>
