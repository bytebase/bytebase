<template>
  <div class="space-y-2 w-192 px-1">
    <div v-if="databaseList.length === 0" class="textinfolabel px-10 py-4">
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
      <div class="flex justify-end items-center py-0.5 space-x-2">
        <YAxisRadioGroup v-model:label="state.label" class="text-sm" />
        <BBTableSearch
          v-if="showSearchBox"
          class="w-60"
          :placeholder="$t('database.search-database')"
          @change-text="(text: string) => (state.keyword = text)"
        />
      </div>

      <DeployDatabaseTable
        :database-list="filteredDatabaseList"
        :label="state.label"
        :environment-list="environmentList"
        :deployment="deployment"
      />
    </template>
  </div>
</template>

<script lang="ts" setup>
import { computed, reactive } from "vue";
import {
  Project,
  DeploymentConfig,
  Environment,
  Database,
  LabelKeyType,
} from "@/types";
import { DeployDatabaseTable } from "../TenantDatabaseTable";
import { filterDatabaseByKeyword } from "@/utils";

const props = withDefaults(
  defineProps<{
    project: Project;
    deployment: DeploymentConfig;
    environmentList: Environment[];
    databaseList: Database[];
    showSearchBox: boolean;
  }>(),
  {
    showSearchBox: false,
  }
);

const state = reactive({
  label: "bb.environment" as LabelKeyType,
  keyword: "",
});

const filteredDatabaseList = computed(() => {
  return props.databaseList.filter((db) => {
    return filterDatabaseByKeyword(db, state.keyword, [
      "name",
      "environment",
      "instance",
      "tenant",
    ]);
  });
});
</script>
