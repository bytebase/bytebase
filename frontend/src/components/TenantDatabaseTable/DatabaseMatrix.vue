<template>
  <slot v-if="showHeader" name="header">
    <div class="text-base font-medium">{{ name }}</div>
  </slot>

  <BBTable
    :data-source="filteredMatrixList"
    :show-header="true"
    :custom-header="true"
    :left-bordered="bordered"
    :right-bordered="bordered"
    :top-bordered="bordered"
    :bottom-bordered="bordered"
    :row-clickable="false"
  >
    <template #header>
      <tr>
        <BBTableHeaderCell compact class="w-1/12 pl-4 pr-2 capitalize">
          {{ hidePrefix(yAxisLabel) }}
        </BBTableHeaderCell>

        <BBTableHeaderCell
          v-for="xValue in filteredXAxisValueList"
          :key="xValue"
          :style="{
            width: `${(100 - 1 / 12) / filteredXAxisValueList.length - 1}%`,
          }"
          class="text-center"
        >
          <template v-if="xValue">{{ xValue }}</template>
          <template v-else>{{ $t("label.empty-label-value") }}</template>
        </BBTableHeaderCell>
      </tr>
    </template>

    <template #body="{ rowData: matrix }">
      <BBTableCell
        :left-padding="4"
        class="pr-2 whitespace-nowrap"
        :class="{
          'text-control-placeholder': !matrix.labelValue,
        }"
      >
        <template v-if="matrix.labelValue">{{ matrix.labelValue }}</template>
        <template v-else>{{ $t("label.empty-label-value") }}</template>
      </BBTableCell>
      <BBTableCell v-for="(dbList, i) in matrix.databaseMatrix" :key="i">
        <div class="flex flex-col items-center space-y-1">
          <DatabaseMatrixItem
            v-for="db in dbList"
            :key="db.id"
            :database="db"
          />
          <span v-if="dbList.length === 0">-</span>
        </div>
      </BBTableCell>
    </template>
  </BBTable>
</template>

<script lang="ts">
import { computed, defineComponent, PropType, ref, watchEffect } from "vue";
import {
  Database,
  Environment,
  LabelKeyType,
  LabelValueType,
} from "../../types";
import DatabaseMatrixItem from "./DatabaseMatrixItem.vue";
import { groupBy, last } from "lodash-es";
import {
  hidePrefix,
  getLabelValue,
  getLabelValuesFromDatabaseList,
  LABEL_VALUE_EMPTY,
} from "../../utils";

type DatabaseMatrix = {
  labelValue: LabelValueType;
  databaseMatrix: Database[][];
};

export default defineComponent({
  name: "TenantDatabaseMatrix",
  components: { DatabaseMatrixItem },
  props: {
    bordered: {
      default: true,
      type: Boolean,
    },
    name: {
      type: String,
      default: "",
    },
    showHeader: {
      type: Boolean,
      default: true,
    },
    databaseList: {
      type: Object as PropType<Database[]>,
      required: true,
    },
    environmentList: {
      type: Array as PropType<Environment[]>,
      required: true,
    },
    xAxisLabel: {
      type: String as PropType<LabelKeyType>,
      required: true,
    },
    yAxisLabel: {
      type: String as PropType<LabelKeyType>,
      required: true,
    },
  },
  setup(props) {
    /**
     * databases are grouped by `yAxisLabel` then by `xAxisLabel`
     * for now, `xAxisLabel` will always be 'bb.environment'
     */
    const filteredXAxisValueList = ref<LabelValueType[]>([]);
    const filteredMatrixList = ref<DatabaseMatrix[]>([]);
    // pre-filtered y-axis values
    const yAxisValueList = computed((): string[] => {
      const key = props.yAxisLabel;
      if (!key) {
        // y-axis is undefined
        return [];
      }

      return getLabelValuesFromDatabaseList(
        key,
        props.databaseList,
        true /* withEmptyValue */
      );
    });

    // pre-filtered x-axis values
    const xAxisValueList = computed(() => {
      // Select all distinct database label values of {{key}}
      const key = props.xAxisLabel;

      return getLabelValuesFromDatabaseList(
        key,
        props.databaseList,
        true /* withEmptyValue */
      );
    });

    // first, group databases by `yAxisLabel` into some rows
    const groupedRowList = computed(() => {
      const key = props.yAxisLabel;
      if (!key) {
        // y-axis is undefined
        return [];
      }
      const dict = groupBy(props.databaseList, (db) => getLabelValue(db, key));
      return yAxisValueList.value
        .map((labelValue) => {
          const databaseList = dict[labelValue] || [];
          return {
            labelValue,
            databaseList,
          };
        })
        .filter((group) => group.databaseList.length > 0);
    });

    // then, group each row by `xAxisLabel` into some columns
    // now the `matrices` is pre-filtered (with empty rows or columns)
    const matrixList = computed((): DatabaseMatrix[] => {
      const key = props.xAxisLabel;
      return groupedRowList.value.map(
        ({ labelValue: yValue, databaseList }) => {
          const databaseMatrix: Database[][] = xAxisValueList.value.map(
            (xValue) =>
              databaseList.filter((db) => getLabelValue(db, key) === xValue)
          );
          return {
            labelValue: yValue,
            databaseMatrix,
          };
        }
      );
    });

    // now filter the axes and matrices
    // we only remove rows/cols of "<empty value>"
    // but keep other empty rows/cols because we want to give a whole view to
    //   the project.
    // e.g. if we hide "Prod" because there are no databases labeled as
    //   "bb:environment: Prod", we are then not able to judge whether there is
    //   no such an environment named "Prod" or "Prod" has no databases.
    watchEffect(() => {
      filteredXAxisValueList.value = [...xAxisValueList.value];

      // Hide the "<empty value>" COL if every row of this col has no databases.
      const hideLastCol = shouldHideEmptyValueCol(
        filteredXAxisValueList.value,
        matrixList.value
      );

      filteredMatrixList.value = [];
      for (let i = 0; i < matrixList.value.length; i++) {
        const row = {
          labelValue: matrixList.value[i].labelValue,
          databaseMatrix: [...matrixList.value[i].databaseMatrix],
        };
        if (hideLastCol) {
          row.databaseMatrix.pop();
        }
        filteredMatrixList.value.push(row);
      }

      // Hide the "<empty value>" ROW if every col of this row has no databases.
      if (shouldHideEmptyValueRow(filteredMatrixList.value)) {
        filteredMatrixList.value.pop();
      }
    });

    return {
      hidePrefix,
      yAxisValueList,
      xAxisValueList,
      filteredXAxisValueList,
      filteredMatrixList,
    };
  },
});

const shouldHideEmptyValueCol = (
  valueList: string[],
  matrixList: DatabaseMatrix[]
): boolean => {
  const lastValue = valueList[valueList.length - 1];
  if (lastValue === LABEL_VALUE_EMPTY) {
    // if every row's "<empty value>" has no databases
    // we should hide the "<empty value>" col
    if (
      matrixList.every((row) => {
        const lastCol = last(row.databaseMatrix) ?? [];
        return lastCol.length === 0;
      })
    ) {
      return true;
    }
  }
  return false;
};

const shouldHideEmptyValueRow = (matrixList: DatabaseMatrix[]) => {
  const lastRow = last(matrixList);
  if (
    lastRow?.labelValue === LABEL_VALUE_EMPTY &&
    lastRow?.databaseMatrix.every((databases) => databases.length === 0)
  ) {
    return true;
  }
  return false;
};
</script>
