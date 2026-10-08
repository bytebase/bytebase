import { create } from "@bufbuild/protobuf";
import type { CallOptions } from "@connectrpc/connect";
import { rolloutServiceClientConnect } from "@/api";
import {
  ListTaskRunsRequestSchema,
  type TaskRun,
} from "@/types/proto-es/v1/rollout_service_pb";

// The server caps a page at 1000 task runs. Ask for the cap so a rollout of
// any realistic size is one round trip, and follow the token past it.
const TASK_RUN_PAGE_SIZE = 1000;

// Lists every task run under `parent`, typically a rollout's wildcard
// `.../stages/-/tasks/-`, by draining the ListTaskRuns pages.
export async function listAllTaskRuns(
  parent: string,
  options?: CallOptions
): Promise<TaskRun[]> {
  const taskRuns: TaskRun[] = [];
  let pageToken = "";
  do {
    const response = await rolloutServiceClientConnect.listTaskRuns(
      create(ListTaskRunsRequestSchema, {
        parent,
        pageSize: TASK_RUN_PAGE_SIZE,
        pageToken,
      }),
      options
    );
    taskRuns.push(...response.taskRuns);
    pageToken = response.nextPageToken;
  } while (pageToken);
  return taskRuns;
}
