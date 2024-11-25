<template>
  <NSelect
    v-bind="$attrs"
    :value="combinedValue"
    :options="options"
    :placeholder="$t('environment.select')"
    :filterable="true"
    :multiple="multiple"
    :filter="filterByName"
    :render-label="renderLabel"
    class="bb-environment-select"
    @update:value="handleValueUpdated"
  />
</template>

<script lang="tsx" setup>
import type { SelectOption } from "naive-ui";
import { NSelect } from "naive-ui";
import { computed } from "vue";
import { useEnvironmentV1Store } from "@/store";
import { State } from "@/types/proto/v1/common";
import type { Environment } from "@/types/proto/v1/environment_service";
import { EnvironmentV1Name } from "../Model";

interface EnvironmentSelectOption extends SelectOption {
  value: string;
  environment: Environment;
}

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

const emit = defineEmits<{
  (event: "update:environment-name", name: string | undefined): void;
  (event: "update:environment-names", names: string[]): void;
}>();
const environmentV1Store = useEnvironmentV1Store();

const combinedValue = computed(() => {
  if (props.multiple) {
    return props.environmentNames || [];
  } else {
    return props.environmentName;
  }
});

const handleValueUpdated = (value: string | string[]) => {
  if (props.multiple) {
    if (!value) {
      // normalize value
      value = [];
    }
    emit("update:environment-names", value as string[]);
  } else {
    if (value === null) {
      // normalize value
      value = "";
    }
    emit("update:environment-name", value as string);
  }
};

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
  return combinedEnvironmentList.value.map<EnvironmentSelectOption>(
    (environment) => {
      return {
        environment,
        value: environment.name,
        label: environment.title,
      };
    }
  );
});

const renderLabel = (option: SelectOption) => {
  const { environment } = option as EnvironmentSelectOption;

  return (
    <EnvironmentV1Name
      environment={environment}
      showIcon={props.showProductionIcon}
      link={false}
    >
      {{
        suffix: () => (
          <span class="opacity-60 ml-1">
            {props.renderSuffix(environment.name)}
          </span>
        ),
      }}
    </EnvironmentV1Name>
  );
};

const filterByName = (pattern: string, option: SelectOption) => {
  const { environment } = option as EnvironmentSelectOption;
  pattern = pattern.toLowerCase();
  return (
    environment.name.toLowerCase().includes(pattern) ||
    environment.title.toLowerCase().includes(pattern)
  );
};
</script>
