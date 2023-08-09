<template>
  <li>
    <div :id="`#${activity.name}`" class="relative pb-4">
      <span
        v-if="index !== activityList.length - 1"
        class="absolute left-4 -ml-px h-full w-0.5 bg-block-border"
        aria-hidden="true"
      ></span>
      <div class="relative flex items-start">
        <ActionIcon :activity="activity" />

        <ActivityAction
          :issue="issue"
          :index="index"
          :activity="activity"
          :similar="similar"
        >
          <template #subject-suffix>
            <slot name="subject-suffix" />
          </template>

          <template #comment>
            <slot name="comment" />
          </template>
        </ActivityAction>
      </div>
    </div>
  </li>
</template>

<script lang="ts" setup>
import { ComposedIssue } from "@/types";
import { LogEntity } from "@/types/proto/v1/logging_service";
import ActionIcon from "./ActionIcon.vue";
import ActivityAction from "./ActivityAction.vue";
import { DistinctActivity } from "./common";

defineProps<{
  issue: ComposedIssue;
  activityList: DistinctActivity[];
  index: number;
  activity: LogEntity;
  similar: LogEntity[];
}>();
</script>
