<template>
  <div class="issue-debug">
    issue vcs push event info if needed
    <!-- <p
      v-if="pushEvent"
      class="mt-1 text-sm text-control-light flex flex-row items-center space-x-1"
    >
      <template v-if="pushEvent.vcsType.startsWith('GITLAB')">
        <img class="h-4 w-auto" src="../../assets/gitlab-logo.svg" />
      </template>
      <template v-else-if="pushEvent.vcsType.startsWith('GITHUB')">
        <img class="h-4 w-auto" src="../../assets/github-logo.svg" />
      </template>
      <template v-else-if="pushEvent.vcsType.startsWith('BITBUCKET')">
        <img class="h-4 w-auto" src="../../assets/bitbucket-logo.svg" />
      </template>
      <a :href="vcsBranchUrl" target="_blank" class="normal-link">{{
        `${vcsBranch}@${pushEvent.repositoryFullPath}`
      }}</a>

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
        <template #time>{{
          dayjs(commit.createdTs * 1000).format("LLL")
        }}</template>
      </i18n-t>
    </p> -->
  </div>
</template>

<script setup lang="ts">
// const pushEvent = computed((): VCSPushEvent | undefined => {
//   if (issue.value.type == "bb.issue.database.schema.update") {
//     const payload = activeTask(issue.value.pipeline!)
//       .payload as TaskDatabaseSchemaUpdatePayload;
//     return payload?.pushEvent;
//   } else if (issue.value.type == "bb.issue.database.data.update") {
//     const payload = activeTask(issue.value.pipeline!)
//       .payload as TaskDatabaseDataUpdatePayload;
//     return payload?.pushEvent;
//   }
//   return undefined;
// });

// const commit = computed(() => {
//   // Use commits[0] for new format
//   // Use fileCommit for legacy data (if possible)
//   // Use undefined otherwise
//   return head(pushEvent.value?.commits) ?? pushEvent.value?.fileCommit;
// });

// const vcsBranch = computed((): string => {
//   if (pushEvent.value) {
//     return pushEvent.value.ref.replace(/^refs\/heads\//g, "");
//   }
//   return "";
// });

// const vcsBranchUrl = computed((): string => {
//   if (pushEvent.value) {
//     if (pushEvent.value.vcsType == "GITLAB") {
//       return `${pushEvent.value.repositoryUrl}/-/tree/${vcsBranch.value}`;
//     } else if (pushEvent.value.vcsType == "GITHUB") {
//       return `${pushEvent.value.repositoryUrl}/tree/${vcsBranch.value}`;
//     } else if (pushEvent.value.vcsType == "BITBUCKET") {
//       return `${pushEvent.value.repositoryUrl}/src/${vcsBranch.value}`;
//     }
//   }
//   return "";
// });
</script>
