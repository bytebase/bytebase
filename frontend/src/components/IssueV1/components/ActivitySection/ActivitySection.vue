<template>
  <div class="px-4 py-2 flex flex-col gap-y-4">
    <div class="flex items-center justify-between">
      <span class="textlabel">
        {{ $t("common.activity") }}
      </span>

      <Subscribers />
    </div>

    <ActivityList />
  </div>
</template>

<script lang="ts" setup>
import ActivityList from "./ActivityList.vue";
import Subscribers from "./Subscribers";
</script>
