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
      />
    </template>
  </PagedTable>
</template>

<script lang="tsx" setup>
import { ref, watch, computed } from "vue";
import type { ComponentExposed } from "vue-component-type-helpers";
import PagedTable from "@/components/v2/Model/PagedTable.vue";
import { useInstanceV1Store, type InstanceFilter } from "@/store";
import { type Instance } from "@/types/proto-es/v1/instance_service_pb";
import InstanceV1Table from "./InstanceV1Table";

const props = defineProps<{
  filter?: InstanceFilter;
  sessionKey: string;
}>();

const instanceStore = useInstanceV1Store();

const instancePagedTable = ref<ComponentExposed<typeof PagedTable<Instance>>>();

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
  });
  return {
    nextPageToken: nextPageToken ?? "",
    list: instances,
  };
};

watch(
  () => props.filter,
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
