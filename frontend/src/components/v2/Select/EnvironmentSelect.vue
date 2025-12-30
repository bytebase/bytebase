<template>
  <NSelect
    v-bind="$attrs"
    :filterable="true"
    :virtual-scroll="true"
    :placeholder="t('environment.select')"
    :multiple="multiple"
    :disabled="disabled"
    :size="size"
    :value="value"
    :options="options"
    :render-label="renderLabel"
    :render-tag="renderTag"
    :filter="filterEnvironment"
    :consistent-menu-width="true"
    class="bb-environment-select"
    @update:value="(val) => $emit('update:value', val)"
  />
</template>

<script lang="tsx" setup>
import type { SelectOption } from "naive-ui";
import { NSelect } from "naive-ui";
import type { SelectBaseOption } from "naive-ui/lib/select/src/interface";
import { computed, type VNodeChild } from "vue";
import { useI18n } from "vue-i18n";
import { useEnvironmentV1Store } from "@/store";
import { formatEnvironmentName } from "@/types";
import type { Environment } from "@/types/v1/environment";
import { EnvironmentV1Name } from "../Model";
import type {
  ResourceSelectOption,
  SelectSize,
} from "./RemoteResourceSelector/types";
import {
  getRenderLabelFunc,
  getRenderTagFunc,
} from "./RemoteResourceSelector/utils";

const { t } = useI18n();

const props = withDefaults(
  defineProps<{
    value?: string[] | string | undefined;
    includeArchived?: boolean;
    showProductionIcon?: boolean;
    multiple?: boolean;
    disabled?: boolean;
    size?: SelectSize;
    filter?: (environment: Environment) => boolean;
    renderSuffix?: (environment: Environment) => VNodeChild;
  }>(),
  {
    showProductionIcon: true,
  }
);

defineEmits<{
  (event: "update:value", name: string[] | string | undefined): void;
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
  return combinedEnvironmentList.value.map<ResourceSelectOption<Environment>>(
    (environment) => {
      return {
        resource: environment,
        value: formatEnvironmentName(environment.id),
        label: environment.title,
      };
    }
  );
});

const customLabel = (environment: Environment) => {
  return (
    <div class="flex items-center gap-x-2">
      <EnvironmentV1Name
        environment={environment}
        showIcon={props.showProductionIcon}
        link={false}
      />
      {props.renderSuffix ? props.renderSuffix(environment) : null}
    </div>
  );
};

const renderLabel = (option: SelectOption, selected: boolean) =>
  getRenderLabelFunc({
    multiple: props.multiple,
    customLabel,
    showResourceName: true,
  })(option as ResourceSelectOption<Environment>, selected, "");

const renderTag = ({
  option,
  handleClose,
}: {
  option: SelectBaseOption;
  handleClose: () => void;
}) => {
  return getRenderTagFunc({
    multiple: props.multiple,
    disabled: props.disabled,
    size: props.size,
    customLabel,
  })({
    option: option as ResourceSelectOption<Environment>,
    handleClose,
  });
};

const filterEnvironment = (pattern: string, option: SelectOption) => {
  const { value, label } = option;
  const search = pattern.trim().toLowerCase();
  return (
    (value as string).toLowerCase().includes(search) ||
    (label as string).toLowerCase().includes(search)
  );
};
</script>
