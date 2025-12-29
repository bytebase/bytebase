<template>
  <NDataTable
    key="instance-table"
    size="small"
    :columns="columnList"
    :data="instanceList"
    :striped="true"
    :bordered="bordered"
    :loading="loading"
    :row-key="(data: Instance) => data.name"
    :checked-row-keys="selectedInstanceNames"
    :row-props="rowProps"
    :paginate-single-page="false"
    @update:checked-row-keys="
      (val) => $emit('update:selected-instance-names', val as string[])
    "
    @update:sorter="$emit('update:sorters', $event)"
  />
</template>

<script setup lang="tsx">
import {
  ChevronDownIcon,
  ChevronUpIcon,
  ExternalLinkIcon,
} from "lucide-vue-next";
import {
  type DataTableColumn,
  type DataTableSortState,
  NButton,
  NDataTable,
} from "naive-ui";
import { computed, reactive, watch } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import EllipsisText from "@/components/EllipsisText.vue";
import { InstanceV1Name } from "@/components/v2";
import { LabelsCell } from "@/components/v2/Model/cells";
import { useEnvironmentV1Store } from "@/store";
import { NULL_ENVIRONMENT_NAME } from "@/types";
import type { Instance } from "@/types/proto-es/v1/instance_service_pb";
import { hostPortOfDataSource, hostPortOfInstanceV1, urlfy } from "@/utils";
import EnvironmentV1Name from "../../EnvironmentV1Name.vue";
import { mapSorterStatus } from "../../utils";

type InstanceDataTableColumn = DataTableColumn<Instance> & {
  hide?: boolean;
};

interface LocalState {
  dataSourceToggle: Set<string>;
  processing: boolean;
}

const props = withDefaults(
  defineProps<{
    instanceList: Instance[];
    bordered?: boolean;
    loading?: boolean;
    showSelection?: boolean;
    defaultExpandDataSource?: string[];
    selectedInstanceNames?: string[];
    disabledSelectedInstanceNames?: Set<string>;
    showAddress?: boolean;
    showExternalLink?: boolean;
    onClick?: (instance: Instance, e: MouseEvent) => void;
    sorters?: DataTableSortState[];
  }>(),
  {
    bordered: true,
    showSelection: true,
    onClick: undefined,
    showAddress: true,
    showExternalLink: true,
    disabledSelectedInstanceNames: () => new Set(),
    defaultExpandDataSource: () => [],
    selectedInstanceNames: () => [],
  }
);

defineEmits<{
  (event: "update:selected-instance-names", val: string[]): void;
  (event: "update:sorters", sorters: DataTableSortState[]): void;
}>();

const { t } = useI18n();
const router = useRouter();
const environmentStore = useEnvironmentV1Store();
const state = reactive<LocalState>({
  dataSourceToggle: new Set(),
  processing: false,
});

watch(
  () => props.defaultExpandDataSource,
  () => {
    state.dataSourceToggle = new Set(props.defaultExpandDataSource);
  },
  {
    immediate: true,
    deep: true,
  }
);

const columnList = computed((): InstanceDataTableColumn[] => {
  const SELECTION: InstanceDataTableColumn = {
    type: "selection",
    hide: !props.showSelection,
    disabled: (instance) =>
      props.disabledSelectedInstanceNames.has(instance.name),
    cellProps: () => {
      return {
        onClick: (e: MouseEvent) => {
          e.stopPropagation();
        },
      };
    },
  };
  const NAME: InstanceDataTableColumn = {
    key: "title",
    title: t("common.name"),
    resizable: true,
    render: (instance) => {
      return <InstanceV1Name instance={instance} link={false} />;
    },
  };
  const ENVIRONMENT: InstanceDataTableColumn = {
    key: "environment",
    title: t("common.environment"),
    className: "whitespace-nowrap",
    resizable: true,
    ellipsis: {
      tooltip: true,
    },
    minWidth: 300,
    render: (instance) => (
      <EnvironmentV1Name
        environment={environmentStore.getEnvironmentByName(
          instance.environment || NULL_ENVIRONMENT_NAME
        )}
        link={false}
        showColor={true}
        nullEnvironmentPlaceholder="Null"
      />
    ),
  };
  const ADDRESS: InstanceDataTableColumn = {
    key: "address",
    title: t("common.address"),
    resizable: true,
    hide: !props.showAddress,
    render: (instance) => {
      return (
        <div class={"flex items-start gap-x-2"}>
          <EllipsisText>
            {state.dataSourceToggle.has(instance.name)
              ? instance.dataSources.map((ds) => (
                  <div>{hostPortOfDataSource(ds)}</div>
                ))
              : hostPortOfInstanceV1(instance)}
          </EllipsisText>
          {instance.dataSources.length > 1 ? (
            <NButton
              quaternary
              size="tiny"
              onClick={(e) => handleDataSourceToggle(e, instance)}
            >
              {state.dataSourceToggle.has(instance.name) ? (
                <ChevronUpIcon class={"w-4 cursor-pointer"} />
              ) : (
                <ChevronDownIcon class={"w-4 cursor-pointer"} />
              )}
            </NButton>
          ) : null}
        </div>
      );
    },
  };
  const EXTERNAL_LINK: InstanceDataTableColumn = {
    key: "project",
    title: t("instance.external-link"),
    resizable: true,
    width: 150,
    hide: !props.showExternalLink,
    render: (instance) =>
      instance.externalLink?.trim().length !== 0 && (
        <NButton
          quaternary
          size="small"
          onClick={() => window.open(urlfy(instance.externalLink), "_blank")}
        >
          <ExternalLinkIcon class="w-4 h-4" />
        </NButton>
      ),
  };
  const LABELS: InstanceDataTableColumn = {
    key: "labels",
    title: t("common.labels"),
    resizable: true,
    width: 300,
    render: (instance) => (
      <LabelsCell labels={instance.labels} showCount={3} placeholder="-" />
    ),
  };
  const LICENSE: InstanceDataTableColumn = {
    key: "instance",
    title: t("subscription.instance-assignment.license"),
    resizable: true,
    width: 100,
    render: (instance) => (instance.activation ? "Y" : ""),
  };

  const columns: InstanceDataTableColumn[] = [
    SELECTION,
    NAME,
    ENVIRONMENT,
    ADDRESS,
    LABELS,
    EXTERNAL_LINK,
    LICENSE,
  ].filter((column) => !column.hide);
  return mapSorterStatus(columns, props.sorters);
});

const handleDataSourceToggle = (e: MouseEvent, instance: Instance) => {
  e.stopPropagation();
  if (state.dataSourceToggle.has(instance.name)) {
    state.dataSourceToggle.delete(instance.name);
  } else {
    state.dataSourceToggle.add(instance.name);
  }
};

const rowProps = (instance: Instance) => {
  return {
    style: "cursor: pointer;",
    onClick: (e: MouseEvent) => {
      if (props.onClick) {
        props.onClick(instance, e);
        return;
      }
      const url = `/${instance.name}`;
      if (e.ctrlKey || e.metaKey) {
        window.open(url, "_blank");
      } else {
        router.push(url);
      }
    },
  };
};
</script>
