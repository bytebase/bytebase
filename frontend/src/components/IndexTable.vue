<template>
  <BBTable
    :column-list="columnList"
    :section-data-source="sectionList"
    :show-header="true"
    :compact-section="false"
  >
    <template #header>
      <BBTableHeaderCell
        :left-padding="4"
        class="w-16"
        :title="columnList[0].title"
      />
      <BBTableHeaderCell
        v-if="showPositionColumn"
        class="w-4"
        :title="columnList[1].title"
      />
      <BBTableHeaderCell class="w-4" :title="columnList[2].title" />
      <BBTableHeaderCell
        v-if="showVisibleColumn"
        class="w-4"
        :title="columnList[3].title"
      />
      <BBTableHeaderCell
        v-if="showCommentColumn"
        class="w-16"
        :title="columnList[4].title"
      />
    </template>
    <template #body="{ rowData: index }">
      <BBTableCell :left-padding="4">
        {{ index.expressions.join(",") }}
      </BBTableCell>
      <BBTableCell v-if="showPositionColumn">
        {{ index.position }}
      </BBTableCell>
      <BBTableCell>
        {{ index.unique }}
      </BBTableCell>
      <BBTableCell v-if="showVisibleColumn">
        {{ index.visible }}
      </BBTableCell>
      <BBTableCell v-if="showCommentColumn">
        {{ index.comment }}
      </BBTableCell>
    </template>
  </BBTable>
</template>

<script lang="ts" setup>
import type { PropType } from "vue";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { BBTable, BBTableCell, BBTableHeaderCell } from "@/bbkit";
import type { BBTableSectionDataSource } from "@/bbkit/types";
import type { ComposedDatabase } from "@/types";
import { Engine } from "@/types/proto/v1/common";
import type { IndexMetadata } from "@/types/proto/v1/database_service";

const props = defineProps({
  database: {
    required: true,
    type: Object as PropType<ComposedDatabase>,
  },
  indexList: {
    required: true,
    type: Object as PropType<IndexMetadata[]>,
  },
});

const { t } = useI18n();
const showVisibleColumn = computed(() => {
  return (
    props.database.instanceResource.engine !== Engine.POSTGRES &&
    props.database.instanceResource.engine !== Engine.MONGODB
  );
});
const showPositionColumn = computed(() => {
  return props.database.instanceResource.engine !== Engine.MONGODB;
});
const showCommentColumn = computed(() => {
  return props.database.instanceResource.engine !== Engine.MONGODB;
});
const columnList = computed(() => [
  {
    title: t("database.expression"),
  },
  {
    title: t("database.position"),
  },
  {
    title: t("database.unique"),
  },
  {
    title: t("database.visible"),
  },
  {
    title: t("database.comment"),
  },
]);
const sectionList = computed(() => {
  const sectionList: BBTableSectionDataSource<IndexMetadata>[] = [];

  for (const index of props.indexList) {
    const item = sectionList.find((item) => item.title == index.name);
    if (item) {
      item.list.push(index);
    } else {
      sectionList.push({
        title: index.name,
        list: [index],
      });
    }
  }

  return sectionList;
});
</script>
