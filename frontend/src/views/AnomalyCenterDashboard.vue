<template>
  <div class="px-6 mt-4">
    <AnomalyTable :mode="'NORMAL'" :anomalyList="anomalyList" />
  </div>
</template>

<script lang="ts">
import { computed, watchEffect } from "vue-demi";
import { useStore } from "vuex";
import AnomalyTable from "../components/AnomalyTable.vue";
import { Anomaly, Database } from "../types";
export default {
  name: "AnomalyCenterDashboard",
  components: { AnomalyTable },
  setup(props, ctx) {
    const store = useStore();

    const currentUser = computed(() => store.getters["auth/currentUser"]());

    const prepareDatabaseList = () => {
      // It will also be called when user logout
      store.dispatch("database/fetchDatabaseList");
    };

    watchEffect(prepareDatabaseList);

    const databaseList = computed((): Database[] => {
      return store.getters["database/databaseListByPrincipalId"](
        currentUser.value.id
      );
    });

    const anomalyList = computed((): Anomaly[] => {
      const list: Anomaly[] = [];
      databaseList.value.forEach((database: Database) => {
        list.push(...database.anomalyList);
      });
      return list;
    });

    return {
      anomalyList,
    };
  },
};
</script>
