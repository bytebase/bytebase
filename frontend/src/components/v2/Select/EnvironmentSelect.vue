<template>
  <LocalResourceSelector
    v-bind="$attrs"
    :placeholder="t('environment.select')"
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
import { computed, type VNodeChild } from "vue";
import { useI18n } from "vue-i18n";
import { useEnvironmentV1Store } from "@/store";
import { formatEnvironmentName } from "@/types";
import type { Environment } from "@/types/v1/environment";
import { EnvironmentV1Name } from "../Model";
import LocalResourceSelector from "./LocalResourceSelector.vue";

const { t } = useI18n();

const props = withDefaults(
  defineProps<{
    environmentName?: string | undefined;
    environmentNames?: string[] | undefined;
    includeArchived?: boolean;
    showProductionIcon?: boolean;
    multiple?: boolean;
    filter?: (environment: Environment, index: number) => boolean;
    renderSuffix?: (environment: string) => VNodeChild;
  }>(),
  {
    environmentName: undefined,
    environmentNames: undefined,
    includeArchived: false,
    showProductionIcon: true,
    multiple: false,
    filter: () => true,
    renderSuffix: () => null,
  }
);

defineEmits<{
  (event: "update:environment-name", name: string | undefined): void;
  (event: "update:environment-names", names: string[]): void;
}>();
const environmentV1Store = useEnvironmentV1Store();

const rawEnvironmentList = computed(() => {
  const list = environmentV1Store.getEnvironmentList();
  return list;
});

const combinedEnvironmentList = computed(() => {
  let list = rawEnvironmentList.value;
  if (props.filter) {
    list = list.filter(props.filter);
  }

  return list;
});

const options = computed(() => {
  return combinedEnvironmentList.value.map((environment) => {
    return {
      resource: {
        ...environment,
        name: formatEnvironmentName(environment.id),
      },
      value: formatEnvironmentName(environment.id),
      label: environment.title,
    };
  });
});

const renderLabel = (environment: Environment) => {
  return (
    <div class="flex items-center gap-x-2">
      <EnvironmentV1Name
        environment={environment}
        showIcon={props.showProductionIcon}
        link={false}
      />
      {props.renderSuffix(environment.name)}
    </div>
  );
};
</script>
