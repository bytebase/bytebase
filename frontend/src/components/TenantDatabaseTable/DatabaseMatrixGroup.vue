<template>
  <div class="flex flex-col items-start w-max space-y-2">
    <div class="space-y-1">
      <DatabaseMatrixItem
        v-for="item in shownDatabases"
        :key="item.name"
        :database="item"
      />
    </div>
    <p v-if="showDisplayMore" class="textinfolabel space-x-1">
      <span>{{
        $t("deployment-config.matched-databases.n", { n: databases.length })
      }}</span>
      <NButton size="tiny" @click="() => (state.showAll = true)">{{
        $t("deployment-config.matched-databases.show-more")
      }}</NButton>
    </p>
  </div>

  <DatabaseMatrixGroupDrawer
    v-if="state.showAll"
    :databases="databases"
    @dismiss="() => (state.showAll = false)"
  />
</template>

<script lang="ts" setup>
import { NButton } from "naive-ui";
import type { PropType } from "vue";
import { computed, reactive } from "vue";
import type { ComposedDatabase } from "@/types";
import DatabaseMatrixGroupDrawer from "./DatabaseMatrixGroupDrawer.vue";
import DatabaseMatrixItem from "./DatabaseMatrixItem.vue";

interface LocalState {
  showAll: boolean;
}

// Show databases with limit.
const MAX_DISPLAY_DATABASES = 10;

const props = defineProps({
  databases: {
    type: Object as PropType<ComposedDatabase[]>,
    required: true,
  },
});

const state = reactive<LocalState>({
  showAll: false,
});

const shownDatabases = computed(() => {
  return props.databases.slice(0, MAX_DISPLAY_DATABASES);
});

const showDisplayMore = computed(() => {
  return props.databases.length > MAX_DISPLAY_DATABASES;
});
</script>
