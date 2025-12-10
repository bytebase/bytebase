<template>
  <div class="flex flex-col gap-y-4">
    <h3 class="text-base font-medium">
      {{ $t("common.activity") }}
    </h3>
    <IssueCommentList />
  </div>
</template>

<script setup lang="ts">
import IssueCommentList from "./IssueCommentList.vue";
</script>
