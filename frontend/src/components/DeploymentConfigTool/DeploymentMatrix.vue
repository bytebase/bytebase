<template>
  <div class="space-y-4 w-192 px-1">
    <div class="flex justify-end items-center">
      <NInputGroup style="width: auto" class="py-0.5">
        <NInputGroupLabel
          :bordered="false"
          style="--n-group-label-color: transparent"
        >
          Group by
        </NInputGroupLabel>
        <YAxisRadioGroup
          v-model:label="state.label"
          :database-list="databaseList"
        />
        <SearchBox
          v-if="showSearchBox"
          v-model:value="state.keyword"
          :placeholder="$t('common.filter-by-name')"
        />
      </NInputGroup>
    </div>
    <div class="w-full overflow-x-auto">
      <DeploymentMatrixDataTable
        :database-list="filteredDatabaseList"
        :label="state.label"
        :environment-list="environmentList"
        :deployment="deployment"
      />
    </div>
  </div>
</template>

<script lang="ts" setup>
import { NInputGroup, NInputGroupLabel } from "naive-ui";
import { computed, reactive } from "vue";
import type { ComposedDatabase } from "@/types";
import type { Environment } from "@/types/proto/v1/environment_service";
import type { DeploymentConfig } from "@/types/proto/v1/project_service";
import { filterDatabaseV1ByKeyword } from "@/utils";
import {
  YAxisRadioGroup,
  DeploymentMatrixDataTable,
} from "../TenantDatabaseTable";
import { SearchBox } from "../v2";

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
  label: "environment",
  keyword: "",
});

const filteredDatabaseList = computed(() => {
  return props.databaseList.filter((db) => {
    return filterDatabaseV1ByKeyword(db, state.keyword, [
      "name",
      "environment",
      "instance",
    ]);
  });
});
</script>
