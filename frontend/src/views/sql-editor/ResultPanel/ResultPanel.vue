<template>
  <ResultViewV1
    class="w-full h-full"
    :execute-params="executeParams"
    :result-set="resultSet"
    :loading="loading"
  />
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { useTabStore } from "@/store";
import { ResultViewV1 } from "../EditorCommon/";

const tabStore = useTabStore();

const resultSet = computed(() => tabStore.currentTab.sqlResultSet);
const executeParams = computed(() => tabStore.currentTab.executeParams);
const loading = computed(() => tabStore.currentTab.isExecutingSQL);
</script>
