<template>
  <div class="px-4 py-2 flex flex-col gap-y-4">
    <div class="flex items-center justify-between">
      <span class="textlabel">
        {{ $t("common.activity") }}
      </span>

      <Subscribers v-if="showSubscribers" />
    </div>

    <ActivityList />
  </div>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { usePageMode } from "@/store";
import ActivityList from "./ActivityList.vue";
import Subscribers from "./Subscribers";

const pageMode = usePageMode();

const showSubscribers = computed(() => {
  return pageMode.value === "BUNDLED";
});
</script>
