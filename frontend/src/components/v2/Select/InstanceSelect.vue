<template>
  <RemoteResourceSelector
    v-bind="$attrs"
    :multiple="false"
    :disabled="disabled"
    :size="size"
    :value="value"
    :render-label="renderLabel"
    :render-tag="renderTag"
    :consistent-menu-width="false"
    class="bb-instance-select"
    :additional-options="additionalOptions"
    :search="handleSearch"
    @update:value="(val) => $emit('update:value', val as (string | undefined))"
  />
</template>

<script lang="tsx" setup>
import { computedAsync } from "@vueuse/core";
import { ChevronRightIcon } from "lucide-vue-next";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { EnvironmentV1Name, InstanceV1Name } from "@/components/v2";
import { useEnvironmentV1Store, useInstanceV1Store } from "@/store";
import {
  isValidInstanceName,
  UNKNOWN_INSTANCE_NAME,
  unknownInstance,
} from "@/types";
import { type Engine } from "@/types/proto-es/v1/common_pb";
import type { Instance } from "@/types/proto-es/v1/instance_service_pb";
import { supportedEngineV1List } from "@/utils";
import RemoteResourceSelector from "./RemoteResourceSelector/index.vue";
import type {
  ResourceSelectOption,
  SelectSize,
} from "./RemoteResourceSelector/types";
import {
  getRenderLabelFunc,
  getRenderTagFunc,
} from "./RemoteResourceSelector/utils";

const props = withDefaults(
  defineProps<{
    value?: string | undefined;
    environmentName?: string;
    projectName?: string;
    allowedEngineList?: Engine[];
    disabled?: boolean;
    size?: SelectSize;
  }>(),
  {
    allowedEngineList: () => supportedEngineV1List(),
  }
);

defineEmits<{
  (event: "update:value", value: string | undefined): void;
}>();

const { t } = useI18n();
const instanceStore = useInstanceV1Store();
const environmentStore = useEnvironmentV1Store();

const getOption = (instance: Instance): ResourceSelectOption<Instance> => ({
  resource: instance,
  value: instance.name,
  label: instance.title,
});

const additionalOptions = computedAsync(async () => {
  const options: ResourceSelectOption<Instance>[] = [];
  if (props.value === UNKNOWN_INSTANCE_NAME) {
    const dummyAll = {
      ...unknownInstance(),
      title: t("instance.all"),
    };
    options.push(getOption(dummyAll));
  }

  if (isValidInstanceName(props.value)) {
    const instance = await instanceStore.getOrFetchInstanceByName(props.value);
    options.push(getOption(instance));
  }
  return options;
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
    options: instances.map(getOption),
  };
};

const customLabel = (instance: Instance, keyword: string) => {
  const isUnknown = instance.name === UNKNOWN_INSTANCE_NAME;
  const environment = environmentStore.getEnvironmentByName(
    instance.environment ?? ""
  );

  return (
    <div class="flex items-center gap-x-1">
      {isUnknown ? null : (
        <EnvironmentV1Name
          environment={environment}
          plain={true}
          link={false}
        />
      )}
      {isUnknown ? null : <ChevronRightIcon class="w-3" />}
      <InstanceV1Name
        instance={instance}
        keyword={keyword}
        plain={true}
        link={false}
      />
    </div>
  );
};

const renderLabel = computed(() => {
  return getRenderLabelFunc({
    multiple: false,
    customLabel,
    showResourceName: true,
  });
});

const renderTag = computed(() => {
  return getRenderTagFunc({
    multiple: false,
    disabled: props.disabled,
    size: props.size,
    customLabel,
  });
});
</script>

<style lang="postcss" scoped>
.bb-instance-select
  :deep(.n-base-selection--active .bb-instance-select--engine-icon) {
  opacity: 0.3;
}
</style>
