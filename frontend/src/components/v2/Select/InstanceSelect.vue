<template>
  <RemoteResourceSelector
    v-bind="$attrs"
    :value="instanceName"
    :custom-label="renderLabel"
    :consistent-menu-width="false"
    class="bb-instance-select"
    :additional-data="additionalData"
    :search="handleSearch"
    :get-option="getOption"
    @update:value="(val) => $emit('update:instance-name', val)"
  />
</template>

<script lang="ts" setup>
import { computedAsync } from "@vueuse/core";
import { h } from "vue";
import { useI18n } from "vue-i18n";
import { useInstanceV1Store } from "@/store";
import {
  isValidInstanceName,
  UNKNOWN_INSTANCE_NAME,
  unknownInstance,
} from "@/types";
import { type Engine } from "@/types/proto-es/v1/common_pb";
import type { Instance } from "@/types/proto-es/v1/instance_service_pb";
import { supportedEngineV1List } from "@/utils";
import { InstanceV1EngineIcon } from "../Model/Instance";
import RemoteResourceSelector from "./RemoteResourceSelector.vue";

const props = withDefaults(
  defineProps<{
    instanceName?: string | undefined;
    environmentName?: string;
    projectName?: string;
    allowedEngineList?: Engine[];
  }>(),
  {
    instanceName: undefined,
    environmentName: undefined,
    allowedEngineList: () => supportedEngineV1List(),
  }
);

const emit = defineEmits<{
  (event: "update:instance-name", value: string | undefined): void;
}>();

const { t } = useI18n();
const instanceStore = useInstanceV1Store();

const additionalData = computedAsync(async () => {
  const data = [];
  if (props.instanceName === UNKNOWN_INSTANCE_NAME) {
    const dummyAll = {
      ...unknownInstance(),
      title: t("instance.all"),
    };
    data.push(dummyAll);
  }

  if (isValidInstanceName(props.instanceName)) {
    const instance = await instanceStore.getOrFetchInstanceByName(
      props.instanceName
    );
    data.push(instance);
  }
  return data;
}, []);

const handleSearch = async (params: {
  search: string;
  pageToken: string;
  pageSize: number;
}) => {
  const { instances, nextPageToken } = await instanceStore.fetchInstanceList({
    pageToken: params.pageToken,
    pageSize: params.pageSize,
    filter: {
      engines: props.allowedEngineList,
      query: params.search,
      environment: props.environmentName,
      project: props.projectName,
    },
  });

  return {
    nextPageToken,
    data: instances,
  };
};

const renderLabel = (instance: Instance) => {
  if (instance.name === UNKNOWN_INSTANCE_NAME) {
    return t("instance.all");
  }
  const icon = h(InstanceV1EngineIcon, {
    instance,
    class: "bb-instance-select--engine-icon shrink-0",
  });
  const text = h("span", {}, instance.title);
  return h(
    "div",
    {
      class: "flex items-center gap-x-2",
    },
    [icon, text]
  );
};

const getOption = (instance: Instance) => ({
  value: instance.name,
  label: instance.title,
});
</script>

<style lang="postcss" scoped>
.bb-instance-select
  :deep(.n-base-selection--active .bb-instance-select--engine-icon) {
  opacity: 0.3;
}
</style>
