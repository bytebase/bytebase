import { computed, ref, Ref, watch } from "vue";
import { type LocationQuery, useRoute, useRouter } from "vue-router";
import { defaultTemplate, templateForType } from "@/plugins";
import {
  pushNotification,
  useDatabaseV1Store,
  useIssueStore,
  useProjectV1Store,
} from "@/store";
import { EMPTY_ID, Issue, IssueCreate, IssueType } from "@/types";
import { SYSTEM_BOT_ID, UNKNOWN_ID } from "@/types";
import { idFromSlug } from "@/utils";
import { BuildNewIssueContext } from "../common";
import { tryGetDefaultAssignee } from "./assignee";
import { maybeBuildGhostIssue } from "./ghost";
import { maybeBuildGrantRequestIssue } from "./grantRequest";
import { buildNewStandardIssue } from "./standard";
import { maybeBuildTenantDeployIssue } from "./tenant";

export function useInitializeIssue(issueSlug: Ref<string>) {
  const issueStore = useIssueStore();
  const create = computed(() => issueSlug.value.toLowerCase() == "new");
  const route = useRoute();
  const router = useRouter();

  const issueCreate = ref<IssueCreate | undefined>();
  const issue = computed((): Issue | IssueCreate | undefined => {
    if (create.value) {
      return issueCreate.value;
    } else {
      const id = idFromSlug(issueSlug.value);
      const issueEntity = issueStore.getIssueById(id);
      if (issueEntity.id === EMPTY_ID || issueEntity.id === UNKNOWN_ID) {
        return undefined;
      }
      return issueEntity;
    }
  });

  const template = computed(() => {
    // Find proper IssueTemplate from route.query.template
    const issueType = route.query.template as IssueType;
    if (issueType) {
      const tpl = templateForType(issueType);
      if (tpl) {
        return tpl;
      }
      pushNotification({
        module: "bytebase",
        style: "WARN",
        title: `Unknown template '${issueType}'.`,
        description: "Fallback to the default template",
      });
    }

    // fallback
    return defaultTemplate();
  });

  watch(
    [issueSlug, create],
    async ([issueSlug, create]) => {
      try {
        if (create) {
          issueCreate.value = undefined;

          await prepareDatabaseListForIssueCreation(route.query);

          issueCreate.value = await buildNewIssue({ template, route });
          if (
            issueCreate.value.assigneeId === UNKNOWN_ID ||
            issueCreate.value.assigneeId === SYSTEM_BOT_ID
          ) {
            // Try to find a default assignee of the first task automatically.
            await tryGetDefaultAssignee(issueCreate.value);
          }
        } else {
          const id = idFromSlug(issueSlug);
          await issueStore.fetchIssueById(id);
        }
      } catch (error) {
        router.push({ name: "error.404" });
        throw error;
      }
    },
    { immediate: true }
  );

  return { create, issue };
}

const buildNewIssue = async (
  context: BuildNewIssueContext
): Promise<IssueCreate> => {
  const grantRequest = await maybeBuildGrantRequestIssue(context);
  if (grantRequest) {
    return grantRequest;
  }

  const ghost = await maybeBuildGhostIssue(context);
  if (ghost) {
    return ghost;
  }

  const tenant = await maybeBuildTenantDeployIssue(context);
  if (tenant) {
    return tenant;
  }

  return buildNewStandardIssue(context);
};

const prepareDatabaseListForIssueCreation = async (query: LocationQuery) => {
  const databaseStore = useDatabaseV1Store();
  // For preparing the database if user visits creating issue url directly.
  // It's horrible to fetchDatabaseByUID one-by-one when query.databaseList
  // is big (100+ sometimes)
  // So we are fetching databaseList by project since that's better cached.
  if (query.project) {
    // If we found query.project, we can directly search database list by project
    const projectId = query.project as string;
    const project = await useProjectV1Store().getOrFetchProjectByUID(projectId);
    await databaseStore.searchDatabaseList({
      parent: `instances/-`,
      filter: `project == "${project.name}"`,
    });
  } else if (query.databaseList) {
    // Otherwise, we don't have the projectId (very rare to see, theoretically)
    // so we need to fetch the first database in databaseList by id,
    // and see what project it belongs.
    const databaseIdList = (query.databaseList as string).split(",");
    if (databaseIdList.length > 0) {
      const firstDB = await databaseStore.getDatabaseByUID(databaseIdList[0]);
      if (databaseIdList.length > 1) {
        // If we have more than one databases in the list
        // fetch the rest of databases by projectId
        await databaseStore.searchDatabaseList({
          parent: `instances/-`,
          filter: `project == "${firstDB.project}"`,
        });
      }
    }
  }
};
