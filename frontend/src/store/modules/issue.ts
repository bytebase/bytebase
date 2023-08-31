import axios from "axios";
import { defineStore } from "pinia";
import { computed, ref, unref, watch, WatchCallback, watchEffect } from "vue";
import {
  empty,
  EMPTY_ID,
  isPagedResponse,
  Issue,
  IssueCreate,
  IssueFind,
  IssueId,
  IssuePatch,
  IssueState,
  IssueStatusPatch,
  MaybeRef,
  Pipeline,
  Principal,
  Project,
  ResourceIdentifier,
  ResourceObject,
  unknown,
  UNKNOWN_ID,
} from "@/types";
import { isDatabaseRelatedIssueType } from "@/utils";
import { useLegacyDatabaseStore } from "./database";
import { usePipelineStore } from "./pipeline";
import { getPrincipalFromIncludedList } from "./principal";
import { useLegacyProjectStore } from "./project";
import { convertEntityList } from "./utils";
import {
  useDatabaseV1Store,
  useInstanceV1Store,
  useActivityV1Store,
} from "./v1";

function convert(issue: ResourceObject, includedList: ResourceObject[]): Issue {
  const result: Issue = {
    ...(issue.attributes as Omit<
      Issue,
      "id" | "project" | "creator" | "updater" | "assignee" | "subscriberList"
    >),
    id: parseInt(issue.id),
    creator: getPrincipalFromIncludedList(
      issue.relationships!.creator.data,
      includedList
    ),
    updater: getPrincipalFromIncludedList(
      issue.relationships!.updater.data,
      includedList
    ),
    assignee: getPrincipalFromIncludedList(
      issue.relationships!.assignee.data,
      includedList
    ),
    project: unknown("PROJECT") as Project,
    pipeline: undefined,
    subscriberList: [],
  };

  try {
    result.payload = JSON.parse(issue.attributes.payload as string);
  } catch {
    result.payload = {};
  }
  const projectStore = useLegacyProjectStore();

  // Compose issue pipeline.
  if (isDatabaseRelatedIssueType(result.type)) {
    let pipeline = unknown("PIPELINE") as Pipeline;
    if (issue.relationships?.pipeline?.data) {
      const pipelineStore = usePipelineStore();
      const pipelineId = (
        issue.relationships!.pipeline.data as ResourceIdentifier
      ).id;
      pipeline.id = parseInt(pipelineId);

      for (const item of includedList || []) {
        if (
          item.type == "pipeline" &&
          issue.relationships!.pipeline.data &&
          (issue.relationships!.pipeline.data as ResourceIdentifier).id ==
            item.id
        ) {
          pipeline = pipelineStore.convert(item, includedList);
        }
      }
    }
    result.pipeline = pipeline;
  }

  // Compose issue project.
  for (const item of includedList || []) {
    if (
      item.type == "project" &&
      (issue.relationships!.project.data as ResourceIdentifier).id == item.id
    ) {
      result.project = projectStore.convert(item, includedList || []);
    }
  }

  // Compose issue subscriberList.
  const subscriberList = [] as Principal[];
  if (issue.relationships!.subscriberList.data) {
    for (const subscriberData of issue.relationships!.subscriberList
      .data as ResourceIdentifier[]) {
      subscriberList.push(
        getPrincipalFromIncludedList(subscriberData, includedList)
      );
    }
  }

  result.subscriberList = subscriberList;

  return result;
}

function getIssueFromIncludedList(
  data:
    | ResourceIdentifier<ResourceObject>
    | ResourceIdentifier<ResourceObject>[]
    | undefined,
  includedList: ResourceObject[]
): Issue {
  if (data == null) {
    return empty("ISSUE");
  }
  for (const item of includedList || []) {
    if (item.type !== "issue") {
      continue;
    }
    if (item.id == (data as ResourceIdentifier).id) {
      return convert(item, includedList);
    }
  }
  return empty("ISSUE");
}

