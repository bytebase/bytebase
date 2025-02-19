<template>
  <ResourceSelect
    v-bind="$attrs"
    :placeholder="$t('environment.select')"
    :multiple="multiple"
    :value="environmentName"
    :values="environmentNames"
    :options="options"
    :custom-label="renderLabel"
    class="bb-environment-select"
    @update:value="(val) => $emit('update:environment-name', val)"
    @update:values="(val) => $emit('update:environment-names', val)"
  />
</template>

<script lang="tsx" setup>
import { computed } from "vue";
import { useEnvironmentV1Store } from "@/store";
import { State } from "@/types/proto/v1/common";
import type { Environment } from "@/types/proto/v1/environment_service";
import { EnvironmentV1Name } from "../Model";
import ResourceSelect from "./ResourceSelect.vue";

const props = withDefaults(
  defineProps<{
    environmentName?: string | undefined;
    environmentNames?: string[] | undefined;
    includeArchived?: boolean;
    showProductionIcon?: boolean;
    multiple?: boolean;
    filter?: (environment: Environment, index: number) => boolean;
    renderSuffix?: (environment: string) => string;
  }>(),
  {
    environmentName: undefined,
    environmentNames: undefined,
    includeArchived: false,
    showProductionIcon: true,
    multiple: false,
    filter: () => true,
    renderSuffix: () => "",
  }
);

defineEmits<{
  (event: "update:environment-name", name: string | undefined): void;
  (event: "update:environment-names", names: string[]): void;
}>();
const environmentV1Store = useEnvironmentV1Store();

const rawEnvironmentList = computed(() => {
  const list = environmentV1Store.getEnvironmentList(true /* showDeleted */);
  return list;
});

const combinedEnvironmentList = computed(() => {
  let list = rawEnvironmentList.value.filter((environment) => {
    if (props.includeArchived) return true;
    if (environment.state === State.ACTIVE) return true;
    // ARCHIVED
    if (environment.name === props.environmentName) return true;
    return false;
  });

  if (props.filter) {
    list = list.filter(props.filter);
  }

  return list;
});

const options = computed(() => {
  return combinedEnvironmentList.value.map((environment) => {
    return {
      resource: environment,
      value: environment.name,
      label: environment.title,
    };
  });
});

const renderLabel = (environment: Environment) => {
  return (
    <EnvironmentV1Name
      environment={environment}
      showIcon={props.showProductionIcon}
      link={false}
    />
  );
};
</script>
