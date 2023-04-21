<template>
  <div class="p-2">
    <SlowQueryPanel v-if="ready" v-model:filter="filter" />
  </div>
</template>

<script lang="ts" setup>
import { shallowRef, watch, watchEffect } from "vue";
import { useRoute, useRouter } from "vue-router";
import { isEqual } from "lodash-es";

import {
  SlowQueryPanel,
  SlowQueryFilterParams,
  defaultSlowQueryFilterParams,
} from "@/components/SlowQuery";
import {
  useDatabaseStore,
  useEnvironmentStore,
  useInstanceStore,
  useProjectStore,
} from "@/store";
import { UNKNOWN_ID } from "@/types";

const route = useRoute();
const router = useRouter();
const ready = shallowRef(false);
const filter = shallowRef<SlowQueryFilterParams>({
  ...defaultSlowQueryFilterParams(),
});

const syncFilterParamsFromQuery = async () => {
  const extractSlowQueryLogFilterFromQuery = async () => {
    const { query } = route;
    const params: SlowQueryFilterParams = defaultSlowQueryFilterParams();
    if (query.environment) {
      const id = parseInt(query.environment as string, 10) ?? UNKNOWN_ID;
      const environment = useEnvironmentStore().getEnvironmentById(id);
      if (environment && environment.id !== UNKNOWN_ID) {
        params.environment = environment;
      }
    }
    if (query.project) {
      const id = parseInt(query.project as string, 10) ?? UNKNOWN_ID;
      const project = await useProjectStore().getOrFetchProjectById(id);
      if (project && project.id !== UNKNOWN_ID) {
        params.project = project;
      }
    }
    if (query.instance) {
      const id = parseInt(query.instance as string, 10) ?? UNKNOWN_ID;
      const instance = await useInstanceStore().getOrFetchInstanceById(id);
      if (instance && instance.id !== UNKNOWN_ID) {
        params.instance = instance;
      }
    }
    if (query.database) {
      const id = parseInt(query.database as string, 10) ?? UNKNOWN_ID;
      const database = await useDatabaseStore().getOrFetchDatabaseById(id);
      if (database && database.id !== UNKNOWN_ID) {
        params.database = database;
      }
    }
    if (query.timeRange) {
      const timeRangeStr = String(query.timeRange);
      const matches = timeRangeStr.match(/^(\d+)-(\d+)$/);
      if (matches && matches.length === 3) {
        const from = parseInt(matches[1], 10) * 1000;
        const to = parseInt(matches[2], 10) * 1000;
        if (from > 0 && to > 0) {
          params.timeRange = [from, to];
        }
      }
    }
    return params;
  };

  const params = await extractSlowQueryLogFilterFromQuery();
  filter.value = params;
  ready.value = true;
};

watchEffect(syncFilterParamsFromQuery);

watch(
  filter,
  (filter) => {
    const wrapQueryFromFilterParams = (params: SlowQueryFilterParams) => {
      const query: Record<string, any> = {};
      if (params.project && params.project.id !== UNKNOWN_ID) {
        query.project = params.project.id;
      }
      if (params.environment && params.environment.id !== UNKNOWN_ID) {
        query.environment = params.environment.id;
      }
      if (params.instance && params.instance.id !== UNKNOWN_ID) {
        query.instance = params.instance.id;
      }
      if (params.database && params.database.id !== UNKNOWN_ID) {
        query.database = params.database.id;
      }
      if (params.timeRange) {
        if (
          !isEqual(params.timeRange, defaultSlowQueryFilterParams().timeRange)
        ) {
          query.timeRange = params.timeRange
            .map((ms) => Math.floor(ms / 1000))
            .join("-");
        }
      }
      return query;
    };

    const query = wrapQueryFromFilterParams(filter);
    router.replace({
      ...route,
      query,
    });
  },
  { deep: true }
);
</script>
