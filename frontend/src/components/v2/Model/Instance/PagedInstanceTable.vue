<template>
  <PagedTable
    ref="instancePagedTable"
    :session-key="sessionKey"
    :fetch-list="fetchInstances"
  >
    <template #table="{ list, loading }">
      <InstanceV1Table
        v-bind="$attrs"
        :loading="loading"
        :instance-list="list"
        :sorters="sorters"
        @update:sorters="onSorterUpdate"
      />
    </template>
  </PagedTable>
</template>

<script lang="tsx" setup>
import { type DataTableSortState } from "naive-ui";
import { computed, ref, watch } from "vue";
import type { ComponentExposed } from "vue-component-type-helpers";
import PagedTable from "@/components/v2/Model/PagedTable.vue";
import { type InstanceFilter, useInstanceV1Store } from "@/store";
import { type Instance } from "@/types/proto-es/v1/instance_service_pb";
import InstanceV1Table from "./InstanceV1Table";

const props = defineProps<{
  filter?: InstanceFilter;
  sessionKey: string;
}>();

const instanceStore = useInstanceV1Store();
const instancePagedTable = ref<ComponentExposed<typeof PagedTable<Instance>>>();

const sorters = ref<DataTableSortState[]>([
  {
    columnKey: "title",
    order: false,
    sorter: true,
  },
  {
    columnKey: "environment",
    order: false,
    sorter: true,
  },
]);

const onSorterUpdate = (sortStates: DataTableSortState[]) => {
  for (const sortState of sortStates) {
    const sorterIndex = sorters.value.findIndex(
      (s) => s.columnKey === sortState.columnKey
    );
    if (sorterIndex >= 0) {
      sorters.value[sorterIndex] = sortState;
    }
  }
};

const orderBy = computed(() => {
  return sorters.value
    .filter((sorter) => sorter.order)
    .map((sorter) => {
      const key = sorter.columnKey.toString();
      const order = sorter.order == "ascend" ? "asc" : "desc";
      return `${key} ${order}`;
    })
    .join(", ");
});

const fetchInstances = async ({
  pageToken,
  pageSize,
}: {
  pageToken: string;
  pageSize: number;
}) => {
  const { nextPageToken, instances } = await instanceStore.fetchInstanceList({
    pageToken,
    pageSize,
    filter: props.filter,
    orderBy: orderBy.value,
  });
  return {
    nextPageToken: nextPageToken ?? "",
    list: instances,
  };
};

watch(
  [() => props.filter, () => orderBy.value],
  () => instancePagedTable.value?.refresh(),
  { deep: true }
);

defineExpose({
  updateCache: (instances: Instance[]) => {
    instancePagedTable.value?.updateCache(instances);
  },
  refresh: () => instancePagedTable.value?.refresh(),
  dataList: computed(() => instancePagedTable.value?.dataList),
});
</script>
