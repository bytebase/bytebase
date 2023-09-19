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
      <div class="w-full overflow-x-auto">
        <DeployDatabaseTable
          :database-list="filteredDatabaseList"
          :label="state.label"
          :environment-list="environmentList"
          :deployment="deployment"
        />
      </div>
    </template>
  </div>
</template>

<script lang="ts" setup>
import { computed, reactive } from "vue";
import { ComposedDatabase, LabelKeyType } from "@/types";
import { Environment } from "@/types/proto/v1/environment_service";
import { DeploymentConfig } from "@/types/proto/v1/project_service";
import { filterDatabaseV1ByKeyword } from "@/utils";
import { DeployDatabaseTable } from "../TenantDatabaseTable";

const props = withDefaults(
  defineProps<{
    deployment: DeploymentConfig;
    environmentList: Environment[];
    databaseList: ComposedDatabase[];
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
    return filterDatabaseV1ByKeyword(db, state.keyword, [
      "name",
      "environment",
      "instance",
      "tenant",
    ]);
  });
});
</script>
