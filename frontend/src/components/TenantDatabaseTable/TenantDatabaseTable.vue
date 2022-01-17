<template>
  <div class="flex flex-col space-y-4">
    <DatabaseMatrix
      v-for="dbGroup in databaseListGroupByName"
      :key="dbGroup.name"
      :name="dbGroup.name"
      :database-list="dbGroup.databaseList"
      :environment-list="environmentList"
      :label-list="labelList"
    />
  </div>
</template>

<script lang="ts">
import { computed, defineComponent, PropType, watchEffect } from "vue";
import { useStore } from "vuex";
import { Database, Environment, Label, Project } from "../../types";
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
    databaseList: {
      type: Object as PropType<Database[]>,
      required: true,
    },
    project: {
      type: Object as PropType<Project>,
      required: true,
    },
    filter: {
      type: String,
      default: "",
    },
  },
  emits: ["select-database"],
  setup(props) {
    const store = useStore();

    const prepareList = () => {
      store.dispatch("environment/fetchEnvironmentList");
      store.dispatch("label/fetchLabelList");
    };
    watchEffect(prepareList);

    const labelList = computed(
      () => store.getters["label/labelList"]() as Label[]
    );
    const environmentList = computed(
      () => store.getters["environment/environmentList"]() as Environment[]
    );

    const filteredDatabaseList = computed(() => {
      if (!props.filter) return props.databaseList;

      return props.databaseList.filter((database) =>
        database.name.toLowerCase().includes(props.filter.toLowerCase())
      );
    });

    const databaseListGroupByName = computed((): DatabaseGroupByName[] => {
      if (props.project.dbNameTemplate) {
        if (labelList.value.length === 0) return [];
      }

      const dict = groupBy(filteredDatabaseList.value, (db) => {
        if (props.project.dbNameTemplate) {
          return parseDatabaseNameByTemplate(
            db.name,
            props.project.dbNameTemplate,
            labelList.value
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
      labelList,
      environmentList,
      databaseListGroupByName,
    };
  },
});
</script>
