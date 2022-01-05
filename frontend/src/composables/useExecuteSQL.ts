import { ref } from "vue";
import { isEmpty } from "lodash-es";
import { useStore } from "vuex";
import {
  isSelectStatement,
  isValidStatement,
} from "../components/MonacoEditor/sqlParser";

const useExecuteSQL = () => {
  const store = useStore();
  const isLoadingData = ref(false);

  const execute = async () => {
    const queryStatement = store.state.sqlEditor.queryStatement;
    const selectedStatement = store.state.sqlEditor.selectedStatement;
    const sqlStatement = selectedStatement || queryStatement;

    if (isEmpty(sqlStatement)) {
      store.dispatch("notification/pushNotification", {
        module: "bytebase",
        style: "CRITICAL",
        title: "Please input your SQL codes in the editor",
      });
      return;
    }

    if (!isValidStatement(sqlStatement)) {
      store.dispatch("notification/pushNotification", {
        module: "bytebase",
        style: "CRITICAL",
        title: "Please check if the statement is correct",
      });
      return;
    }

    if (!isSelectStatement(sqlStatement)) {
      store.dispatch("notification/pushNotification", {
        module: "bytebase",
        style: "CRITICAL",
        title: "Only SELECT statements are allowed",
      });
      return;
    }
    try {
      isLoadingData.value = true;
      const res = await store.dispatch("sqlEditor/executeQuery", {
        statement: sqlStatement,
      });
      isLoadingData.value = false;

      if (res.error) {
        store.dispatch("notification/pushNotification", {
          module: "bytebase",
          style: "CRITICAL",
          title: res.error,
        });
        return;
      }
    } catch (error) {
      isLoadingData.value = false;
      store.dispatch("notification/pushNotification", {
        module: "bytebase",
        style: "CRITICAL",
        title: error,
      });
    }
  };

  return {
    isLoadingData,
    execute,
  };
};

export { useExecuteSQL };