export const useIssueStore = defineStore("issue", {
  state: (): IssueState => ({
    issueById: new Map(),
    isCreatingIssue: false,
  }),
  getters: {
    issueList: (state) => {
      return [...state.issueById.values()];
    },
  },
  actions: {
    getIssueById(issueId: IssueId): Issue {
      if (issueId == EMPTY_ID) {
        return empty("ISSUE") as Issue;
      }

      return this.issueById.get(issueId) || (unknown("ISSUE") as Issue);
    },
    setIssueById({ issueId, issue }: { issueId: IssueId; issue: Issue }) {
      this.issueById.set(issueId, issue);
    },
    async fetchPagedIssueList(params: IssueFind & { token?: string }) {
      const queryList = buildQueryListByIssueFind(params);

      let url = "/api/issue";
      if (queryList.length > 0) {
        url += `?${queryList.join("&")}`;
      }
      const responseData = (await axios.get(url)).data;
      const issueList = convertEntityList(
        responseData,
        "issues",
        convert,
        getIssueFromIncludedList
      );

      // The issue list API returns incomplete issue entities (without
      // task.instance and task.database compositions)
      // So we shouldn't store the incomplete cache items in issueById set here.

      const nextToken = isPagedResponse(responseData, "issues")
        ? responseData.data.attributes.nextToken
        : "";
      return {
        nextToken,
        issueList,
      };
    },
    async fetchIssueList(params: IssueFind) {
      const result = await this.fetchPagedIssueList(params);
      return result.issueList;
    },
    async fetchIssueById(issueId: IssueId) {
      const data = (await axios.get(`/api/issue/${issueId}`)).data;
      const issue = convert(data.data, data.included);
      this.setIssueById({
        issueId,
        issue,
      });

      // It might be the first time the particular instance/database objects are returned,
      // so that we should also update instance/database store, otherwise, we may get
      // unknown instance/database when navigating to other UI from the issue detail page
      // since other UIs are getting instance/database by id from the store.
      if (isDatabaseRelatedIssueType(issue.type)) {
        const databaseStore = useLegacyDatabaseStore();
        const instanceV1Store = useInstanceV1Store();
        const databaseV1Store = useDatabaseV1Store();
        for (const stage of issue.pipeline!.stageList) {
          for (const task of stage.taskList) {
            instanceV1Store.getOrFetchInstanceByUID(String(task.instance.id));

            if (task.database) {
              databaseV1Store.getOrFetchDatabaseByUID(String(task.database.id));

              // Legacy compatibility
              databaseStore.upsertDatabaseList({
                databaseList: [task.database],
              });
            }
          }
        }
      }
      return issue;
    },
    async getOrFetchIssueById(issueId: IssueId) {
      if (issueId === EMPTY_ID) return empty("ISSUE");
      if (issueId === UNKNOWN_ID) return unknown("ISSUE");
      if (!this.issueById.has(issueId)) {
        await this.fetchIssueById(issueId);
      }
      return this.getIssueById(issueId);
    },
    async createIssue(newIssue: IssueCreate) {
      try {
        this.isCreatingIssue = true;
        const data = (
          await axios.post(`/api/issue`, {
            data: {
              type: "IssueCreate",
              attributes: {
                ...newIssue,
                // Server expects payload as string, so we stringify first.
                createContext: JSON.stringify(newIssue.createContext),
                payload: JSON.stringify(newIssue.payload),
              },
            },
          })
        ).data;
        const createdIssue = convert(data.data, data.included);

        this.setIssueById({
          issueId: createdIssue.id,
          issue: createdIssue,
        });

        return createdIssue;
      } finally {
        this.isCreatingIssue = false;
      }
    },
    async validateIssue(newIssue: IssueCreate) {
      const data = (
        await axios.post(`/api/issue`, {
          data: {
            type: "IssueCreate",
            attributes: {
              ...newIssue,
              // Server expects payload as string, so we stringify first.
              createContext: JSON.stringify(newIssue.createContext),
              payload: JSON.stringify(newIssue.payload),
              validateOnly: true,
            },
          },
        })
      ).data;
      const createdIssue = convert(data.data, data.included);
      return createdIssue;
    },
    async patchIssue({
      issueId,
      issuePatch,
    }: {
      issueId: IssueId;
      issuePatch: IssuePatch;
    }) {
      const data = (
        await axios.patch(`/api/issue/${issueId}`, {
          data: {
            type: "issuePatch",
            attributes: issuePatch,
          },
        })
      ).data;
      const updatedIssue = convert(data.data, data.included);

      this.setIssueById({
        issueId: issueId,
        issue: updatedIssue,
      });

      useActivityV1Store().fetchActivityListByIssueId(issueId);

      return updatedIssue;
    },
    async updateIssueStatus({
      issueId,
      issueStatusPatch,
    }: {
      issueId: IssueId;
      issueStatusPatch: IssueStatusPatch;
    }) {
      const data = (
        await axios.patch(`/api/issue/${issueId}/status`, {
          data: {
            type: "issueStatusPatch",
            attributes: issueStatusPatch,
          },
        })
      ).data;
      const updatedIssue = convert(data.data, data.included);

      this.setIssueById({
        issueId: issueId,
        issue: updatedIssue,
      });

      useActivityV1Store().fetchActivityListByIssueId(issueId);

      return updatedIssue;
    },
  },
});

export const buildQueryListByIssueFind = (
  params: IssueFind & { token?: string }
): string[] => {
  const {
    projectId,
    principalId,
    creatorId,
    assigneeId,
    subscriberId,
    statusList,
    limit,
    token,
  } = params;

  const queryList = [];
  if (statusList && statusList.length > 0) {
    queryList.push(`status=${statusList.join(",")}`);
  }
  if (principalId) {
    queryList.push(`user=${principalId}`);
  }
  if (creatorId) {
    queryList.push(`creator=${creatorId}`);
  }
  if (assigneeId) {
    queryList.push(`assignee=${assigneeId}`);
  }
  if (subscriberId) {
    queryList.push(`subscriber=${subscriberId}`);
  }
  if (projectId) {
    queryList.push(`project=${projectId}`);
  }
  if (limit) {
    queryList.push(`limit=${limit}`);
  }
  if (token) {
    queryList.push(`token=${token}`);
  }

  return queryList;
};

// expose global list refresh features
const REFRESH_ISSUE_LIST = ref(Math.random());
export const refreshIssueList = () => {
  REFRESH_ISSUE_LIST.value = Math.random();
};
export const useRefreshIssueList = (callback: WatchCallback) => {
  watch(REFRESH_ISSUE_LIST, callback);
};

export const useIssueById = (issueId: MaybeRef<IssueId>, lazy = false) => {
  const store = useIssueStore();

  watchEffect(() => {
    const id = unref(issueId);
    if (id !== UNKNOWN_ID) {
      if (lazy && store.issueById.has(id)) {
        // Don't fetch again if we have a local copy
        // when lazy === true
        return;
      }
      store.fetchIssueById(id);
    }
  });

  return computed(() => {
    return store.getIssueById(unref(issueId));
  });
};
