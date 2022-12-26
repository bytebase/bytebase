<template>
  <BBTable
    :column-list="columnNameList"
    :data-source="columnList"
    :show-header="true"
    :left-bordered="true"
    :right-bordered="true"
    v-bind="$attrs"
  >
    <template #body="{ rowData: column }">
      <BBTableCell
        v-if="showSensitiveColumn"
        :left-padding="4"
        class="w-[1%] text-center"
      >
        <!-- width: 1% means as narrow as possible -->
        <input
          type="checkbox"
          class="h-4 w-4 text-accent rounded disabled:cursor-not-allowed border-control-border focus:ring-accent"
          :disabled="!allowAdmin"
          :checked="isSensitiveColumn(column)"
          @input="
            toggleSensitiveColumn(
              column,
              ($event.target as HTMLInputElement).checked,
              $event
            )
          "
        />
      </BBTableCell>
      <BBTableCell class="w-16" :left-padding="showSensitiveColumn ? 2 : 4">
        {{ column.name }}
      </BBTableCell>
      <BBTableCell class="w-8">
        {{ column.type }}
      </BBTableCell>
      <BBTableCell class="w-8">
        {{ column.default }}
      </BBTableCell>
      <BBTableCell class="w-8">
        {{ column.nullable }}
      </BBTableCell>
      <BBTableCell
        v-if="
          engine != 'POSTGRES' &&
          engine != 'CLICKHOUSE' &&
          engine != 'SNOWFLAKE'
        "
        class="w-8"
      >
        {{ column.characterSet }}
      </BBTableCell>
      <BBTableCell
        v-if="engine != 'CLICKHOUSE' && engine != 'SNOWFLAKE'"
        class="w-8"
      >
        {{ column.collation }}
      </BBTableCell>
      <BBTableCell class="w-16">
        {{ column.comment }}
      </BBTableCell>
    </template>
  </BBTable>

  <FeatureModal
    v-if="state.showFeatureModal"
    feature="bb.feature.sensitive-data"
    @cancel="state.showFeatureModal = false"
  />
</template>

<script lang="ts">
import { cloneDeep } from "lodash-es";
import { computed, defineComponent, PropType, reactive } from "vue";
import { useI18n } from "vue-i18n";
import {
  Column,
  Database,
  SensitiveData,
  SensitiveDataPolicyPayload,
} from "@/types";
import { ColumnMetadata, TableMetadata } from "@/types/proto/store/database";
import { featureToRef, useCurrentUser, usePolicyStore } from "@/store";
import { hasWorkspacePermission } from "@/utils";
import { BBTableColumn } from "@/bbkit/types";

type LocalState = {
  showFeatureModal: boolean;
};

export default defineComponent({
  name: "ColumnTable",
  props: {
    database: {
      required: true,
      type: Object as PropType<Database>,
    },
    table: {
      required: true,
      type: Object as PropType<TableMetadata>,
    },
    columnList: {
      required: true,
      type: Object as PropType<ColumnMetadata[]>,
    },
    sensitiveDataList: {
      required: true,
      type: Array as PropType<SensitiveData[]>,
    },
  },
  setup(props) {
    const { t } = useI18n();
    const state = reactive<LocalState>({
      showFeatureModal: false,
    });
    const engine = computed(() => {
      return props.database.instance.engine;
    });

    const hasSensitiveDataFeature = featureToRef("bb.feature.sensitive-data");
    const showSensitiveColumn = computed(() => {
      return hasSensitiveDataFeature.value && engine.value === "MYSQL";
    });

    const currentUser = useCurrentUser();
    const allowAdmin = computed(() => {
      if (
        hasWorkspacePermission(
          "bb.permission.workspace.manage-sensitive-data",
          currentUser.value.role
        )
      ) {
        // True if the currentUser has workspace level sensitive data
        // R+W privileges. AKA DBA or Workspace owner
        return true;
      }

      // False otherwise
      return false;
    });

    const NORMAL_COLUMN_LIST = computed(() => {
      const columnList: BBTableColumn[] = [
        {
          title: t("common.name"),
        },
        {
          title: t("common.type"),
        },
        {
          title: t("common.Default"),
        },
        {
          title: t("database.nullable"),
        },
        {
          title: t("db.character-set"),
        },
        {
          title: t("db.collation"),
        },
        {
          title: t("database.comment"),
        },
      ];
      if (showSensitiveColumn.value) {
        columnList.unshift({
          title: t("database.sensitive"),
          center: true,
          nowrap: true,
        });
      }
      return columnList;
    });
    const POSTGRES_COLUMN_LIST = computed((): BBTableColumn[] => [
      {
        title: t("common.name"),
      },
      {
        title: t("common.type"),
      },
      {
        title: t("common.Default"),
      },
      {
        title: t("database.nullable"),
      },
      {
        title: t("db.collation"),
      },
      {
        title: t("database.comment"),
      },
    ]);
    const CLICKHOUSE_SNOWFLAKE_COLUMN_LIST = computed((): BBTableColumn[] => [
      {
        title: t("common.name"),
      },
      {
        title: t("common.type"),
      },
      {
        title: t("common.Default"),
      },
      {
        title: t("database.nullable"),
      },
      {
        title: t("database.comment"),
      },
    ]);

    const columnNameList = computed(() => {
      switch (engine.value) {
        case "POSTGRES":
          return POSTGRES_COLUMN_LIST.value;
        case "CLICKHOUSE":
        case "SNOWFLAKE":
          return CLICKHOUSE_SNOWFLAKE_COLUMN_LIST.value;
        default:
          return NORMAL_COLUMN_LIST.value;
      }
    });

    const isSensitiveColumn = (column: Column) => {
      return (
        props.sensitiveDataList.findIndex((sensitiveData) => {
          return (
            sensitiveData.table === props.table.name &&
            sensitiveData.column === column.name
          );
        }) >= 0
      );
    };

    const toggleSensitiveColumn = (column: Column, on: boolean, e: Event) => {
      if (!hasSensitiveDataFeature.value) {
        state.showFeatureModal = true;

        // Revert UI states
        e.preventDefault();
        e.stopPropagation();
        (e.target as HTMLInputElement).checked = !on;
        return;
      }

      const index = props.sensitiveDataList.findIndex((sensitiveData) => {
        return (
          sensitiveData.table === props.table.name &&
          sensitiveData.column === column.name
        );
      });
      const sensitiveDataList = cloneDeep(props.sensitiveDataList);
      if (on && index < 0) {
        // Turn on sensitive
        sensitiveDataList.push({
          table: props.table.name,
          column: column.name,
          maskType: "DEFAULT",
        });
      } else if (!on && index >= 0) {
        sensitiveDataList.splice(index, 1);
      }
      const payload: SensitiveDataPolicyPayload = {
        sensitiveDataList,
      };
      usePolicyStore().upsertPolicyByDatabaseAndType({
        databaseId: props.database.id,
        type: "bb.policy.sensitive-data",
        policyUpsert: {
          payload,
        },
      });
    };

    return {
      engine,
      state,
      columnNameList,
      showSensitiveColumn,
      allowAdmin,
      isSensitiveColumn,
      toggleSensitiveColumn,
    };
  },
});
</script>
