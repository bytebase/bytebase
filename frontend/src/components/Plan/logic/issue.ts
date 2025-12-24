import { createDiscreteApi } from "naive-ui";
import type { LocationQueryRaw } from "vue-router";
import { useIssueLayoutVersion } from "@/composables/useIssueLayoutVersion";
import { t } from "@/plugins/i18n";
import { router } from "@/router";
import {
  PROJECT_V1_ROUTE_ISSUE_DETAIL,
  PROJECT_V1_ROUTE_PLAN_DETAIL_SPEC_DETAIL,
} from "@/router/dashboard/projectV1";
import {
  useDatabaseV1Store,
  useDBGroupStore,
  useProjectV1Store,
} from "@/store";
import { isValidDatabaseGroupName, isValidDatabaseName } from "@/types";
import {
  extractDatabaseGroupName,
  extractDatabaseResourceName,
  extractProjectResourceName,
  generateIssueTitle,
} from "@/utils";

const showDatabaseDriftedWarningDialog = () => {
  const { dialog } = createDiscreteApi(["dialog"]);

  return new Promise((resolve) => {
    dialog.create({
      type: "warning",
      positiveText: t("common.confirm"),
      negativeText: t("common.cancel"),
      title: t("issue.schema-drift-detected.self"),
      content: t("issue.schema-drift-detected.description"),
      autoFocus: false,
      onNegativeClick: () => {
        resolve(false);
      },
      onPositiveClick: () => {
        resolve(true);
      },
    });
  });
};

export const preCreateIssue = async (project: string, targets: string[]) => {
  const type = "bb.issue.database.update";
  const databaseStore = useDatabaseV1Store();
  const dbGroupStore = useDBGroupStore();
  const projectStore = useProjectV1Store();
  const { enabledNewLayout } = useIssueLayoutVersion();

  const databaseNames: string[] = [];
  let hasDraft = false;
  for (const target of targets) {
    if (isValidDatabaseGroupName(target)) {
      const dbGroup = dbGroupStore.getDBGroupByName(target);
      databaseNames.push(dbGroup.title);
    } else if (isValidDatabaseName(target)) {
      const db = databaseStore.getDatabaseByName(target);
      if (db.drifted) {
        hasDraft = true;
      }
      databaseNames.push(db.databaseName);
    }
  }

  if (hasDraft) {
    const confirmed = await showDatabaseDriftedWarningDialog();
    if (!confirmed) {
      return;
    }
  }

  const isDatabaseGroup = targets.every((target) =>
    isValidDatabaseGroupName(target)
  );

  // Fetch project to check enforce_issue_title setting
  const projectEntity = await projectStore.getOrFetchProjectByName(project);

  if (enabledNewLayout.value) {
    // New CI/CD layout: navigate to plan detail page
    const query: LocationQueryRaw = {
      template: type,
    };

    if (isDatabaseGroup) {
      const databaseGroupName = targets[0];
      query.databaseGroupName = databaseGroupName;
      // Only set title from generated if enforceIssueTitle is false.
      if (!projectEntity.enforceIssueTitle) {
        query.name = generateIssueTitle(type, [
          extractDatabaseGroupName(databaseGroupName),
        ]);
      }
    } else {
      query.databaseList = targets.join(",");
      // Only set title from generated if enforceIssueTitle is false.
      if (!projectEntity.enforceIssueTitle) {
        query.name = generateIssueTitle(
          type,
          targets.map((db) => {
            const { databaseName } = extractDatabaseResourceName(db);
            return databaseName;
          })
        );
      }
    }

    router.push({
      name: PROJECT_V1_ROUTE_PLAN_DETAIL_SPEC_DETAIL,
      params: {
        projectId: extractProjectResourceName(project),
        planId: "create",
        specId: "placeholder",
      },
      query,
    });
  } else {
    // Legacy layout: navigate to issue detail page
    const query: LocationQueryRaw = {
      template: type,
      databaseList: targets.join(","),
    };
    // Only set title from generated if enforceIssueTitle is false.
    if (!projectEntity.enforceIssueTitle) {
      query.name = generateIssueTitle(type, databaseNames);
    }

    router.push({
      name: PROJECT_V1_ROUTE_ISSUE_DETAIL,
      params: {
        projectId: extractProjectResourceName(project),
        issueSlug: "create",
      },
      query,
    });
  }
};
