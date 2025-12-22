<template>
  <div>
    <div
      class="flex flex-col md:flex-row md:items-center gap-y-2 justify-between"
    >
      <div class="radio-set-row">
        <NRadioGroup
          :value="transferSource"
          class="gap-x-4"
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
          class="w-40!"
          :value="
            environment ? formatEnvironmentName(environment.id) : undefined
          "
          @update:value="changeEnvironmentFilter($event as (string | undefined))"
        />
        <InstanceSelect
          class="w-40!"
          :project-name="sourceProjectName"
          :value="instance?.name ?? ''"
          :environment-name="environment?.name"
          @update:value="changeInstanceFilter"
        />
        <SearchBox
          class="w-40!"
          :value="searchText"
          :placeholder="$t('database.filter-database')"
          @update:value="$emit('update:search-text', $event)"
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
  formatEnvironmentName,
  isValidEnvironmentName,
  isValidInstanceName,
} from "@/types";
import { type InstanceResource } from "@/types/proto-es/v1/instance_service_pb";
import type { Environment } from "@/types/v1/environment";
import EnvironmentSelect from "../v2/Select/EnvironmentSelect.vue";
import type { TransferSource } from "./utils";

withDefaults(
  defineProps<{
    sourceProjectName: string;
    transferSource: TransferSource;
    hasPermissionForDefaultProject: boolean;
    instance?: InstanceResource;
    environment?: Environment;
    searchText: string;
  }>(),
  {
    instance: undefined,
    environment: undefined,
    searchText: "",
  }
);

const emit = defineEmits<{
  (event: "update:transferSource", src: TransferSource): void;
  (event: "update:instance", instance: InstanceResource | undefined): void;
  (event: "update:environment", env: Environment | undefined): void;
  (event: "update:search-text", searchText: string): void;
}>();

const changeEnvironmentFilter = (name: string | undefined) => {
  emit("update:instance", undefined);
  if (!isValidEnvironmentName(name)) {
    return emit("update:environment", undefined);
  }
  emit(
    "update:environment",
    useEnvironmentV1Store().getEnvironmentByName(name)
  );
};

const changeInstanceFilter = (name: string | undefined) => {
  if (!isValidInstanceName(name)) {
    return emit("update:instance", undefined);
  }

  emit(
    "update:instance",
    useInstanceResourceByName(name).instance.value as InstanceResource
  );
};
</script>
