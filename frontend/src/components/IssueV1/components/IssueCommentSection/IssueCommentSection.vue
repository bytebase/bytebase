<template>
  <div class="px-4 py-2 flex flex-col gap-y-4">
    <div class="flex items-center justify-between">
      <span class="textlabel">
        {{ $t("common.activity") }}
      </span>

      <Subscribers v-if="!hideSubscribers" />
    </div>

    <IssueCommentList />
  </div>
</template>

<script lang="ts" setup>
import { useAppFeature } from "@/store";
import IssueCommentList from "./IssueCommentList.vue";
import Subscribers from "./Subscribers";

const hideSubscribers = useAppFeature("bb.feature.issue.hide-subscribers");
</script>
