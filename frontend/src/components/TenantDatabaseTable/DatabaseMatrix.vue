<template>
  <slot v-if="showHeader" name="header">
    <div class="text-base font-medium">{{ name }}</div>
  </slot>

  <BBTable
    :data-source="filteredMatrices"
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
        <BBTableHeaderCell compact class="w-1/12 pl-3 pr-2">
          <YAxisSwitch
            v-if="yAxisLabel"
            v-model:label="yAxisLabel"
            :label-list="selectableLabelList"
          />
        </BBTableHeaderCell>

        <BBTableHeaderCell
          v-for="xValue in filteredXAxisValues"
          :key="xValue"
          :style="{
            width: `${(100 - 1 / 12) / filteredXAxisValues.length - 1}%`,
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
            :custom-click="customClick"
            @select-database="(db) => $emit('select-database', db)"
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
  Label,
  LabelKeyType,
  LabelValueType,
} from "../../types";
import DatabaseMatrixItem from "./DatabaseMatrixItem.vue";
import { groupBy } from "lodash-es";
import { findDefaultGroupByLabel, getLabelValue } from "../../utils";
import YAxisSwitch1 from "./YAxisSwitch.vue";

type DatabaseMatrix = {
  labelValue: LabelValueType;
  databaseMatrix: Database[][];
};

export default defineComponent({
  name: "TenantDatabaseMatrix",
  components: { DatabaseMatrixItem, YAxisSwitch: YAxisSwitch1 },
  props: {
    bordered: {
      default: true,
      type: Boolean,
    },
    customClick: {
      default: false,
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
    labelList: {
      type: Array as PropType<Label[]>,
      required: true,
    },
  },
  emits: ["select-database"],
  setup(props) {
    // make "bb.environment" non-selectable because it was already specified to the x-axis
    const selectableLabelList = computed(() => {
      const excludes = new Set(["bb.environment"]);
      return props.labelList.filter((label) => !excludes.has(label.key));
    });

    /**
     * databases are grouped by `yAxisLabel` then by `xAxisLabel`
     * for now, `xAxisLabel` will always be 'bb.environment'
     */
    const yAxisLabel = ref<LabelKeyType>();
    const xAxisLabel = ref<LabelKeyType>("bb.environment");

    const filteredXAxisValues = ref<LabelValueType[]>([]);
    const filteredMatrices = ref<DatabaseMatrix[]>([]);

    // find the default label key to firstBy (y-axis)
    watchEffect(() => {
      // "bb.environment" is excluded because it was specified to the x-axis
      yAxisLabel.value = findDefaultGroupByLabel(
        selectableLabelList.value,
        props.databaseList
      );
    });

    // pre-filtered y-axis values
    const yAxisValues = computed((): string[] => {
      const key = yAxisLabel.value;
      if (!key) {
        // y-axis is undefined
        return [];
      }

      // order based on label.valueList
      // plus one more "<empty value>"
      const label = props.labelList.find((label) => label.key === key);
      if (!label) return [];
      return [...label.valueList, ""];
    });

    // pre-filtered x-axis values
    const xAxisValues = computed(() => {
      // order based on label.valueList
      // plus one more "<empty value>"
      const key = xAxisLabel.value;
      const label = props.labelList.find((label) => label.key === key);
      if (!label) return [];
      return [...label.valueList, ""];
    });

    // first, group databases by `yAxisLabel` into some rows
    const groupedRows = computed(() => {
      const key = yAxisLabel.value;
      if (!key) {
        // y-axis is undefined
        return [];
      }
      const dict = groupBy(props.databaseList, (db) => getLabelValue(db, key));
      return yAxisValues.value.map((labelValue) => {
        const databaseList = dict[labelValue] || [];
        return {
          labelValue,
          databaseList,
        };
      });
    });

    // then, group each row by `xAxisLabel` into some columns
    // now the `matrices` is pre-filtered (with empty rows or columns)
    const matrices = computed((): DatabaseMatrix[] => {
      const key = xAxisLabel.value;
      return groupedRows.value.map(({ labelValue: yValue, databaseList }) => {
        const databaseMatrix: Database[][] = xAxisValues.value.map((xValue) =>
          databaseList.filter((db) => getLabelValue(db, key) === xValue)
        );
        return {
          labelValue: yValue,
          databaseMatrix,
        };
      });
    });

    // now filter the axes and matrices
    // we only remove rows/cols of "<empty value>"
    // but keep other empty rows/cols because we want to give a whole view to
    //   the project.
    // e.g. if we hide "Prod" because there are no databases labeled as
    //   "bb:environment: Prod", we are then not able to judge whether there is
    //   no such an environment named "Prod" or "Prod" has no databases.
    watchEffect(() => {
      filteredXAxisValues.value = [...xAxisValues.value];
      // if every row's "<empty value>" has no databases
      // we should hide the "<empty value>" col
      const shouldClipLastCol = matrices.value.every(
        (row) =>
          row.databaseMatrix.length > 0 &&
          row.databaseMatrix[row.databaseMatrix.length - 1].length === 0
      );
      if (shouldClipLastCol) filteredXAxisValues.value.pop();

      filteredMatrices.value = [];
      for (let i = 0; i < matrices.value.length; i++) {
        const row = {
          labelValue: matrices.value[i].labelValue,
          databaseMatrix: [...matrices.value[i].databaseMatrix],
        };
        if (shouldClipLastCol) {
          row.databaseMatrix.pop();
        }
        filteredMatrices.value.push(row);
      }
      // if every col of "<empty value>" row is empty
      // we should hide the "<empty value>" row
      if (
        filteredMatrices.value.length > 0 &&
        filteredMatrices.value[
          filteredMatrices.value.length - 1
        ].databaseMatrix.every((item) => item.length === 0)
      ) {
        filteredMatrices.value.pop();
      }
    });

    return {
      selectableLabelList,
      yAxisLabel,
      yAxisValues,
      xAxisValues,
      filteredXAxisValues,
      filteredMatrices,
    };
  },
});
</script>
