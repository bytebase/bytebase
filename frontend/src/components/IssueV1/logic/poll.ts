import { watch } from "vue";
import { databaseForTask } from "@/components/IssueV1/logic";
import { useProgressivePoll } from "@/composables/useProgressivePoll";
import {
  experimentalFetchIssueByUID,
  useChangeHistoryStore,
  useInstanceV1Store,
  useDBSchemaV1Store,
} from "@/store";
import { useListCache } from "@/store/modules/v1/cache";
import type { ComposedIssue } from "@/types";
import { IssueStatus } from "@/types/proto/v1/issue_service";
import { Task_Type } from "@/types/proto/v1/rollout_service";
import { extractIssueUID, extractProjectResourceName } from "@/utils";
import { flattenTaskV1List } from "@/utils";
import { useIssueContext } from "./context";

const clearCache = (issue: ComposedIssue) => {
  const changeHistoryStore = useChangeHistoryStore();
  const tasks = flattenTaskV1List(issue.rolloutEntity);

  for (const task of tasks) {
    const database = databaseForTask(issue, task);
    switch (task.type) {
      case Task_Type.DATABASE_CREATE:
        useInstanceV1Store()
          .syncInstance(database.instance, false /* not enable full sync */)
          .then(() => {
            const cache = useListCache("database");
            cache.deleteCache(database.instance);
            cache.deleteCache(database.project);
          });
        break;
      case Task_Type.DATABASE_DATA_EXPORT:
      case Task_Type.UNRECOGNIZED:
      case Task_Type.TYPE_UNSPECIFIED:
        continue;
      case Task_Type.DATABASE_DATA_UPDATE:
        changeHistoryStore.clearCache(database.name);
        break;
      default:
        useDBSchemaV1Store().removeCache(database.name);
        changeHistoryStore.clearCache(database.name);
    }
  }
};

export const usePollIssue = () => {
  const { isCreating, ready, issue, events } = useIssueContext();

  const refreshIssue = () => {
    if (isCreating.value || !ready.value) return;
    experimentalFetchIssueByUID(
      extractIssueUID(issue.value.name),
      extractProjectResourceName(issue.value.project)
    ).then((updatedIssue) => {
      if (
        issue.value.status !== IssueStatus.DONE &&
        updatedIssue.status === IssueStatus.DONE
      ) {
        clearCache(updatedIssue);
      }

      issue.value = updatedIssue;
    });
  };

  const poller = useProgressivePoll(refreshIssue, {
    interval: {
      min: 500,
      max: 10000,
      growth: 2,
      jitter: 500,
    },
  });

  watch(
    () => [isCreating.value, ready.value],
    () => {
      if (!isCreating.value && ready.value) {
        poller.start();
      } else {
        poller.stop();
      }
    },
    {
      immediate: true,
    }
  );

  events.on("status-changed", ({ eager }) => {
    if (eager) {
      refreshIssue();
      poller.restart();
    }
  });
};
