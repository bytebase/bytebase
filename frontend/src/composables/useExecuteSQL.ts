import { isEmpty } from "lodash-es";
import {
  isSelectStatement,
  isValidStatement,
} from "../components/MonacoEditor/sqlParser";

const useExecuteSQL = async (store: any) => {
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
