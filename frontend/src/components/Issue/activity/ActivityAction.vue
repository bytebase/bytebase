<template>
  <div class="ml-3 min-w-0 flex-1">
    <div class="min-w-0 flex-1 pt-1 flex justify-between">
      <div class="text-sm text-control-light space-x-1">
        <ActionCreator
          v-if="
            activity.creator.id !== SYSTEM_BOT_ID ||
            activity.type === 'bb.issue.comment.create'
          "
          :activity="activity"
        />

        <ActionSentence :issue="issue" :activity="activity" />

        <HumanizeTs :ts="activity.createdTs" class="ml-1" />

        <span
          v-if="
            activity.createdTs != activity.updatedTs &&
            activity.type == 'bb.issue.comment.create'
          "
        >
          <span>({{ $t("common.edited") }}</span>
          <HumanizeTs :ts="activity.updatedTs" class="ml-1" />)
        </span>

        <span
          v-if="similar.length > 0"
          class="text-sm font-normal text-gray-400 ml-1"
        >
          {{
            $t("activity.n-similar-activities", {
              count: similar.length + 1,
            })
          }}
        </span>
      </div>

      <slot name="subject-suffix"></slot>
    </div>
    <div class="mt-2 text-sm text-control whitespace-pre-wrap">
      <slot name="comment">
        <Comment :activity="activity" />
      </slot>

      <template v-if="activity.type == 'bb.pipeline.task.file.commit'">
        <a
          :href="fileCommitActivityUrl(activity)"
          target="__blank"
          class="normal-link flex flex-row items-center"
        >
          {{ $t("issue.view-commit") }}
          <heroicons-outline:external-link class="w-4 h-4" />
        </a>
      </template>
    </div>
  </div>
</template>

<script lang="ts" setup>
import {
  Issue,
  Activity,
  ActivityTaskFileCommitPayload,
  SYSTEM_BOT_ID,
} from "@/types";
import Comment from "./Comment";
import ActionCreator from "./ActionCreator.vue";
import ActionSentence from "./ActionSentence.vue";

defineProps<{
  issue: Issue;
  index: number;
  activity: Activity;
  similar: Activity[];
}>();

const fileCommitActivityUrl = (activity: Activity) => {
  const payload = activity.payload as ActivityTaskFileCommitPayload;
  if (payload.vcsInstanceUrl.includes("https://github.com"))
    return `${payload.vcsInstanceUrl}/${payload.repositoryFullPath}/commit/${payload.commitId}`;
  return `${payload.vcsInstanceUrl}/${payload.repositoryFullPath}/-/commit/${payload.commitId}`;
};
</script>
