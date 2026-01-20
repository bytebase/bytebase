<template>
  <NDataTable
    :columns="columns"
    :data="sequenceList"
    :max-height="640"
    :virtual-scroll="true"
    :striped="true"
    :row-key="
      (sequence: SequenceMetadata) => `${database.name}.${schemaName}.${sequence.name}`
    "
    :loading="loading"
    :bordered="true"
  />
</template>

<script lang="tsx" setup>
import type { DataTableColumn } from "naive-ui";
import { NCheckbox, NDataTable, NPerformantEllipsis } from "naive-ui";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import type {
  Database,
  SequenceMetadata,
} from "@/types/proto-es/v1/database_service_pb";

withDefaults(
  defineProps<{
    database: Database;
    schemaName?: string;
    sequenceList: SequenceMetadata[];
    loading?: boolean;
  }>(),
  {
    schemaName: "",
    loading: false,
  }
);

const { t } = useI18n();

const columns = computed(() => {
  const columns: (DataTableColumn<SequenceMetadata> & {
    hide?: boolean;
  })[] = [
    {
      key: "name",
      title: t("common.name"),
      resizable: true,
      render: (seq) => {
        return (
          <NPerformantEllipsis>
            {{
              default: () => seq.name,
            }}
          </NPerformantEllipsis>
        );
      },
    },
    {
      key: "dataType",
      title: t("db.sequence.data-type"),
      resizable: true,
      render: (seq) => {
        return seq.dataType;
      },
    },
    {
      key: "start",
      title: t("db.sequence.start"),
      resizable: true,
      render: (seq) => {
        return seq.start;
      },
    },
    {
      key: "minValue",
      title: t("db.sequence.min-value"),
      resizable: true,
      render: (seq) => {
        return seq.minValue;
      },
    },
    {
      key: "maxValue",
      title: t("db.sequence.max-value"),
      resizable: true,
      render: (seq) => {
        return (
          <NPerformantEllipsis>
            {{
              default: () => seq.maxValue,
            }}
          </NPerformantEllipsis>
        );
      },
    },
    {
      key: "increment",
      title: t("db.sequence.increment"),
      resizable: true,
      render: (seq) => {
        return seq.increment;
      },
    },
    {
      key: "cycle",
      title: t("db.sequence.cycle"),
      resizable: true,
      render: (seq) => {
        return <NCheckbox checked={seq.cycle} disabled={true} />;
      },
    },
    {
      key: "cacheSize",
      title: t("db.sequence.cacheSize"),
      resizable: true,
      render: (seq) => {
        return seq.cacheSize;
      },
    },
    {
      key: "lastValue",
      title: t("db.sequence.lastValue"),
      resizable: true,
      render: (seq) => {
        return (
          <NPerformantEllipsis>
            {{
              default: () => seq.lastValue,
            }}
          </NPerformantEllipsis>
        );
      },
    },
  ];

  return columns.filter((column) => !column.hide);
});
</script>
