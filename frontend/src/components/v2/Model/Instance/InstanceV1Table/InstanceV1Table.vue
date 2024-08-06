<template>
  <div class="space-y-2">
    <InstanceOperations
      v-if="showOperation"
      :instance-list="selectedInstanceList"
    />

    <NDataTable
      key="instance-table"
      size="small"
      :columns="columnList"
      :data="instanceList"
      :striped="true"
      :bordered="bordered"
      :loading="loading"
      :row-key="(data: ComposedInstance) => data.name"
      :checked-row-keys="Array.from(state.selectedInstance)"
      :row-props="rowProps"
      :pagination="{ pageSize: 20 }"
      :paginate-single-page="false"
      @update:checked-row-keys="
        (val) => (state.selectedInstance = new Set(val as string[]))
      "
    />
  </div>
</template>

<script setup lang="tsx">
import { ExternalLinkIcon } from "lucide-vue-next";
import { NButton, NDataTable, type DataTableColumn } from "naive-ui";
import { computed, reactive } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { InstanceV1Name } from "@/components/v2";
import type { ComposedInstance } from "@/types";
import { urlfy, hostPortOfInstanceV1 } from "@/utils";
import EnvironmentV1Name from "../../EnvironmentV1Name.vue";
import InstanceOperations from "./InstanceOperations.vue";

type InstanceDataTableColumn = DataTableColumn<ComposedInstance> & {
  hide?: boolean;
};

interface LocalState {
  selectedInstance: Set<string>;
  processing: boolean;
}

const props = withDefaults(
  defineProps<{
    instanceList: ComposedInstance[];
    bordered?: boolean;
    loading?: boolean;
    showSelection?: boolean;
    showOperation?: boolean;
    canAssignLicense?: boolean;
    onClick?: (instance: ComposedInstance, e: MouseEvent) => void;
  }>(),
  {
    bordered: true,
    showSelection: true,
    showOperation: true,
    canAssignLicense: true,
    onClick: undefined,
  }
);

const { t } = useI18n();
const router = useRouter();
const state = reactive<LocalState>({
  selectedInstance: new Set(),
  processing: false,
});

const columnList = computed((): InstanceDataTableColumn[] => {
  const SELECTION: InstanceDataTableColumn = {
    type: "selection",
    hide: !props.showSelection,
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
    render: (instance) => (
      <EnvironmentV1Name
        environment={instance.environmentEntity}
        link={false}
      />
    ),
  };
  const ADDRESS: InstanceDataTableColumn = {
    key: "address",
    title: t("common.address"),
    resizable: true,
    render: (instance) => hostPortOfInstanceV1(instance),
  };
  const EXTERNAL_LINK: InstanceDataTableColumn = {
    key: "project",
    title: t("instance.external-link"),
    resizable: true,
    width: 150,
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
  const LICENSE: InstanceDataTableColumn = {
    key: "instance",
    title: t("subscription.instance-assignment.license"),
    resizable: true,
    width: 100,
    render: (instance) => (instance.activation ? "Y" : ""),
  };

  return [SELECTION, NAME, ENVIRONMENT, ADDRESS, EXTERNAL_LINK, LICENSE].filter(
    (column) => !column.hide
  );
});

const rowProps = (instance: ComposedInstance) => {
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

const selectedInstanceList = computed(() => {
  return props.instanceList.filter((instance) =>
    state.selectedInstance.has(instance.name)
  );
});
</script>
