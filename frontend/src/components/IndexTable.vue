<template>
  <div v-for="(section, sectionIndex) in sectionList" :key="sectionIndex">
    <h3 class="text-left pl-4 pt-4 pb-2 text-base font-medium text-gray-900">
      {{ section.title }}
    </h3>
    <NDataTable
      size="small"
      :columns="columns"
      :data="section.list"
      :striped="true"
      :bordered="true"
    />
  </div>
</template>

<script lang="ts" setup>
import type { DataTableColumn } from "naive-ui";
import { NDataTable } from "naive-ui";
import type { PropType } from "vue";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { Engine } from "@/types/proto-es/v1/common_pb";
import type {
  Database,
  IndexMetadata,
} from "@/types/proto-es/v1/database_service_pb";
import { getInstanceResource } from "@/utils";

const props = defineProps({
  database: {
    required: true,
    type: Object as PropType<Database>,
  },
  indexList: {
    required: true,
    type: Object as PropType<IndexMetadata[]>,
  },
});

const { t } = useI18n();
const showVisibleColumn = computed(() => {
  const engine = getInstanceResource(props.database).engine;
  return engine !== Engine.POSTGRES && engine !== Engine.MONGODB;
});
const showCommentColumn = computed(() => {
  const engine = getInstanceResource(props.database).engine;
  return engine !== Engine.MONGODB;
});
const columns = computed((): DataTableColumn<IndexMetadata>[] => {
  const cols: DataTableColumn<IndexMetadata>[] = [
    {
      title: t("database.expression"),
      key: "expressions",
      render: (row) => row.expressions.join(","),
    },
  ];

  cols.push({
    title: t("database.unique"),
    key: "unique",
    render: (row) => String(row.unique),
  });

  if (showVisibleColumn.value) {
    cols.push({
      title: t("database.visible"),
      key: "visible",
      render: (row) => String(row.visible),
    });
  }

  if (showCommentColumn.value) {
    cols.push({
      title: t("database.comment"),
      key: "comment",
      render: (row) => row.comment || "-",
    });
  }

  return cols;
});

const sectionList = computed(() => {
  const sections: { title: string; list: IndexMetadata[] }[] = [];

  for (const index of props.indexList) {
    const item = sections.find((item) => item.title == index.name);
    if (item) {
      item.list.push(index);
    } else {
      sections.push({
        title: index.name,
        list: [index],
      });
    }
  }

  return sections;
});
</script>
