import { useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { DatabaseTargetDisplay } from "@/components/DatabaseTargetDisplay";
import { EnvironmentLabel } from "@/components/EnvironmentLabel";
import { Alert } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import { SearchInput } from "@/components/ui/search-input";
import { cn } from "@/lib/utils";
import type { Database } from "@/types/proto-es/v1/database_service_pb";
import { extractDatabaseResourceName } from "@/utils";
import type { DeployTargets } from "./useDeployTargets";

// Databases shown as chips before the "+N more" action, as on the plan page.
export const VISIBLE_DATABASES = 3;
// The "+N more" list gets a search box above this many databases.
const SEARCHABLE_DATABASES = 20;

// The picked databases in the order the deploy reaches them: one section per
// environment, joined by a line, and a warning for databases with no
// environment.
export function DeployOrder({ targets }: { readonly targets: DeployTargets }) {
  const { t } = useTranslation();
  const { stages, unassigned, total } = targets;
  return (
    <div className="flex flex-col gap-y-3">
      {stages.length > 0 && (
        <div className="flex flex-col gap-y-2">
          {total > VISIBLE_DATABASES && (
            <p className="text-sm text-control-light">
              {t("gitops.setup.deploy-order.summary", {
                databases: t("gitops.setup.deploy-order.database-count", {
                  count: total,
                }),
                environments: t("gitops.setup.deploy-order.environment-count", {
                  count: stages.length,
                }),
              })}
            </p>
          )}
          <ol
            aria-label={t("gitops.setup.deploy-order.label")}
            className="rounded-sm border border-block-border px-4 py-1"
          >
            {stages.map((stage, index) => (
              <li
                key={stage.environment}
                className="relative flex flex-col gap-y-2 py-3 pl-8"
              >
                {index > 0 && (
                  <span
                    aria-hidden="true"
                    className="absolute top-0 left-3 h-6 w-px -translate-x-1/2 bg-control-border"
                  />
                )}
                {index < stages.length - 1 && (
                  <span
                    aria-hidden="true"
                    className="absolute top-6 bottom-0 left-3 w-px -translate-x-1/2 bg-control-border"
                  />
                )}
                <span
                  aria-hidden="true"
                  className="absolute top-6 left-3 size-2.5 -translate-x-1/2 -translate-y-1/2 rounded-full border-2 border-control-border bg-background"
                />
                <div className="flex h-6 items-center">
                  <EnvironmentLabel environmentName={stage.environment} />
                </div>
                <DatabaseChips databases={stage.databases} />
              </li>
            ))}
          </ol>
        </div>
      )}
      {unassigned.length > 0 && (
        <Alert
          variant="warning"
          title={t("gitops.setup.deploy-order.unassigned-title", {
            count: unassigned.length,
          })}
          description={t("gitops.setup.deploy-order.unassigned-description", {
            count: unassigned.length,
          })}
        >
          <DatabaseChips className="mt-2" databases={unassigned} />
        </Alert>
      )}
    </div>
  );
}

export function DatabaseChips({
  databases,
  className,
}: {
  readonly databases: Database[];
  readonly className?: string;
}) {
  const visible = databases.slice(0, VISIBLE_DATABASES);
  const extra = databases.slice(VISIBLE_DATABASES);
  return (
    <div className={cn("flex min-w-0 flex-wrap items-center gap-2", className)}>
      {visible.map((database) => (
        <DatabaseChip key={database.name} database={database} />
      ))}
      {extra.length > 0 && <MoreDatabases databases={extra} />}
    </div>
  );
}

function DatabaseChip({ database }: { readonly database: Database }) {
  return (
    <div className="inline-flex max-w-full min-w-0 cursor-default items-center gap-x-1 rounded-sm border bg-control-bg/50 px-2 py-1">
      <DatabaseTargetDisplay database={database} />
    </div>
  );
}

function MoreDatabases({ databases }: { readonly databases: Database[] }) {
  const { t } = useTranslation();
  const [searchText, setSearchText] = useState("");
  const filtered = useMemo(() => {
    const query = searchText.trim().toLowerCase();
    if (!query) return databases;
    return databases.filter(
      (database) =>
        extractDatabaseResourceName(database.name)
          .databaseName.toLowerCase()
          .includes(query) || database.name.toLowerCase().includes(query)
    );
  }, [databases, searchText]);
  return (
    <Popover>
      <PopoverTrigger
        render={
          <Button
            className="text-accent hover:bg-accent/10 hover:text-accent"
            size="xs"
            type="button"
            appearance="secondary"
          />
        }
      >
        {t("common.n-more", { n: databases.length })}
      </PopoverTrigger>
      <PopoverContent
        align="start"
        className="w-[min(520px,calc(100vw-2rem))] p-0"
        side="bottom"
      >
        <div className="border-b border-control-border px-3 py-2">
          <div className="text-sm font-medium text-control">
            {t("common.databases")} ({databases.length})
          </div>
        </div>
        {databases.length > SEARCHABLE_DATABASES && (
          <div className="border-b border-control-border px-3 py-2">
            <SearchInput
              placeholder={t("common.search")}
              value={searchText}
              onChange={(event) => setSearchText(event.target.value)}
            />
          </div>
        )}
        <div className="max-h-80 overflow-y-auto p-2">
          {filtered.length > 0 ? (
            <div className="flex flex-col gap-1">
              {filtered.map((database) => (
                <div
                  key={database.name}
                  className="min-w-0 rounded-xs px-2 py-1.5 hover:bg-control-bg"
                >
                  <DatabaseTargetDisplay database={database} />
                </div>
              ))}
            </div>
          ) : (
            <div className="px-2 py-4 text-center text-sm text-control-light">
              {t("common.no-data")}
            </div>
          )}
        </div>
      </PopoverContent>
    </Popover>
  );
}
