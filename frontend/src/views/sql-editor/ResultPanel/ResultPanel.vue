<template>
  <ResultView
    class="w-full h-full"
    :result-set="queryResult"
    :loading="loading"
  />
</template>

<script lang="ts" setup>
import { computed } from "vue";

import { ResultView } from "../EditorCommon/";
import { useTabStore } from "@/store";

const tabStore = useTabStore();

const queryResult = computed(() => tabStore.currentTab.queryResult);
const loading = computed(() => tabStore.currentTab.isExecutingSQL);
</script>
