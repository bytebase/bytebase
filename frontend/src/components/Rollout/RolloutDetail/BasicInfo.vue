<template>
  <h2 class="text-2xl">{{ rollout.title }}</h2>
  <div class="flex flex-row items-center gap-4">
    <div class="flex items-center gap-1">
      <BBAvatar size="MINI" :username="rollout.creatorEntity.title" />
      <span class="textlabel truncate">{{ rollout.creatorEntity.title }}</span>
    </div>
    <div class="flex items-center gap-1">
      <Clock4Icon class="w-4 h-auto textinfolabel" />
      <span class="textlabel">{{
        humanizeDate(getDateForPbTimestamp(rollout.createTime))
      }}</span>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { Clock4Icon } from "lucide-vue-next";
import { BBAvatar } from "@/bbkit";
import { getDateForPbTimestamp } from "@/types";
import { humanizeDate } from "@/utils";
import { useRolloutDetailContext } from "./context";

const { rollout } = useRolloutDetailContext();
</script>
