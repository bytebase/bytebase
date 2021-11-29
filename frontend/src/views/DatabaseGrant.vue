<template>
  <!-- eslint-disable vue/attribute-hyphenation -->
  <DataSourceMemberForm
    :data-source="state.dataSource"
    :principalID="state.granteeID"
    :issueID="state.issueID"
    @submit="submit"
    @cancel="cancel"
  />
</template>

<script lang="ts">
import { reactive, onMounted, onUnmounted } from "vue";
import { useStore } from "vuex";
import { useRouter } from "vue-router";
import DataSourceMemberForm from "../components/DataSourceMemberForm.vue";
import { IssueID, PrincipalID, DataSource } from "../types";
import { fullDataSourcePath } from "../utils";

interface LocalState {
  dataSource?: DataSource;
  granteeID?: PrincipalID;
  issueID?: IssueID;
}

export default {
  name: "DatabaseGrant",
  components: { DataSourceMemberForm },
  async setup() {
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
        ? await store.dispatch("dataSource/fetchDataSourceByID", {
            dataSourceID: router.currentRoute.value.query.datasource,
            databaseID: router.currentRoute.value.query.database,
          })
        : undefined;

    const state = reactive<LocalState>({
      dataSource,
      granteeID: router.currentRoute.value.query.grantee as PrincipalID,
      issueID: router.currentRoute.value.query.issue as IssueID,
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
