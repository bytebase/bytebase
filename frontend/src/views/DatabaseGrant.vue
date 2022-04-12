<template>
  <!-- eslint-disable vue/attribute-hyphenation -->
  <DataSourceMemberForm
    :data-source="state.dataSource"
    :principalId="state.granteeId"
    :issueId="state.issueId"
    @submit="submit"
    @cancel="cancel"
  />
</template>

<script lang="ts">
import { reactive, onMounted, onUnmounted } from "vue";
import { useRouter } from "vue-router";
import DataSourceMemberForm from "../components/DataSourceMemberForm.vue";
import { IssueId, PrincipalId, DataSource } from "../types";
import { fullDataSourcePath } from "../utils";
import { useDataSourceStore } from "@/store";

interface LocalState {
  dataSource?: DataSource;
  granteeId?: PrincipalId;
  issueId?: IssueId;
}

export default {
  name: "DatabaseGrant",
  components: { DataSourceMemberForm },
  async setup() {
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
        ? await useDataSourceStore().fetchDataSourceById({
            dataSourceId: router.currentRoute.value.query.datasource as number,
            databaseId: router.currentRoute.value.query.database as number,
          })
        : undefined;

    const state = reactive<LocalState>({
      dataSource,
      granteeId: router.currentRoute.value.query.grantee as PrincipalId,
      issueId: router.currentRoute.value.query.issue as IssueId,
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
