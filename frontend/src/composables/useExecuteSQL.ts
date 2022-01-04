import { isEmpty } from "lodash-es";
import { isSelectStatement } from "../components/MonacoEditor/sqlParser";

const useExecuteSQL = async (store: any) => {
  const queryStatement = store.state.sqlEditor.queryStatement;
  const selectedStatement = store.state.sqlEditor.selectedStatement;
  const sqlStatement = selectedStatement || queryStatement;

  if (!isEmpty(sqlStatement) && !isSelectStatement(sqlStatement)) {
    store.dispatch("notification/pushNotification", {
      module: "bytebase",
      style: "CRITICAL",
      title: "Only SELECT statements are allowed",
    });
    return;
  }

  try {
    const res = await store.dispatch("sqlEditor/executeQuery", {
      statement: sqlStatement,
    });

    if (res.error) {
      store.dispatch("notification/pushNotification", {
        module: "bytebase",
        style: "CRITICAL",
        title: res.error,
      });
      return;
    }
  } catch (error) {
    store.dispatch("notification/pushNotification", {
      module: "bytebase",
      style: "CRITICAL",
      title: error,
    });
  }
};

export { useExecuteSQL };
