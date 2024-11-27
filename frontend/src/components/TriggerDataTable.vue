<template>
  <NDataTable
    :columns="columns"
    :data="triggerList"
    :max-height="640"
    :virtual-scroll="true"
    :striped="true"
    :bordered="true"
  />
</template>

<script lang="tsx" setup>
import type { DataTableColumn } from "naive-ui";
import { NDataTable, NPerformantEllipsis } from "naive-ui";
import type { PropType } from "vue";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import type { ComposedDatabase } from "@/types";
import type { TriggerMetadata } from "@/types/proto/v1/database_service";
import EllipsisSQLView from "./EllipsisSQLView.vue";

defineProps({
  database: {
    required: true,
    type: Object as PropType<ComposedDatabase>,
  },
  schemaName: {
    type: String,
    default: "",
  },
  triggerList: {
    required: true,
    type: Object as PropType<TriggerMetadata[]>,
  },
});

const { t } = useI18n();

const columns = computed(() => {
  const columns: (DataTableColumn<TriggerMetadata> & { hide?: boolean })[] = [
    {
      key: "name",
      title: t("common.name"),
      resizable: true,
      render: (trigger) => {
        return trigger.name;
      },
    },
    {
      key: "table-name",
      title: t("db.trigger.table-name"),
      resizable: true,
      render: (trigger) => {
        return trigger.tableName;
      },
    },
    {
      key: "event",
      title: t("db.trigger.event"),
      resizable: true,
      render: (trigger) => {
        return trigger.event;
      },
    },
    {
      key: "timing",
      title: t("db.trigger.timing"),
      resizable: true,
      render: (trigger) => {
        return trigger.timing;
      },
    },
    {
      key: "body",
      title: t("db.trigger.body"),
      resizable: true,
      render: (trigger) => {
        return (
          <EllipsisSQLView
            sql={trigger.body}
            lines={1}
            contentStyle="line-height: 25px"
          />
        );
      },
    },
    {
      key: "sql-mode",
      title: "SQL mode",
      resizable: true,
      render: (trigger) => {
        return (
          <NPerformantEllipsis>
            {{
              default: () => trigger.sqlMode,
              tooltip: () => (
                <div
                  class="text-wrap whitespace-pre break-words break-all"
                  style="max-width: calc(min(33vw, 320px))"
                >
                  {trigger.sqlMode.replaceAll(",", ",\n")}
                </div>
              ),
            }}
          </NPerformantEllipsis>
        );
      },
    },
  ];

  return columns.filter((column) => !column.hide);
});
</script>
