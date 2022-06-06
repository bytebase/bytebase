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
          <div class="flex justify-between items-center py-0.5">
            <div class="flex-1">
              <label class="text-base text-control">
                <template v-if="databaseListGroupByName.length > 1">
                  {{ $t("deployment-config.select-database-group") }}
                </template>
                <template v-else-if="state.selectedDatabaseName">
                  {{ state.selectedDatabaseName }}
                </template>
              </label>
            </div>
            <YAxisRadioGroup
              v-model:label="label"
              :label-list="labelList"
              class="text-sm"
            />
          </div>

          <template v-if="databaseListGroupByName.length === 1">
            <DeployDatabaseTable
              class="mt-4"
              :database-list="databaseListGroupByName[0].list"
              :label="label"
              :label-list="labelList"
              :environment-list="environmentList"
              :deployment="deployment!"
            />
          </template>
          <template v-else>
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
                <template #arrow>
                  <input
                    type="radio"
                    class="radio"
                    :checked="name === state.selectedDatabaseName"
                  />
                </template>
                <template #header>
                  <span class="text-base">{{ name }}</span>
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
    </template>
  </div>
</template>

<script lang="ts" setup>
/* eslint-disable vue/no-mutating-props */

import { computed, watchEffect, watch, ref, h, useSlots } from "vue";
import { NCollapse, NCollapseItem } from "naive-ui";
import { groupBy } from "lodash-es";
import { Translation, useI18n } from "vue-i18n";
import { RouterLink } from "vue-router";
import type {
  Database,
  DatabaseId,
  Environment,
  LabelKeyType,
  Project,
} from "@/types";
import { UNKNOWN_ID } from "@/types";
import { DeployDatabaseTable } from "../TenantDatabaseTable";
import {
  parseDatabaseNameByTemplate,
  getPipelineFromDeploymentSchedule,
  projectSlug,
} from "@/utils";
import { useDeploymentStore, useLabelList } from "@/store";
import { useOverrideSubtitle } from "@/bbkit/BBModal.vue";

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

const emit = defineEmits<{
  (event: "dismiss"): void;
}>();

const { t } = useI18n();

const deploymentStore = useDeploymentStore();

const fetchData = () => {
  if (props.project) {
    deploymentStore.fetchDeploymentConfigByProjectId(props.project.id);
  }
};

watchEffect(fetchData);

const label = ref<LabelKeyType>("bb.environment");
const labelList = useLabelList();

const deployment = computed(() => {
  if (props.project) {
    return deploymentStore.getDeploymentConfigByProjectId(props.project.id);
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

useOverrideSubtitle(() => {
  return h(
    Translation,
    {
      tag: "p",
      class: "textinfolabel",
      keypath: "deployment-config.pipeline-generated-from-deployment-config",
    },
    {
      deployment_config: () =>
        h(
          RouterLink,
          {
            to: {
              path: `/project/${projectSlug(props.project!)}`,
              hash: "#deployment-config",
            },
            activeClass: "",
            exactActiveClass: "",
            class: "underline hover:bg-link-hover",
            onClick: () => emit("dismiss"),
          },
          {
            default: () => t("common.deployment-config"),
          }
        ),
    }
  );
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
