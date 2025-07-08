<template>
  <li>
    <div class="relative pb-4">
      <span
        v-if="issueComments.length > 0"
        class="absolute left-4 -ml-px h-full w-0.5 bg-block-border"
        aria-hidden="true"
      ></span>
      <div class="relative flex items-start">
        <div>
          <div class="relative pl-0.5">
            <div
              class="w-7 h-7 bg-control-bg rounded-full ring-4 ring-white flex items-center justify-center"
            >
              <PlusIcon class="w-5 h-5 text-control" />
            </div>
          </div>
        </div>

        <div class="ml-3 min-w-0 flex-1">
          <div class="min-w-0 flex-1 pt-1 flex justify-between">
            <div class="text-sm text-control-light space-x-1">
              <ActionCreator :creator="issue.creator" />
              <span>{{ $t("activity.sentence.created-issue") }}</span>
              <HumanizeTs
                :ts="getTimeForPbTimestampProtoEs(issue.createTime, 0) / 1000"
                class="ml-1 text-gray-400"
              />
            </div>
          </div>
        </div>
      </div>
    </div>
  </li>
</template>

<script lang="ts" setup>
import { PlusIcon } from "lucide-vue-next";
import HumanizeTs from "@/components/misc/HumanizeTs.vue";
import { getTimeForPbTimestampProtoEs, type ComposedIssue } from "@/types";
import ActionCreator from "./ActionCreator.vue";
import type { DistinctIssueComment } from "./common";

defineProps<{
  issue: ComposedIssue;
  issueComments: DistinctIssueComment[];
}>();
</script>
