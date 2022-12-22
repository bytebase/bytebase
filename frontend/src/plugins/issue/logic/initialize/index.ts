import { computed, ref, Ref, watch } from "vue";
import { type LocationQuery, useRoute, useRouter } from "vue-router";
import type { Issue, IssueCreate, IssueType } from "@/types";
import { SYSTEM_BOT_ID, UNKNOWN_ID } from "@/types";
import { pushNotification, useDatabaseStore, useIssueStore } from "@/store";
import { idFromSlug } from "@/utils";
import { defaultTemplate, templateForType } from "@/plugins";
import { BuildNewIssueContext } from "../common";
import { maybeBuildTenantDeployIssue } from "./tenant";
import { maybeBuildGhostIssue } from "./ghost";
import { buildNewStandardIssue } from "./standard";
import { tryGetDefaultAssignee } from "./assignee";

export function useInitializeIssue(issueSlug: Ref<string>) {
  const issueStore = useIssueStore();
  const create = computed(() => issueSlug.value.toLowerCase() == "new");
  const route = useRoute();
  const router = useRouter();

  const issue = ref<Issue | IssueCreate | undefined>();

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
      issue.value = undefined;

      try {
        if (create) {
          await prepareDatabaseListForIssueCreation(route.query);

          issue.value = await buildNewIssue({ template, route });
          if (
            issue.value.assigneeId === UNKNOWN_ID ||
            issue.value.assigneeId === SYSTEM_BOT_ID
          ) {
            // Try to find a default assignee of the first task automatically.
            await tryGetDefaultAssignee(issue.value);
          }
        } else {
          const id = idFromSlug(issueSlug);
          const fetchedIssue = await issueStore.fetchIssueById(id);
          issue.value = fetchedIssue;
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
  const databaseStore = useDatabaseStore();
  // For preparing the database if user visits creating issue url directly.
  // It's horrible to fetchDatabaseById one-by-one when query.databaseList
  // is big (100+ sometimes)
  // So we are fetching databaseList by project since that's better cached.
  if (query.project) {
    // If we found query.project, we can directly fetchDatabaseListByProjectId
    const projectId = query.project as string;
    await databaseStore.fetchDatabaseListByProjectId(parseInt(projectId, 10));
  } else {
    // Otherwise, we don't have the projectId (very rare to see, theoretically)
    // so we need to fetch the first database in databaseList by id,
    // and see what project it belongs.
    const databaseIdList = (query.databaseList as string)
      .split(",")
      .map((str) => parseInt(str, 10));
    if (databaseIdList.length > 0) {
      const firstDB = await databaseStore.getOrFetchDatabaseById(
        databaseIdList[0]
      );
      if (databaseIdList.length > 1) {
        // If we have more than one databases in the list
        // fetch the rest of databases by projectId
        await databaseStore.fetchDatabaseListByProjectId(firstDB.project.id);
      }
    }
  }
};
