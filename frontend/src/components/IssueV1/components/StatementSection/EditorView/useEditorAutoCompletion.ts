import { computed, Ref, watch } from "vue";
import { databaseForTask, useIssueContext } from "@/components/IssueV1/logic";
import MonacoEditor from "@/components/MonacoEditor";
import { useDBSchemaV1Store } from "@/store";
import { ComposedDatabase, UNKNOWN_ID } from "@/types";
import { TableMetadata } from "@/types/proto/v1/database_service";

export const useEditorAutoCompletion = (
  editorRef: Ref<InstanceType<typeof MonacoEditor> | undefined>
) => {
  const { issue, selectedTask } = useIssueContext();
  const dbSchemaStore = useDBSchemaV1Store();
  const databaseList = computed(() => {
    const db = databaseForTask(issue.value, selectedTask.value);
    if (db.uid === String(UNKNOWN_ID)) {
      return [];
    }
    return [db];
  });

  watch(
    databaseList,
    (list) => {
      list.forEach((db) => {
        if (db.uid !== String(UNKNOWN_ID)) {
          dbSchemaStore.getOrFetchDatabaseMetadata(db.name);
        }
      });
    },
    { immediate: true }
  );

  const tableList = computed(() => {
    return databaseList.value
      .map((item) => dbSchemaStore.getTableList(item.name))
      .flat();
  });

  const updateEditorAutoCompletionContext = async () => {
    const databaseMap: Map<ComposedDatabase, TableMetadata[]> = new Map();
    for (const database of databaseList.value) {
      const tableList = await dbSchemaStore.getOrFetchTableList(database.name);
      databaseMap.set(database, tableList);
    }
    editorRef.value?.setEditorAutoCompletionContextV1(databaseMap);
  };

  watch([databaseList, tableList], () => {
    updateEditorAutoCompletionContext();
  });

  return { updateEditorAutoCompletionContext };
};
