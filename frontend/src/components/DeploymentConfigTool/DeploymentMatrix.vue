<template>
  <div class="space-y-2 w-192 px-1">
    <div
      v-if="databaseListGroupByName.length === 0"
      class="textinfolabel px-10 py-4"
    >
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
      <div class="flex justify-between items-center py-0.5 space-x-2">
        <select
          v-model="state.selectedDatabaseName"
          class="btn-select w-40 disabled:cursor-not-allowed"
        >
          <option disabled>{{ $t("db.select") }}</option>
          <option
            v-for="(group, i) in databaseListGroupByName"
            :key="i"
            :value="group.name"
          >
            {{ group.name }}
          </option>
        </select>

        <YAxisRadioGroup
          v-model:label="state.label"
          :label-list="labelList"
          class="text-sm"
        />
      </div>

      <div v-if="selectedDatabaseGroup">
        <DeployDatabaseTable
          :database-list="selectedDatabaseGroup.list"
          :label="state.label"
          :label-list="labelList"
          :environment-list="environmentList"
          :deployment="deployment"
        />
      </div>
    </template>
  </div>
</template>

<script lang="ts" setup>
import { computed, reactive, watch } from "vue";
import {
  Project,
  DeploymentConfig,
  Environment,
  Database,
  Label,
  LabelKeyType,
} from "@/types";
import { groupBy } from "lodash-es";
import { parseDatabaseNameByTemplate } from "@/utils";
import { DeployDatabaseTable } from "../TenantDatabaseTable";

type DatabaseGroup = {
  name: string;
  list: Database[];
};

const props = defineProps<{
  project: Project;
  deployment: DeploymentConfig;
  environmentList: Environment[];
  databaseList: Database[];
  labelList: Label[];
}>();

const state = reactive({
  label: "bb.environment" as LabelKeyType,
  selectedDatabaseName: undefined as string | undefined,
});

const databaseListGroupByName = computed((): DatabaseGroup[] => {
  const { dbNameTemplate } = props.project;

  if (dbNameTemplate && props.labelList.length === 0) {
    // We can't calculate dbname correctly if labelList hasn't been fetched
    // So return empty array as a fallback
    return [];
  }

  const dict = groupBy(props.databaseList, (db) => {
    if (dbNameTemplate) {
      return parseDatabaseNameByTemplate(
        db.name,
        dbNameTemplate,
        props.labelList
      );
    } else {
      return db.name;
    }
  });
  return Object.keys(dict).map((name) => ({
    name,
    list: dict[name],
  }));
});

watch(
  databaseListGroupByName,
  (groups) => {
    // reset selection when databaseList changed
    if (groups.length > 0) {
      state.selectedDatabaseName = groups[0].name;
    } else {
      state.selectedDatabaseName = undefined;
    }
  },
  { immediate: true }
);

const selectedDatabaseGroup = computed((): DatabaseGroup | undefined => {
  if (!state.selectedDatabaseName) return undefined;

  return databaseListGroupByName.value.find(
    (group) => group.name === state.selectedDatabaseName
  );
});
</script>
