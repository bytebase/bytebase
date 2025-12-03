import { createDiscreteApi } from "naive-ui";
import type { LocationQueryRaw } from "vue-router";
import { t } from "@/plugins/i18n";
import { router } from "@/router";
import { PROJECT_V1_ROUTE_ISSUE_DETAIL } from "@/router/dashboard/projectV1";
import { useDatabaseV1Store, useDBGroupStore } from "@/store";
import { isValidDatabaseGroupName, isValidDatabaseName } from "@/types";
import { extractProjectResourceName, generateIssueTitle } from "@/utils";

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

  const query: LocationQueryRaw = {
    template: type,
    name: generateIssueTitle(type, databaseNames),
    databaseList: targets.join(","),
  };

  router.push({
    name: PROJECT_V1_ROUTE_ISSUE_DETAIL,
    params: {
      projectId: extractProjectResourceName(project),
      issueSlug: "create",
    },
    query,
  });
};
