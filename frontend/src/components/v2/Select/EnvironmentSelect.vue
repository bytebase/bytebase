<template>
  <NSelect
    v-bind="$attrs"
    :value="value"
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

<script lang="ts" setup>
import type { SelectOption } from "naive-ui";
import { NSelect } from "naive-ui";
import { computed, h, watchEffect } from "vue";
import { useI18n } from "vue-i18n";
import { useEnvironmentV1Store, useProjectV1Store } from "@/store";
import { State } from "@/types/proto/v1/common";
import type { Environment } from "@/types/proto/v1/environment_service";
import { EnvironmentV1Name } from "../Model";

interface EnvironmentSelectOption extends SelectOption {
  value: string;
  environment: Environment;
}

const props = withDefaults(
  defineProps<{
    environment?: string | undefined;
    environments?: string[] | undefined;
    defaultEnvironmentName?: string | undefined;
    includeArchived?: boolean;
    showProductionIcon?: boolean;
    useResourceId?: boolean;
    multiple?: boolean;
    filter?: (environment: Environment, index: number) => boolean;
  }>(),
  {
    environment: undefined,
    environments: undefined,
    defaultEnvironmentName: undefined,
    includeArchived: false,
    showProductionIcon: true,
    useResourceId: false,
    multiple: false,
    filter: () => true,
  }
);

const emit = defineEmits<{
  (event: "update:environment", id: string | undefined): void;
  (event: "update:environments", id: string[]): void;
}>();

const { t } = useI18n();
const projectV1Store = useProjectV1Store();
const environmentV1Store = useEnvironmentV1Store();

const prepare = () => {
  projectV1Store.fetchProjectList(true /* showDeleted */);
};

const value = computed(() => {
  if (props.multiple) {
    return props.environments || [];
  } else {
    return props.environment;
  }
});

const handleValueUpdated = (value: string | string[]) => {
  if (props.multiple) {
    if (!value) {
      // normalize value
      value = [];
    }
    emit("update:environments", value as string[]);
  } else {
    if (value === null) {
      // normalize value
      value = "";
    }
    emit("update:environment", value as string);
  }
};

const rawEnvironmentList = computed(() => {
  const list = environmentV1Store.getEnvironmentList(true /* showDeleted */);
  return list;
});

const getEnvironmentValue = (environment: Environment): string => {
  return props.useResourceId ? environment.name : environment.uid;
};

const combinedEnvironmentList = computed(() => {
  let list = rawEnvironmentList.value.filter((environment) => {
    if (props.includeArchived) return true;
    if (environment.state === State.ACTIVE) return true;
    // ARCHIVED
    if (getEnvironmentValue(environment) === props.environment) return true;
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
        value: getEnvironmentValue(environment),
        label: environment.title,
      };
    }
  );
});

const renderLabel = (option: SelectOption) => {
  const { environment } = option as EnvironmentSelectOption;
  return h(EnvironmentV1Name, {
    environment,
    showIcon: props.showProductionIcon,
    link: false,
    suffix:
      props.defaultEnvironmentName === environment.name
        ? `(${t("common.default")})`
        : "",
  });
};

const filterByName = (pattern: string, option: SelectOption) => {
  const { environment } = option as EnvironmentSelectOption;
  pattern = pattern.toLowerCase();
  return (
    environment.name.toLowerCase().includes(pattern) ||
    environment.title.toLowerCase().includes(pattern)
  );
};

watchEffect(prepare);
</script>
