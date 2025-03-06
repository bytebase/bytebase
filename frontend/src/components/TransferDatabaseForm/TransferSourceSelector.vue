<template>
  <div class="textlabel">
    <div
      class="flex flex-col md:flex-row md:items-center gap-y-2 justify-between"
    >
      <div v-if="project.name !== DEFAULT_PROJECT_NAME" class="radio-set-row">
        <NRadioGroup
          :value="transferSource"
          class="space-x-4"
          size="large"
          @update:value="$emit('update:transferSource', $event)"
        >
          <NRadio :value="'OTHER'">
            {{ $t("quick-action.from-projects") }}
          </NRadio>
          <NRadio v-if="hasPermissionForDefaultProject" :value="'DEFAULT'">
            {{ $t("quick-action.from-unassigned-databases") }}
          </NRadio>
        </NRadioGroup>
      </div>
      <NInputGroup style="width: auto">
        <EnvironmentSelect
          class="!w-40"
          :environment-name="environmentFilter?.name"
          @update:environment-name="changeEnvironmentFilter"
        />
        <InstanceSelect
          class="!w-40"
          :instance="instanceFilter?.name"
          @update:instance-name="changeInstanceFilter"
        />
        <SearchBox
          class="!w-40"
          :value="searchText"
          :placeholder="$t('database.filter-database')"
          @update:value="$emit('search-text-change', $event)"
        />
      </NInputGroup>
    </div>
    <div v-if="transferSource == 'DEFAULT'" class="textinfolabel mt-2">
      {{ $t("quick-action.unassigned-db-hint") }}
    </div>
  </div>
</template>

<script lang="ts" setup>
import { NInputGroup, NRadio, NRadioGroup } from "naive-ui";
import { InstanceSelect, SearchBox } from "@/components/v2";
import { useEnvironmentV1Store, useInstanceResourceByName } from "@/store";
import {
  DEFAULT_PROJECT_NAME,
  isValidEnvironmentName,
  isValidInstanceName,
} from "@/types";
import type { Environment } from "@/types/proto/v1/environment_service";
import type { InstanceResource } from "@/types/proto/v1/instance_service";
import type { Project } from "@/types/proto/v1/project_service";
import EnvironmentSelect from "../v2/Select/EnvironmentSelect.vue";
import type { TransferSource } from "./utils";

withDefaults(
  defineProps<{
    project: Project;
    transferSource: TransferSource;
    hasPermissionForDefaultProject: boolean;
    instanceFilter?: InstanceResource;
    environmentFilter?: Environment;
    searchText: string;
  }>(),
  {
    instanceFilter: undefined,
    environmentFilter: undefined,
    searchText: "",
  }
);

const emit = defineEmits<{
  (event: "update:transferSource", src: TransferSource): void;
  (event: "select-instance", instance: InstanceResource | undefined): void;
  (event: "select-environment", env: Environment | undefined): void;
  (event: "search-text-change", searchText: string): void;
}>();

const changeEnvironmentFilter = (name: string | undefined) => {
  if (!isValidEnvironmentName(name)) {
    return emit("select-environment", undefined);
  }
  emit(
    "select-environment",
    useEnvironmentV1Store().getEnvironmentByName(name)
  );
};

const changeInstanceFilter = (name: string | undefined) => {
  if (!isValidInstanceName(name)) {
    return emit("select-instance", undefined);
  }
  emit("select-instance", useInstanceResourceByName(name));
};
</script>
