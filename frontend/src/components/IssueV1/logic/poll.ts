import { isEqual } from "lodash-es";
import { watch } from "vue";
import { useProgressivePoll } from "@/composables/useProgressivePoll";
import {
  experimentalFetchIssueByUID,
  useChangelogStore,
  useDBSchemaV1Store,
  useInstanceV1Store,
} from "@/store";
import { useListCache } from "@/store/modules/v1/cache";
import type { ComposedIssue } from "@/types";
import { IssueStatus } from "@/types/proto-es/v1/issue_service_pb";
import { Task_Type } from "@/types/proto-es/v1/rollout_service_pb";
import { databaseForTask, extractIssueUID, flattenTaskV1List } from "@/utils";
import { useIssueContext } from "./context";
import { projectOfIssue } from "./utils";

const clearCache = (issue: ComposedIssue) => {
  const changelogStore = useChangelogStore();
  const tasks = flattenTaskV1List(issue.rolloutEntity);

  for (const task of tasks) {
    const database = databaseForTask(projectOfIssue(issue), task);
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
      case Task_Type.DATABASE_EXPORT:
      case Task_Type.TYPE_UNSPECIFIED:
        continue;
      case Task_Type.DATABASE_MIGRATE:
        // Always clear the schema cache for MIGRATE tasks
        useDBSchemaV1Store().removeCache(database.name);
        break;
      default:
        useDBSchemaV1Store().removeCache(database.name);
        changelogStore.clearCache(database.name);
    }
  }
};

export const usePollIssue = () => {
  const { isCreating, ready, issue, events } = useIssueContext();

  const refreshIssue = () => {
    if (isCreating.value || !ready.value) return;
    experimentalFetchIssueByUID(
      extractIssueUID(issue.value.name),
      issue.value.project
    ).then((updatedIssue) => {
      if (
        issue.value.status !== IssueStatus.DONE &&
        updatedIssue.status === IssueStatus.DONE
      ) {
        clearCache(updatedIssue);
      }

      if (!isEqual(issue.value, updatedIssue)) {
        issue.value = updatedIssue;
      }
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
