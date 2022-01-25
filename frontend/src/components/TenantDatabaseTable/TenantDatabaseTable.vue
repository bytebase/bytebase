<template>
  <div class="flex flex-col space-y-4">
    <DatabaseMatrix
      v-for="dbGroup in databaseListGroupByName"
      :key="dbGroup.name"
      :name="dbGroup.name"
      :database-list="dbGroup.databaseList"
      :environment-list="environmentList"
      :label-list="labelList"
      :x-axis-label="xAxisLabel"
      :y-axis-label="yAxisLabel"
    />
  </div>
</template>

<script lang="ts">
import { computed, defineComponent, PropType, watchEffect } from "vue";
import { useStore } from "vuex";
import {
  Database,
  Environment,
  Label,
  Project,
  LabelKeyType,
} from "../../types";
import { groupBy } from "lodash-es";
import DatabaseMatrix from "./DatabaseMatrix.vue";
import { parseDatabaseNameByTemplate } from "../../utils";

type Mode = "ALL" | "ALL_SHORT" | "INSTANCE" | "PROJECT" | "PROJECT_SHORT";

type DatabaseGroupByName = {
  name: string;
  databaseList: Database[];
};

export default defineComponent({
  name: "TenantDatabaseTable",
  components: { DatabaseMatrix },
  props: {
    bordered: {
      default: true,
      type: Boolean,
    },
    mode: {
      default: "ALL",
      type: String as PropType<Mode>,
    },
    customClick: {
      default: false,
      type: Boolean,
    },
    labelList: {
      type: Array as PropType<Label[]>,
      required: true,
    },
    databaseList: {
      type: Object as PropType<Database[]>,
      required: true,
    },
    project: {
      type: Object as PropType<Project>,
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
  emits: ["select-database"],
  setup(props) {
    const store = useStore();

    const prepareList = () => {
      store.dispatch("environment/fetchEnvironmentList");
    };
    watchEffect(prepareList);

    const environmentList = computed(
      () => store.getters["environment/environmentList"]() as Environment[]
    );

    const databaseListGroupByName = computed((): DatabaseGroupByName[] => {
      if (props.project.dbNameTemplate) {
        if (props.labelList.length === 0) return [];
      }

      const dict = groupBy(props.databaseList, (db) => {
        if (props.project.dbNameTemplate) {
          return parseDatabaseNameByTemplate(
            db.name,
            props.project.dbNameTemplate,
            props.labelList
          );
        } else {
          return db.name;
        }
      });
      return Object.keys(dict).map((name) => ({
        name,
        databaseList: dict[name],
      }));
    });

    return {
      environmentList,
      databaseListGroupByName,
    };
  },
});
</script>
