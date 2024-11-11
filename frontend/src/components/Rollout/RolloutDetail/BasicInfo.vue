<template>
  <h2 class="text-2xl">{{ rollout.title }}</h2>
  <div class="flex flex-row items-center gap-4">
    <router-link
      v-if="rollout.issue"
      :to="`/${rollout.issue}`"
      class="normal-link flex items-center gap-1"
    >
      <CircleDotIcon class="w-5 h-auto textinfolabel" />
      <span>#{{ issueUid }}</span>
    </router-link>
    <div class="flex items-center gap-1">
      <BBAvatar size="MINI" :username="rollout.creatorEntity.title" />
      <span class="textlabel truncate">{{ rollout.creatorEntity.title }}</span>
    </div>
    <div class="flex items-center gap-1">
      <Clock4Icon class="w-5 h-auto textinfolabel" />
      <span class="textlabel">{{
        humanizeDate(getDateForPbTimestamp(rollout.createTime))
      }}</span>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { Clock4Icon, CircleDotIcon } from "lucide-vue-next";
import { computed } from "vue";
import { BBAvatar } from "@/bbkit";
import { getDateForPbTimestamp } from "@/types";
import { extractIssueUID, humanizeDate } from "@/utils";
import { useRolloutDetailContext } from "./context";

const { rollout } = useRolloutDetailContext();

const issueUid = computed(() => extractIssueUID(rollout.value.issue));
</script>
