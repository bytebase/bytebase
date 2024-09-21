<template>
  <NDataTable
    key="database-table"
    size="small"
    :columns="columnList"
    :data="groupedStageList"
    :bordered="bordered"
    :row-key="(data: DeploymentMatrixRowData) => data.labelValue"
  ></NDataTable>
</template>

<script lang="tsx" setup>
import { groupBy } from "lodash-es";
import { NDataTable, type DataTableColumn } from "naive-ui";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import type { ComposedDatabase } from "@/types";
import type { Environment } from "@/types/proto/v1/environment_service";
import type { DeploymentConfig } from "@/types/proto/v1/project_service";
import {
  displayDeploymentMatchSelectorKey,
  getLabelValuesFromDatabaseV1List,
  getPipelineFromDeploymentScheduleV1,
  getSemanticLabelValue,
} from "@/utils";
import DatabaseMatrixGroup from "./DatabaseMatrixGroup.vue";

type DeploymentMatrixRowData = {
  labelValue: string;
  stages: ComposedDatabase[][];
  unmatched: ComposedDatabase[];
};

type DeploymentMatrixDataTableColumn =
  DataTableColumn<DeploymentMatrixRowData> & {
    hide?: boolean;
  };

const props = withDefaults(
  defineProps<{
    databaseList: ComposedDatabase[];
    label: string;
    environmentList: Environment[];
    deployment: DeploymentConfig;
    bordered?: boolean;
    showRest?: boolean;
  }>(),
  {
    bordered: true,
    showRest: true,
  }
);

const { t } = useI18n();

const databaseGroupList = computed(() => {
  const key = props.label;
  const dict = groupBy(props.databaseList, (db) =>
    getSemanticLabelValue(db, key)
  );
  const rows = getLabelValuesFromDatabaseV1List(
    props.label,
    props.databaseList
  ).map((labelValue) => {
    const databaseList = dict[labelValue] || [];
    return {
      labelValue,
      databaseList,
    };
  });

  // Add the empty label value row only if it matches any db.
  const emptyLabelDBList = dict[""] || [];
  if (emptyLabelDBList.length > 0) {
    rows.push({labelValue: "", databaseList: emptyLabelDBList})
  }

  return rows;
});

const groupedStageList = computed(() => {
  return databaseGroupList.value.map(({ labelValue, databaseList }) => {
    const stages = getPipelineFromDeploymentScheduleV1(
      databaseList,
      props.deployment.schedule
    );
    const affectedNames = stages.flatMap((dbs) => dbs.map((db) => db.name));
    const dict = new Set(affectedNames);
    const unmatched = props.showRest
      ? databaseList.filter((db) => !dict.has(db.name))
      : [];

    return {
      labelValue,
      stages,
      unmatched,
    };
  });
});

const hasRest = computed(() => {
  return groupedStageList.value.some((group) => group.unmatched.length > 0);
});

const columnList = computed((): DeploymentMatrixDataTableColumn[] => {
  const ENVIRONTMENT_ID: DeploymentMatrixDataTableColumn = {
    key: "environment-id",
    title: displayDeploymentMatchSelectorKey(props.label),
    hide: !(props.label === "environment"),
    width: 160,
    render: (data) => {
      return data.labelValue;
    },
  };
  const DATABASE_UNIT: DeploymentMatrixDataTableColumn = {
    key: "database-unit",
    title: displayDeploymentMatchSelectorKey(props.label),
    hide: props.label === "environment",
    width: 160,
    render: (data) => {
      if (!data.labelValue) {
        return <span class="text-opacity-70 italic">{t("common.empty")}</span>;
      }
      return data.labelValue;
    },
  };

  const deployments = props.deployment.schedule?.deployments ?? [];
  const columnCount = hasRest.value
    ? deployments.length + 1
    : deployments.length;
  const STAGE_LIST: DeploymentMatrixDataTableColumn[] = (deployments ?? []).map(
    (dep, i) => ({
      key: `stage-${i}`,
      title: dep.title,
      width: `${100 / columnCount}%`,
      minWidth: 240,
      render: (data) => {
        const matched = data.stages[i];
        if (matched.length === 0) {
          return "-";
        }
        return <DatabaseMatrixGroup databases={matched} />;
      },
    })
  );
  const UNMATCHED_LIST: DeploymentMatrixDataTableColumn = {
    key: `unmatched`,
    title: "Unmatched",
    width: `${100 / columnCount}%`,
    minWidth: 240,
    hide: !hasRest.value,
    render: (data) => {
      const unmatched = data.unmatched;
      if (unmatched.length === 0) {
        return "-";
      }
      return <DatabaseMatrixGroup databases={unmatched} />;
    },
  };

  return [ENVIRONTMENT_ID, DATABASE_UNIT, ...STAGE_LIST, UNMATCHED_LIST].filter(
    (col) => !col.hide
  );
});
</script>
