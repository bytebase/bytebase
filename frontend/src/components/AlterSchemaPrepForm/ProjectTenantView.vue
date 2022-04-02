<template>
  <!-- eslint-disable vue/no-mutating-props -->

  <div class="project-tenant-view">
    <template v-if="!!project">
      <template v-if="deployment?.id === UNKNOWN_ID">
        <i18n-t
          tag="p"
          keypath="deployment-config.project-has-no-deployment-config"
        >
          <template #go>
            <router-link
              :to="{
                path: `/project/${projectSlug(project)}`,
                hash: '#deployment-config',
              }"
              active-class=""
              exact-active-class=""
              class="px-1 underline hover:bg-link-hover"
              @click="$emit('dismiss')"
            >
              {{ $t("deployment-config.go-and-config") }}
            </router-link>
          </template>
        </i18n-t>
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
        <template v-else>
          <YAxisRadioGroup
            v-model:label="label"
            :label-list="labelList"
            class="text-sm pt-2 pb-1"
          />
          <NCollapse
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
                :label="label"
                :label-list="labelList"
                :environment-list="environmentList"
                :deployment="deployment!"
              />
            </NCollapseItem>
          </NCollapse>
        </template>
      </template>
    </template>
  </div>
</template>

<script lang="ts" setup>
/* eslint-disable vue/no-mutating-props */

import { computed, watchEffect, watch, ref } from "vue";
import { useStore } from "vuex";
import {
  Database,
  DatabaseId,
  DeploymentConfig,
  Environment,
  Label,
  LabelKeyType,
  Project,
  UNKNOWN_ID,
} from "../../types";
import { NCollapse, NCollapseItem } from "naive-ui";
import { groupBy } from "lodash-es";
import { DeployDatabaseTable } from "../TenantDatabaseTable";
import { parseDatabaseNameByTemplate } from "../../utils";
import { getPipelineFromDeploymentSchedule } from "../../utils";
import { useLabelStore } from "@/store";
import { storeToRefs } from "pinia";

export type State = {
  selectedDatabaseName: string | undefined;
  deployingTenantDatabaseList: DatabaseId[];
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
const labelStore = useLabelStore();

const fetchData = () => {
  labelStore.fetchLabelList();
  if (props.project) {
    store.dispatch(
      "deployment/fetchDeploymentConfigByProjectId",
      props.project.id
    );
  }
};

watchEffect(fetchData);

const label = ref<LabelKeyType>("bb.environment");
const { labelList } = storeToRefs(labelStore);

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
    if (!props.project) return [];
    if (props.project.dbNameTemplate && labelList.value.length === 0) return [];

    const dict = groupBy(props.databaseList, (db) => {
      if (props.project!.dbNameTemplate) {
        return parseDatabaseNameByTemplate(
          db.name,
          props.project!.dbNameTemplate,
          labelList.value
        );
      } else {
        return db.name;
      }
    });
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

watchEffect(() => {
  if (!deployment.value) return;
  const name = props.state.selectedDatabaseName;
  if (!name) {
    props.state.deployingTenantDatabaseList = [];
  } else {
    // find the selected database list group by name
    const databaseGroup = databaseListGroupByName.value.find(
      (group) => group.name === name
    );
    const databaseList = databaseGroup?.list || [];

    // calculate the deployment matching to preview the pipeline
    const stages = getPipelineFromDeploymentSchedule(
      databaseList,
      deployment.value.schedule
    );

    // flatten all stages' database id list
    // these databases are to be deployed
    const databaseIdList = stages.flatMap((stage) => stage.map((db) => db.id));
    props.state.deployingTenantDatabaseList = databaseIdList;
  }
});
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
