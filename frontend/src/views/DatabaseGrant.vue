<style scoped>
/*  Removed the ticker in the number field  */
input::-webkit-outer-spin-button,
input::-webkit-inner-spin-button {
  -webkit-appearance: none;
  margin: 0;
}

/* Firefox */
input[type="number"] {
  -moz-appearance: textfield;
}
</style>

<template>
  <DataSourceMemberForm
    :dataSource="state.dataSource"
    :principalId="state.granteeId"
    :taskId="state.taskId"
    @submit="submit"
    @cancel="cancel"
  />
</template>

<script lang="ts">
import { reactive, onMounted, onUnmounted } from "vue";
import { useStore } from "vuex";
import { useRouter } from "vue-router";
import DataSourceMemberForm from "../components/DataSourceMemberForm.vue";
import { TaskId, PrincipalId, DataSource } from "../types";
import { fullDataSourcePath } from "../utils";

interface LocalState {
  dataSource?: DataSource;
  granteeId?: PrincipalId;
  taskId?: TaskId;
}

export default {
  name: "DatabaseGrant",
  props: {},
  components: { DataSourceMemberForm },
  async setup(props, ctx) {
    const store = useStore();
    const router = useRouter();

    const keyboardHandler = (e: KeyboardEvent) => {
      if (e.code == "Escape") {
        cancel();
      }
    };

    onMounted(() => {
      document.addEventListener("keydown", keyboardHandler);
    });

    onUnmounted(() => {
      document.removeEventListener("keydown", keyboardHandler);
    });

    const dataSource =
      router.currentRoute.value.query.datasource &&
      router.currentRoute.value.query.database
        ? await store.dispatch("dataSource/fetchDataSourceById", {
            dataSourceId: router.currentRoute.value.query.datasource,
            databaseId: router.currentRoute.value.query.database,
          })
        : undefined;

    const state = reactive<LocalState>({
      dataSource,
      granteeId: router.currentRoute.value.query.grantee as PrincipalId,
      taskId: router.currentRoute.value.query.task as TaskId,
    });

    const cancel = () => {
      if (window.history.state?.back) {
        router.go(-1);
      } else {
        router.push("/");
      }
    };

    const submit = (createdDataSource: DataSource) => {
      router.push(fullDataSourcePath(createdDataSource));
    };

    return {
      state,
      cancel,
      submit,
    };
  },
};
</script>
