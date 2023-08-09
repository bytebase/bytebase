<template>
  <div
    v-if="payload || isSheetOversize"
    class="w-full p-4 flex items-center bg-yellow-50 gap-x-3"
  >
    <div class="flex-shrink-0">
      <heroicons-solid:information-circle class="h-5 w-5 text-yellow-400" />
    </div>
    <div class="flex-1 text-sm font-medium text-yellow-800">
      <i18n-t v-if="payload" tag="h3" keypath="sheet.from-issue-warning">
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
        <template v-if="isSheetOversize" #oversize>
          {{ $t("sheet.content-oversize-warning") }}
        </template>
      </i18n-t>
      <div v-else-if="isSheetOversize">
        {{ $t("sheet.content-oversize-warning") }}
      </div>
    </div>
    <DownloadSheetButton
      v-if="tab.sheetName"
      :sheet="tab.sheetName"
      size="small"
    />
  </div>
</template>

<script lang="ts" setup>
import { NTooltip } from "naive-ui";
import { shallowRef, computed, ref, watch } from "vue";
import DownloadSheetButton from "@/components/Sheet/DownloadSheetButton.vue";
import { useIssueStore, useSheetV1Store, useTabStore } from "@/store";
import { Issue } from "@/types";
import { getSheetIssueBacktracePayloadV1 } from "@/utils";

const tabStore = useTabStore();
const sheetV1Store = useSheetV1Store();
const issueStore = useIssueStore();
const tab = computed(() => tabStore.currentTab);
const loading = ref(true);
const issue = shallowRef<Issue>();

const sheet = computed(() => {
  const { sheetName } = tab.value;
  if (!sheetName) return undefined;
  return sheetV1Store.getSheetByName(sheetName);
});

const payload = computed(() => {
  if (!sheet.value) return undefined;
  return getSheetIssueBacktracePayloadV1(sheet.value);
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

const isSheetOversize = computed(() => {
  if (!sheet.value) {
    return false;
  }

  return (
    new TextDecoder().decode(sheet.value.content).length <
    sheet.value.contentSize
  );
});
</script>
