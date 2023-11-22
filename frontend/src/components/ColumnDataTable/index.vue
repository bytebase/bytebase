<template>
  <NDataTable
    :columns="columns"
    :data="filteredColumnList"
    :max-height="640"
    :virtual-scroll="true"
    :striped="true"
    :bordered="true"
    :bottom-bordered="true"
  />
</template>

<script lang="ts" setup>
import { DataTableColumn, NDataTable } from "naive-ui";
import { computed, PropType } from "vue";
import { h } from "vue";
import { useI18n } from "vue-i18n";
import { getColumnDefaultValuePlaceholder } from "@/components/SchemaEditorV1/utils/columnDefaultValue";
import ClassificationLevelBadge from "@/components/SchemaTemplate/ClassificationLevelBadge.vue";
import { useCurrentUserV1, useSubscriptionV1Store } from "@/store";
import { ComposedDatabase } from "@/types";
import { Engine } from "@/types/proto/v1/common";
import {
  ColumnMetadata,
  TableMetadata,
} from "@/types/proto/v1/database_service";
import { MaskData } from "@/types/proto/v1/org_policy_service";
import { DataClassificationSetting_DataClassificationConfig } from "@/types/proto/v1/setting_service";
import { hasWorkspacePermissionV1 } from "@/utils";
import LabelsCell from "./LabelsCell.vue";
import MaskingLevelCell from "./MaskingLevelCell.vue";
import SemanticTypeCell from "./SemanticTypeCell.vue";

const props = defineProps({
  database: {
    required: true,
    type: Object as PropType<ComposedDatabase>,
  },
  schema: {
    required: true,
    type: String,
  },
  table: {
    required: true,
    type: Object as PropType<TableMetadata>,
  },
  columnList: {
    required: true,
    type: Object as PropType<ColumnMetadata[]>,
  },
  maskDataList: {
    required: true,
    type: Array as PropType<MaskData[]>,
  },
  classificationConfig: {
    required: false,
    default: undefined,
    type: Object as PropType<
      DataClassificationSetting_DataClassificationConfig | undefined
    >,
  },
  search: {
    required: false,
    default: "",
    type: String,
  },
});

const { t } = useI18n();
const engine = computed(() => {
  return props.database.instanceEntity.engine;
});
const currentUserV1 = useCurrentUserV1();
const subscriptionV1Store = useSubscriptionV1Store();

const hasSensitiveDataFeature = computed(() => {
  return subscriptionV1Store.hasFeature("bb.feature.sensitive-data");
});

const showSensitiveColumn = computed(() => {
  return (
    hasSensitiveDataFeature.value &&
    (engine.value === Engine.MYSQL ||
      engine.value === Engine.TIDB ||
      engine.value === Engine.POSTGRES ||
      engine.value === Engine.REDSHIFT ||
      engine.value === Engine.ORACLE ||
      engine.value === Engine.SNOWFLAKE ||
      engine.value === Engine.MSSQL ||
      engine.value === Engine.RISINGWAVE)
  );
});

const showClassificationColumn = computed(() => {
  return (
    (engine.value === Engine.MYSQL || engine.value === Engine.POSTGRES) &&
    props.classificationConfig
  );
});

const showCollationColumn = computed(() => {
  return (
    engine.value !== Engine.CLICKHOUSE && engine.value !== Engine.SNOWFLAKE
  );
});

const hasSensitiveDataPermission = computed(() => {
  if (
    hasWorkspacePermissionV1(
      "bb.permission.workspace.manage-sensitive-data",
      currentUserV1.value.userRole
    )
  ) {
    // True if the currentUser has workspace level sensitive data
    // R+W privileges. AKA DBA or Workspace owner
    return true;
  }

  // False otherwise
  return false;
});

const columns = computed(() => {
  const columns: (DataTableColumn<ColumnMetadata> & { hide?: boolean })[] = [
    {
      key: "name",
      title: t("common.name"),
      resizable: true,
      width: 140,
      ellipsis: true,
      render: (column) => {
        return column.name;
      },
    },
    {
      key: "maskingLevel",
      title: t("settings.sensitive-data.masking-level.self"),
      hide: !showSensitiveColumn.value,
      resizable: true,
      width: 220,
      render: (column) => {
        return h(MaskingLevelCell, {
          database: props.database,
          schema: props.schema,
          table: props.table,
          column: column,
          maskDataList: props.maskDataList,
          readonly: !hasSensitiveDataPermission.value,
        });
      },
    },
    {
      key: "semanticType",
      title: t("settings.sensitive-data.semantic-types.self"),
      hide: !showSensitiveColumn.value,
      resizable: true,
      width: 140,
      render: (column) => {
        return h(SemanticTypeCell, {
          database: props.database,
          schema: props.schema,
          table: props.table,
          column: column,
          readonly: !hasSensitiveDataPermission.value,
        });
      },
    },
    {
      key: "classification",
      title: t("database.classification.self"),
      hide: !showClassificationColumn.value,
      width: 140,
      render: (column) => {
        return h(ClassificationLevelBadge, {
          classification: column.classification,
          classificationConfig: props.classificationConfig,
        });
      },
    },
    {
      key: "type",
      title: t("common.type"),
      resizable: true,
      width: 140,
      ellipsis: true,
      render: (column) => {
        return column.type;
      },
    },
    {
      key: "default",
      title: t("common.default"),
      resizable: true,
      width: 140,
      ellipsis: true,
      render: (column) => {
        return getColumnDefaultValuePlaceholder(column);
      },
    },
    {
      key: "nullable",
      title: t("database.nullable"),
      resizable: true,
      width: 140,
      render: (column) => {
        return column.nullable;
      },
    },
    {
      key: "characterSet",
      title: t("db.character-set"),
      hide: engine.value === Engine.POSTGRES,
      resizable: true,
      width: 140,
      ellipsis: true,
      render: (column) => {
        return column.characterSet;
      },
    },
    {
      key: "collation",
      title: t("db.collation"),
      hide: !showCollationColumn.value,
      resizable: true,
      width: 140,
      ellipsis: true,
      render: (column) => {
        return column.collation;
      },
    },
    {
      key: "comment",
      title: t("database.comment"),
      resizable: true,
      width: 140,
      ellipsis: true,
      render: (column) => {
        return column.userComment;
      },
    },
    {
      key: "labels",
      resizable: true,
      width: 140,
      title: t("common.labels"),
      render: (column) => {
        return h(LabelsCell, {
          database: props.database,
          schema: props.schema,
          table: props.table,
          column: column,
          readonly: !hasSensitiveDataPermission.value,
        });
      },
    },
  ];

  return columns.filter((column) => !column.hide);
});

const filteredColumnList = computed(() => {
  if (props.search) {
    return props.columnList.filter((column) => {
      return column.name.toLowerCase().includes(props.search.toLowerCase());
    });
  }
  return props.columnList;
});
</script>
