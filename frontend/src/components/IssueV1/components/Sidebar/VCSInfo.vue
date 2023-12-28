<template>
  <div
    v-if="pushEvent"
    class="text-sm text-control-light flex flex-col items-start gap-y-1"
  >
    <div class="space-x-1">
      <template v-if="pushEvent.vcsType === VcsType.GITLAB">
        <img class="h-4 w-auto inline-block" src="@/assets/gitlab-logo.svg" />
      </template>
      <template v-else-if="pushEvent.vcsType === VcsType.GITHUB">
        <img class="h-4 w-auto inline-block" src="@/assets/github-logo.svg" />
      </template>
      <template v-else-if="pushEvent.vcsType === VcsType.BITBUCKET">
        <img
          class="h-4 w-auto inline-block"
          src="@/assets/bitbucket-logo.svg"
        />
      </template>
      <a :href="vcsBranchUrl" target="_blank" class="normal-link">{{
        `${vcsBranch}@${pushEvent.repositoryFullPath}`
      }}</a>
    </div>

    <i18n-t
      v-if="commit && commit.id && commit.url"
      keypath="issue.commit-by-at"
      tag="span"
    >
      <template #id>
        <a :href="commit.url" target="_blank" class="normal-link"
          >{{ commit.id.substring(0, 7) }}:</a
        >
      </template>
      <template #title>
        <span class="text-main">{{ commit.title }}</span>
      </template>
      <template #author>{{ pushEvent.authorName }}</template>
      <template #time>
        <HumanizeDate :date="commit.createdTime" />
      </template>
    </i18n-t>
  </div>
</template>

<script setup lang="ts">
import { head } from "lodash-es";
import { computed } from "vue";
import HumanizeDate from "@/components/misc/HumanizeDate.vue";
import { PushEvent, VcsType } from "@/types/proto/v1/vcs";
import { useIssueContext } from "../../logic";
import { useActiveTaskSheet } from "./useActiveTaskSheet";

const { isCreating } = useIssueContext();

const { sheet, sheetReady } = useActiveTaskSheet();

const pushEvent = computed((): PushEvent | undefined => {
  if (isCreating.value) return undefined;

  if (!sheetReady.value) return undefined;
  if (!sheet.value) return undefined;
  return sheet.value.pushEvent;
});

const commit = computed(() => {
  // Use commits[0] for new format
  // Use fileCommit for legacy data (if possible)
  // Use undefined otherwise
  return head(pushEvent.value?.commits) ?? pushEvent.value?.fileCommit;
});

const vcsBranch = computed((): string => {
  if (pushEvent.value) {
    return pushEvent.value.ref.replace(/^refs\/heads\//g, "");
  }
  return "";
});

const vcsBranchUrl = computed((): string => {
  if (pushEvent.value) {
    if (pushEvent.value.vcsType === VcsType.GITLAB) {
      return `${pushEvent.value.repositoryUrl}/-/tree/${vcsBranch.value}`;
    } else if (pushEvent.value.vcsType === VcsType.GITHUB) {
      return `${pushEvent.value.repositoryUrl}/tree/${vcsBranch.value}`;
    } else if (pushEvent.value.vcsType === VcsType.BITBUCKET) {
      return `${pushEvent.value.repositoryUrl}/src/${vcsBranch.value}`;
    }
  }
  return "";
});

defineExpose({
  shown: computed(() => {
    return !!pushEvent.value;
  }),
});
</script>
