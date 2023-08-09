<template>
  <div class="ml-3 min-w-0 flex-1">
    <div class="min-w-0 flex-1 pt-1 flex justify-between">
      <div class="text-sm text-control-light space-x-1">
        <ActionCreator
          v-if="
            extractUserResourceName(activity.creator) !== SYSTEM_BOT_EMAIL ||
            activity.action === LogEntity_Action.ACTION_ISSUE_COMMENT_CREATE
          "
          :activity="activity"
        />

        <ActionSentence :issue="issue" :activity="activity" />

        <HumanizeTs
          :ts="(activity.createTime?.getTime() ?? 0) / 1000"
          class="ml-1"
        />

        <span
          v-if="
            activity.createTime?.getTime() !== activity.updateTime?.getTime() &&
            activity.action == LogEntity_Action.ACTION_ISSUE_COMMENT_CREATE
          "
        >
          <span>({{ $t("common.edited") }}</span>
          <HumanizeTs
            :ts="(activity.updateTime?.getTime() ?? 0) / 1000"
            class="ml-1"
          />)
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
      <slot name="comment" />

      <template
        v-if="
          activity.action == LogEntity_Action.ACTION_PIPELINE_TASK_FILE_COMMIT
        "
      >
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
  ComposedIssue,
  ActivityTaskFileCommitPayload,
  SYSTEM_BOT_EMAIL,
} from "@/types";
import ActionCreator from "./ActionCreator.vue";
import ActionSentence from "./ActionSentence.vue";
import { LogEntity, LogEntity_Action } from "@/types/proto/v1/logging_service";
import { extractUserResourceName } from "@/utils";

defineProps<{
  issue: ComposedIssue;
  index: number;
  activity: LogEntity;
  similar: LogEntity[];
}>();

const fileCommitActivityUrl = (activity: LogEntity) => {
  const payload = JSON.parse(activity.payload) as ActivityTaskFileCommitPayload;
  if (payload.vcsInstanceUrl.includes("https://github.com"))
    return `${payload.vcsInstanceUrl}/${payload.repositoryFullPath}/commit/${payload.commitId}`;
  return `${payload.vcsInstanceUrl}/${payload.repositoryFullPath}/-/commit/${payload.commitId}`;
};
</script>
