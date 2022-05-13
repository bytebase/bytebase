import { computed, Ref } from "vue";
import { Database, IssueCreate, PITRContext, SYSTEM_BOT_ID } from "@/types";
import { useBackupListByDatabaseId, useIssueStore } from "@/store";

export const usePITRLogic = (database: Ref<Database>) => {
  const backupList = useBackupListByDatabaseId(
    computed(() => database.value.id)
  );
  const doneBackupList = computed(() =>
    backupList.value.filter((backup) => backup.status === "DONE")
  );

  const pitrAvailable = computed(() => doneBackupList.value.length > 0);

  const createPITRIssue = async (
    pointTimeTs: number,
    params: Partial<IssueCreate> = {}
  ) => {
    const issueStore = useIssueStore();
    const createContext: PITRContext = {
      databaseId: database.value.id,
      pointInTimeTs: pointTimeTs,
    };
    const issueCreate: IssueCreate = {
      name: `Restore database [${database.value.name}]`,
      type: "bb.issue.database.pitr",
      description: "",
      assigneeId: SYSTEM_BOT_ID,
      projectId: database.value.project.id,
      payload: {},
      createContext,
      ...params,
    };

    await issueStore.validateIssue(issueCreate);

    const issue = await issueStore.createIssue(issueCreate);

    return issue;
  };

  return {
    backupList,
    doneBackupList,
    pitrAvailable,
    createPITRIssue,
  };
};
