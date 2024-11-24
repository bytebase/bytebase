<template>
  <NDataTable
    :columns="columns"
    :data="sequenceList"
    :max-height="640"
    :virtual-scroll="true"
    :striped="true"
    :bordered="true"
  />
</template>

<script lang="tsx" setup>
import type { DataTableColumn } from "naive-ui";
import { NCheckbox, NDataTable, NPerformantEllipsis } from "naive-ui";
import type { PropType } from "vue";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import type { ComposedDatabase } from "@/types";
import type { SequenceMetadata } from "@/types/proto/v1/database_service";

defineProps({
  database: {
    required: true,
    type: Object as PropType<ComposedDatabase>,
  },
  schemaName: {
    type: String,
    default: "",
  },
  sequenceList: {
    required: true,
    type: Array as PropType<SequenceMetadata[]>,
  },
});

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
