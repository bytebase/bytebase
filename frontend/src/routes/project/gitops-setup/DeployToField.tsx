import { useTranslation } from "react-i18next";
import {
  PROJECT_V1_ROUTE_DATABASE_GROUP_DETAIL,
  PROJECT_V1_ROUTE_DATABASE_GROUPS_CREATE,
  PROJECT_V1_ROUTE_DATABASES,
} from "@/app/router/handles";
import { DatabaseSelect } from "@/components/DatabaseSelect";
import { RouterLink } from "@/components/RouterLink";
import { Alert } from "@/components/ui/alert";
import { buttonVariants } from "@/components/ui/button";
import { Combobox } from "@/components/ui/combobox";
import { FormField } from "@/components/ui/form";
import { Tabs, TabsList, TabsPanel, TabsTrigger } from "@/components/ui/tabs";
import { getProjectNameAndDatabaseGroupName } from "@/stores/modules/v1/common";
import type { DatabaseGroup } from "@/types/proto-es/v1/database_group_service_pb";
import type { Database } from "@/types/proto-es/v1/database_service_pb";
import { DeployOrder } from "./DeployOrder";
import { setupFieldId } from "./setupField";
import type { DeployTargets } from "./useDeployTargets";

export type TargetTab = "group" | "databases";

const linkClass = "text-sm font-medium text-accent hover:underline";

// The database group, or the databases, GitOps deploys to, and the order the
// deploy reaches them.
export function DeployToField({
  projectId,
  projectName,
  groups,
  noPickableDatabase,
  tab,
  onTabChange,
  groupName,
  onGroupChange,
  databaseNames,
  onDatabasesChange,
  targets,
  unsupported,
}: {
  readonly projectId: string;
  readonly projectName: string;
  readonly groups: DatabaseGroup[];
  // Every database of the project has no environment.
  readonly noPickableDatabase: boolean;
  readonly tab: TargetTab;
  readonly onTabChange: (tab: TargetTab) => void;
  readonly groupName: string;
  readonly onGroupChange: (name: string) => void;
  readonly databaseNames: string[];
  readonly onDatabasesChange: (names: string[]) => void;
  readonly targets: DeployTargets;
  // The databases the deploy fails on with the picked workflow type.
  readonly unsupported: Database[];
}) {
  const { t } = useTranslation();
  const picked = tab === "group" ? groupName !== "" : databaseNames.length > 0;
  const groupMatchesNone =
    tab === "group" && picked && !targets.loading && targets.total === 0;

  return (
    <FormField
      id={setupFieldId("deploy-to")}
      className="scroll-my-4"
      title={t("gitops.setup.deploy-to.label")}
    >
      <div className="flex min-w-0 flex-col gap-y-2">
        <Tabs
          value={tab}
          onValueChange={(value) => onTabChange(value as TargetTab)}
        >
          <TabsList>
            <TabsTrigger value="group">
              {t("gitops.setup.databases.group-tab")}
            </TabsTrigger>
            <TabsTrigger value="databases">
              {t("gitops.setup.databases.databases-tab")}
            </TabsTrigger>
          </TabsList>

          <TabsPanel value="group">
            {groups.length === 0 ? (
              <div className="flex flex-col items-start gap-y-2 rounded-sm border border-block-border p-4">
                <p className="text-sm font-medium text-main">
                  {t("gitops.setup.databases.no-group-title")}
                </p>
                <p className="text-sm text-control-light">
                  {t("gitops.setup.databases.no-group-description")}
                </p>
                <RouterLink
                  to={{
                    name: PROJECT_V1_ROUTE_DATABASE_GROUPS_CREATE,
                    params: { projectId },
                  }}
                  className={buttonVariants({
                    appearance: "outline",
                    size: "sm",
                  })}
                >
                  {t("gitops.setup.databases.create-group")}
                </RouterLink>
              </div>
            ) : (
              <div className="flex flex-col gap-y-1.5">
                <Combobox
                  value={groupName}
                  options={groups.map((group) => ({
                    value: group.name,
                    label: group.title || group.name,
                  }))}
                  placeholder={t("gitops.setup.databases.group-placeholder")}
                  onChange={onGroupChange}
                />
                <p className="text-xs text-control-light">
                  {t("gitops.setup.databases.group-hint")}{" "}
                  {groupName && <ViewGroupLink groupName={groupName} />}
                </p>
              </div>
            )}
          </TabsPanel>

          <TabsPanel value="databases">
            {noPickableDatabase ? (
              <Alert
                variant="warning"
                title={t("gitops.setup.databases.none-pickable-title")}
                description={t(
                  "gitops.setup.databases.none-pickable-description"
                )}
              >
                <RouterLink
                  to={{
                    name: PROJECT_V1_ROUTE_DATABASES,
                    params: { projectId },
                  }}
                  className={linkClass}
                >
                  {t("gitops.setup.databases.view-databases")}
                </RouterLink>
              </Alert>
            ) : (
              <div className="flex flex-col gap-y-1.5">
                <DatabaseSelect
                  multiple
                  requireEnvironment
                  projectName={projectName}
                  value={databaseNames}
                  placeholder={t(
                    "gitops.setup.databases.databases-placeholder"
                  )}
                  onChange={(names) => onDatabasesChange(names)}
                />
                <p className="text-xs text-control-light">
                  {t("gitops.setup.databases.databases-hint")}
                </p>
              </div>
            )}
          </TabsPanel>
        </Tabs>

        {groupMatchesNone && (
          <p className="text-sm text-control-light">
            {t("gitops.setup.databases.group-matches-none")}
          </p>
        )}

        {picked && targets.total > 0 && (
          <DeployOrder targets={targets} unsupported={unsupported} />
        )}
      </div>
    </FormField>
  );
}

function ViewGroupLink({ groupName }: { readonly groupName: string }) {
  const { t } = useTranslation();
  const [projectId, databaseGroupName] =
    getProjectNameAndDatabaseGroupName(groupName);
  return (
    <RouterLink
      to={{
        name: PROJECT_V1_ROUTE_DATABASE_GROUP_DETAIL,
        params: { projectId, databaseGroupName },
      }}
      className="text-xs font-medium whitespace-nowrap text-accent hover:underline"
    >
      {t("gitops.setup.databases.view-group")}
    </RouterLink>
  );
}
