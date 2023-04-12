<template>
  <div v-if="payload" class="w-full p-4 flex bg-yellow-50">
    <div class="flex-shrink-0">
      <heroicons-solid:information-circle class="h-5 w-5 text-yellow-400" />
    </div>
    <div class="ml-3">
      <i18n-t
        tag="h3"
        keypath="sheet.from-sheet-warning"
        class="text-sm font-medium text-yellow-800"
      >
        <template #issue>
          <NTooltip :disabled="loading || !issue">
            <template #trigger>
              <a
                v-if="issue"
                class="normal-link"
                href="`/issue/${issueSlug(issue.name,  issue.id)}`"
              >
                #{{ payload.issueId }}
              </a>
              <span v-else>#{{ payload.issueId }}</span>
            </template>

            <div class="max-w-[20rem]">
              {{ issue?.name }}
            </div>
          </NTooltip>
        </template>
      </i18n-t>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { useIssueStore, useSheetStore, useTabStore } from "@/store";
import { Issue } from "@/types";
import { getSheetIssueBacktracePayload } from "@/utils";
import { NTooltip } from "naive-ui";
import { shallowRef } from "vue";
import { computed, ref, watch } from "vue";

const tabStore = useTabStore();
const sheetStore = useSheetStore();
const issueStore = useIssueStore();
const tab = computed(() => tabStore.currentTab);
const loading = ref(true);
const issue = shallowRef<Issue>();

const sheet = computed(() => {
  const { sheetId } = tab.value;
  if (!sheetId) return undefined;
  return sheetStore.getSheetById(sheetId);
});

const payload = computed(() => {
  if (!sheet.value) return undefined;
  return getSheetIssueBacktracePayload(sheet.value);
});

watch(
  payload,
  (payload) => {
    loading.value = true;
    const finish = (_issue: Issue | undefined) => {
      loading.value = false;
      issue.value = _issue;
    };

    if (!payload) {
      return finish(undefined);
    }
    const { issueId } = payload;
    issueStore.getOrFetchIssueById(issueId).then((_issue) => {
      if (_issue.id === issueId) {
        return finish(_issue);
      }
    });
  },
  { immediate: true }
);
</script>
