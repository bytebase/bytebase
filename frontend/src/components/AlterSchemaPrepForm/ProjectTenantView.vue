<template>
  <!-- eslint-disable vue/no-mutating-props -->

  <div class="project-tenant-view">
    <template v-if="!!project">
      <template v-if="deployment?.id === UNKNOWN_ID">
        <p>this project has no deployment config. go and config first.</p>
        <p>
          <router-link to="#deployment-config" @click="$emit('dismiss')">
            go
          </router-link>
        </p>
      </template>
      <template v-else>
        <div v-if="databaseListGroupByName.length === 0" class="textinfolabel">
          <i18n-t keypath="project.overview.no-db-prompt" tag="p">
            <template #newDb>
              <span class="text-main">{{ $t("quick-action.new-db") }}</span>
            </template>
            <template #transferInDb>
              <span class="text-main">
                {{ $t("quick-action.transfer-in-db") }}
              </span>
            </template>
          </i18n-t>
        </div>
        <NCollapse
          v-else
          display-directive="if"
          accordion
          :expanded-names="state.selectedDatabaseName"
          @update:expanded-names="
            (names) => (state.selectedDatabaseName = names[0])
          "
        >
          <NCollapseItem
            v-for="{ name, list } in databaseListGroupByName"
            :key="name"
            :title="name"
            :name="name"
          >
            <template #header>
              <span class="text-base">{{ name }}</span>
              <span v-if="name === state.selectedDatabaseName">
                <heroicons-outline:check class="w-5 h-5 ml-2 text-success" />
              </span>
            </template>
            <template #header-extra>
              <span class="text-control-placeholder">
                {{ $t("deployment-config.n-databases", list.length) }}
              </span>
            </template>

            <DeployDatabaseTable
              :database-list="list"
              :label-list="labelList"
              :environment-list="environmentList"
              :deployment="deployment!"
            />
          </NCollapseItem>
        </NCollapse>
      </template>
    </template>
  </div>
</template>

<script lang="ts" setup>
/* eslint-disable vue/no-mutating-props */

import { computed, defineProps, defineEmits, watchEffect, watch } from "vue";
import { useStore } from "vuex";
import {
  Database,
  DeploymentConfig,
  Environment,
  Label,
  Project,
  UNKNOWN_ID,
} from "../../types";
import { NCollapse, NCollapseItem } from "naive-ui";
import { groupBy } from "lodash-es";
import { DeployDatabaseTable } from "../TenantDatabaseTable";

export type State = {
  selectedDatabaseName: string | undefined;
};

const props = defineProps<{
  databaseList: Database[];
  environmentList: Environment[];
  project?: Project;
  state: State;
}>();

defineEmits<{
  (event: "dismiss"): void;
}>();

const store = useStore();

const fetchData = () => {
  store.dispatch("label/fetchLabelList");
  if (props.project) {
    store.dispatch(
      "deployment/fetchDeploymentConfigByProjectId",
      props.project.id
    );
  }
};

watchEffect(fetchData);

const labelList = computed(() => store.getters["label/labelList"]() as Label[]);

const deployment = computed(() => {
  if (props.project) {
    return store.getters["deployment/deploymentConfigByProjectId"](
      props.project.id
    ) as DeploymentConfig;
  } else {
    return undefined;
  }
});

const databaseListGroupByName = computed(
  (): { name: string; list: Database[] }[] => {
    const dict = groupBy(props.databaseList, "name");
    return Object.keys(dict).map((name) => ({
      name,
      list: dict[name],
    }));
  }
);

watch(
  databaseListGroupByName,
  (groups) => {
    // reset selection when databaseList changed
    if (groups.length > 0) {
      props.state.selectedDatabaseName = groups[0].name;
    } else {
      props.state.selectedDatabaseName = undefined;
    }
  },
  { immediate: true }
);
</script>

<style scoped lang="postcss">
.project-tenant-view {
  @apply w-192;
}

.project-tenant-view :global(.n-collapse-item) {
  @apply mt-0 !important;
}

.project-tenant-view
  :global(.n-collapse-item.n-collapse-item--active + .n-collapse-item) {
  @apply border-0 !important;
}

.project-tenant-view :global(.n-collapse-item__header) {
  @apply pt-4 pb-4 border-control-light !important;
}

.project-tenant-view :global(.n-collapse-item__content-inner) {
  @apply pt-0 !important;
}
</style>
