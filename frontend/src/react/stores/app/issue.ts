import { create as createProto } from "@bufbuild/protobuf";
import { createContextValues } from "@connectrpc/connect";
import { issueServiceClientConnect } from "@/connect";
import { silentContextKey } from "@/connect/context-key";
import { GetIssueRequestSchema } from "@/types/proto-es/v1/issue_service_pb";
import type { AppSliceCreator, IssueSlice } from "./types";

// Resource names are `projects/{project}/issues/{issue}` — yank the project
// resource out of the head without importing the heavy `@/utils/v1/project`
// barrel (which pulls the Pinia actuator chain into the app-store graph).
function projectResourceFromIssueName(issueName: string): string {
  const match = issueName.match(/^(projects\/[^/]+)\//);
  return match?.[1] ?? "";
}

/**
 * Port of the legacy Pinia `useIssueV1Store.fetchIssueByName`. Stateless
 * fetch (no per-issue cache — callers consume the result once) that also
 * primes the owning project in the app store, matching the Pinia behavior
 * so downstream code that reads the project synchronously after the call
 * sees it cached.
 */
export const createIssueSlice: AppSliceCreator<IssueSlice> = (_set, get) => ({
  fetchIssueByName: async (name, silent = false) => {
    const issue = await issueServiceClientConnect.getIssue(
      createProto(GetIssueRequestSchema, { name }),
      {
        contextValues: createContextValues().set(silentContextKey, silent),
      }
    );
    const projectName = projectResourceFromIssueName(issue.name);
    if (projectName) {
      await get().fetchProject(projectName);
    }
    return issue;
  },
});
