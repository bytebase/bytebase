<template>
  <div class="px-6 py-5 flex flex-col gap-y-5">
    <div class="flex items-center justify-between">
      <span class="text-base font-semibold text-gray-900">
        {{ $t("common.activity") }}
      </span>
    </div>

    <IssueCommentList />
  </div>
</template>

<script lang="ts" setup>
import IssueCommentList from "./IssueCommentList.vue";
</script>
