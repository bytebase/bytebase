<template>
  <NSelect
    v-bind="$attrs"
    :value="environment"
    :options="options"
    :placeholder="$t('environment.select')"
    :filterable="true"
    :filter="filterByName"
    class="bb-environment-select"
    @update:value="$emit('update:environment', $event)"
  />
</template>

<script lang="ts" setup>
import { NSelect, SelectOption } from "naive-ui";
import { computed, watchEffect } from "vue";
import { useEnvironmentV1Store, useProjectV1Store } from "@/store";
import { State } from "@/types/proto/v1/common";
import { Environment } from "@/types/proto/v1/environment_service";

interface EnvironmentSelectOption extends SelectOption {
  value: string;
  environment: Environment;
}

const props = withDefaults(
  defineProps<{
    environment?: string | undefined;
    includeArchived?: boolean;
    filter?: (environment: Environment, index: number) => boolean;
  }>(),
  {
    environment: undefined,
    includeArchived: false,
    filter: () => true,
  }
);

defineEmits<{
  (event: "update:environment", id: string | undefined): void;
}>();

const projectV1Store = useProjectV1Store();
const environmentV1Store = useEnvironmentV1Store();

const prepare = () => {
  projectV1Store.fetchProjectList(true /* showDeleted */);
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
    if (environment.uid === props.environment) return true;
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
        value: environment.uid,
        label: environment.title,
      };
    }
  );
});

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
