<template>
  <ResultViewV1
    class="w-full h-full"
    :result-set="resultSet"
    :loading="loading"
  />
</template>

<script lang="ts" setup>
import { computed } from "vue";

import { ResultViewV1 } from "../EditorCommon/";
import { useTabStore } from "@/store";

const tabStore = useTabStore();

const resultSet = computed(() => tabStore.currentTab.sqlResultSet);
const loading = computed(() => tabStore.currentTab.isExecutingSQL);
</script>
