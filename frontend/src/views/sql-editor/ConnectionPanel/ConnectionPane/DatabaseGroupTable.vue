<template>
  <div
    class="w-full"
  >
    <div class="w-full flex flex-row justify-between items-center mb-3">
      <p class="textinfolabel">
        {{ $t("database-group.select") }}
      </p>
      <SearchBox
        v-model:value="search"
        :placeholder="$t('common.filter-by-name')"
      />
    </div>
    <DatabaseGroupDataTable
      :database-group-list="filteredDbGroupList"
      :single-selection="false"
      :show-selection="true"
      :show-external-link="true"
      :page-size="100"
      :loading="!ready"
      :selected-database-group-names="databaseGroupNames"
      @update:selected-database-group-names="(groups) => $emit('update:databaseGroupNames', groups)"
    />
  </div>
</template>

<script lang="tsx" setup>
import { computed, ref } from "vue";
import DatabaseGroupDataTable from "@/components/DatabaseGroup/DatabaseGroupDataTable.vue";
import { SearchBox } from "@/components/v2";
import { useDBGroupListByProject, useSQLEditorStore } from "@/store/modules";
import { DatabaseGroupView } from "@/types/proto-es/v1/database_group_service_pb";

defineProps<{
  databaseGroupNames: string[];
}>();

defineEmits<{
  (
    event: "update:databaseGroupNames",
    databaseGroupNames: string[]
  ): Promise<void>;
}>();

const editorStore = useSQLEditorStore();

const search = ref("");

const { dbGroupList, ready } = useDBGroupListByProject(
  computed(() => editorStore.project),
  DatabaseGroupView.FULL
);

const filteredDbGroupList = computed(() => {
  const filter = search.value.trim().toLowerCase();
  if (!filter) {
    return dbGroupList.value;
  }
  return dbGroupList.value.filter((group) => {
    return (
      group.name.toLowerCase().includes(filter) ||
      group.title.toLowerCase().includes(filter)
    );
  });
});
</script>