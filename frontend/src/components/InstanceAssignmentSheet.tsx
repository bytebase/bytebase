import { create } from "@bufbuild/protobuf";
import { FieldMaskSchema } from "@bufbuild/protobuf/wkt";
import { Check } from "lucide-react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { EnvironmentLabel } from "@/components/EnvironmentLabel";
import { LearnMoreLink } from "@/components/LearnMoreLink";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { EllipsisText } from "@/components/ui/ellipsis-text";
import {
  Sheet,
  SheetBody,
  SheetContent,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";
import { Table } from "@/components/ui/table";
import { useServerState, useSubscriptionState } from "@/hooks/useAppState";
import { PagedTableFooter } from "@/hooks/usePagedData";
import { pushNotification } from "@/stores";
import { useAppStore } from "@/stores/app";
import { isValidInstanceName } from "@/types";
import { State } from "@/types/proto-es/v1/common_pb";
import type {
  Instance,
  UpdateInstanceRequest,
} from "@/types/proto-es/v1/instance_service_pb";
import { UpdateInstanceRequestSchema } from "@/types/proto-es/v1/instance_service_pb";
import { PlanType } from "@/types/proto-es/v1/subscription_service_pb";
import {
  extractInstanceResourceName,
  extractProjectResourceName,
  hasProjectPermissionV2,
  hasWorkspacePermissionV2,
} from "@/utils";

const PAGE_SIZE = 50;

interface InstancePageCursor {
  parents: (string | undefined)[];
  parentIndex: number;
  pageToken: string;
}

export interface InstanceAssignmentSheetProps {
  open: boolean;
  selectedInstanceList?: string[];
  onOpenChange: (open: boolean) => void;
  onUpdated?: () => void;
}

export function InstanceAssignmentSheet({
  open,
  selectedInstanceList,
  onOpenChange,
  onUpdated,
}: InstanceAssignmentSheetProps) {
  const { t } = useTranslation();
  const refreshServerInfo = useAppStore((state) => state.refreshServerInfo);

  const { instanceLicenseCount, currentPlan } = useSubscriptionState();
  const { activatedInstanceCount } = useServerState();

  const [instances, setInstances] = useState<Instance[]>([]);
  const [selectedNames, setSelectedNames] = useState<Set<string>>(new Set());
  const [nextPage, setNextPage] = useState<InstancePageCursor>();
  const [loading, setLoading] = useState(false);
  const [fetchingMore, setFetchingMore] = useState(false);
  const [processing, setProcessing] = useState(false);
  const fetchIdRef = useRef(0);
  const selectedInstanceKey = useMemo(
    () => (selectedInstanceList ?? []).join("\n"),
    [selectedInstanceList]
  );

  const canManageSubscription =
    hasWorkspacePermissionV2("bb.instances.update") &&
    currentPlan !== PlanType.FREE;

  const totalLicenseCount =
    instanceLicenseCount === Number.MAX_VALUE
      ? t("common.unlimited")
      : `${instanceLicenseCount}`;

  const applyActivatedSelection = useCallback((list: Instance[]) => {
    setSelectedNames((prev) => {
      const next = new Set(prev);
      for (const instance of list) {
        if (instance.activation) {
          next.add(instance.name);
        }
      }
      return next;
    });
  }, []);

  const fetchInstances = useCallback(
    async (refresh: boolean, cursor?: InstancePageCursor) => {
      const fetchId = ++fetchIdRef.current;
      if (refresh) {
        setLoading(true);
      } else {
        setFetchingMore(true);
      }
      try {
        const store = useAppStore.getState();
        let page = cursor;
        if (refresh || !page) {
          const parents: (string | undefined)[] = [undefined];
          let pageToken = "";
          do {
            const result = await store.fetchProjectList({
              pageSize: PAGE_SIZE,
              pageToken,
              filter: { state: State.ACTIVE, excludeDefault: true },
            });
            parents.push(
              ...result.projects
                .filter((project) =>
                  hasProjectPermissionV2(project, "bb.instances.list")
                )
                .map((project) => project.name)
            );
            pageToken = result.nextPageToken ?? "";
          } while (pageToken);
          page = { parents, parentIndex: 0, pageToken: "" };
        } else {
          page = { ...page };
        }
        const fetched: Instance[] = [];
        // Each collection owns its continuation token; never reuse it under another parent.
        while (page.parentIndex < page.parents.length) {
          const result = await store.fetchInstanceList({
            parent: page.parents[page.parentIndex],
            pageSize: PAGE_SIZE - fetched.length,
            pageToken: page.pageToken,
          });
          fetched.push(...result.instances);
          page.pageToken = result.nextPageToken ?? "";
          if (!page.pageToken) page.parentIndex++;
          if (
            fetched.length >= PAGE_SIZE ||
            (page.pageToken && fetched.length > 0)
          )
            break;
        }
        if (fetchId !== fetchIdRef.current) {
          return;
        }
        setInstances((prev) => (refresh ? fetched : [...prev, ...fetched]));
        setNextPage(page.parentIndex < page.parents.length ? page : undefined);
        applyActivatedSelection(fetched);
      } finally {
        if (fetchId === fetchIdRef.current) {
          setLoading(false);
          setFetchingMore(false);
        }
      }
    },
    [applyActivatedSelection]
  );

  useEffect(() => {
    if (!open) {
      fetchIdRef.current++;
      setInstances([]);
      setSelectedNames(new Set());
      setNextPage(undefined);
      setProcessing(false);
      return;
    }

    setSelectedNames(
      new Set(selectedInstanceKey ? selectedInstanceKey.split("\n") : [])
    );
    fetchInstances(true);
  }, [fetchInstances, open, selectedInstanceKey]);

  const toggleSelection = useCallback((name: string) => {
    setSelectedNames((prev) => {
      const next = new Set(prev);
      if (next.has(name)) {
        next.delete(name);
      } else {
        next.add(name);
      }
      return next;
    });
  }, []);

  const handleConfirm = useCallback(async () => {
    if (processing || !canManageSubscription) {
      return;
    }
    setProcessing(true);
    try {
      const requests: UpdateInstanceRequest[] = [];
      for (const instanceName of selectedNames) {
        const instance =
          (await useAppStore
            .getState()
            .getOrFetchInstanceByName(instanceName)) ??
          useAppStore.getState().getInstanceByName(instanceName);
        if (!isValidInstanceName(instance.name)) {
          continue;
        }
        if (instance.activation) {
          continue;
        }
        requests.push(
          create(UpdateInstanceRequestSchema, {
            instance: {
              ...instance,
              activation: true,
            },
            updateMask: create(FieldMaskSchema, { paths: ["activation"] }),
          })
        );
      }

      for (const instance of instances) {
        if (instance.activation && !selectedNames.has(instance.name)) {
          requests.push(
            create(UpdateInstanceRequestSchema, {
              instance: {
                ...instance,
                activation: false,
              },
              updateMask: create(FieldMaskSchema, { paths: ["activation"] }),
            })
          );
        }
      }

      // Release capacity first so a full license pool can move between collections.
      for (const activation of [false, true]) {
        const groups = new Map<string | undefined, UpdateInstanceRequest[]>();
        for (const request of requests) {
          if (request.instance?.activation !== activation) continue;
          const projectID = extractProjectResourceName(request.instance.name);
          const parent = projectID ? `projects/${projectID}` : undefined;
          const group = groups.get(parent) ?? [];
          group.push(request);
          groups.set(parent, group);
        }
        for (const [parent, group] of groups) {
          const updated = await useAppStore
            .getState()
            .batchUpdateInstances(group, parent);
          for (const instance of updated) {
            useAppStore.getState().updateDatabaseInstance(instance);
          }
        }
      }
      await refreshServerInfo();
      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: t("subscription.instance-assignment.success-notification"),
      });
      onUpdated?.();
      onOpenChange(false);
    } finally {
      setProcessing(false);
    }
  }, [
    canManageSubscription,
    instances,
    onOpenChange,
    onUpdated,
    processing,
    selectedNames,
    t,
    refreshServerInfo,
  ]);

  const projectedLicenseCount = useMemo(() => {
    const loadedByName = new Map(
      instances.map((instance) => [instance.name, instance])
    );
    let count = activatedInstanceCount;
    for (const name of selectedNames) {
      const instance =
        loadedByName.get(name) ??
        useAppStore.getState().getInstanceByName(name);
      if (!instance.activation) count++;
    }
    for (const instance of instances) {
      if (instance.activation && !selectedNames.has(instance.name)) count--;
    }
    return count;
  }, [activatedInstanceCount, instances, selectedNames]);

  const confirmDisabled =
    !canManageSubscription ||
    loading ||
    fetchingMore ||
    processing ||
    projectedLicenseCount > instanceLicenseCount;

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent width="standard">
        <SheetHeader>
          <SheetTitle>
            {t("subscription.instance-assignment.manage-license")}
          </SheetTitle>
        </SheetHeader>
        <SheetBody className="gap-y-6">
          <div>
            <div className="flex gap-x-2 text-control-light">
              <span>
                {t("subscription.instance-assignment.used-and-total-license")}
              </span>
              <LearnMoreLink
                href="https://docs.bytebase.com/administration/license?source=console"
                className="text-sm"
              />
            </div>
            <div className="mt-1 flex items-center gap-x-2 text-4xl">
              <span>{activatedInstanceCount}</span>
              <span className="text-xl text-control-light">/</span>
              <span>{totalLicenseCount}</span>
            </div>
          </div>

          <div className="overflow-x-auto rounded-sm border border-control-border">
            <Table className="min-w-[36rem]">
              <thead>
                <tr className="border-b border-control-border bg-control-bg text-left">
                  {canManageSubscription && <th className="w-12 px-4 py-2" />}
                  <th className="px-4 py-2 font-medium">{t("common.name")}</th>
                  <th className="px-4 py-2 font-medium">
                    {t("common.environment")}
                  </th>
                  <th className="px-4 py-2 font-medium">
                    {t("subscription.instance-assignment.license")}
                  </th>
                </tr>
              </thead>
              <tbody>
                {loading && instances.length === 0 ? (
                  <tr>
                    <td
                      colSpan={canManageSubscription ? 4 : 3}
                      className="px-4 py-8 text-center text-control-light"
                    >
                      {t("common.loading")}
                    </td>
                  </tr>
                ) : instances.length === 0 ? (
                  <tr>
                    <td
                      colSpan={canManageSubscription ? 4 : 3}
                      className="px-4 py-8 text-center text-control-light"
                    >
                      {t("common.no-data")}
                    </td>
                  </tr>
                ) : (
                  instances.map((instance) => {
                    const checked = selectedNames.has(instance.name);
                    return (
                      <tr
                        key={instance.name}
                        className="border-b border-control-border last:border-b-0"
                      >
                        {canManageSubscription && (
                          <td className="px-4 py-2">
                            <Checkbox
                              checked={checked}
                              onCheckedChange={() =>
                                toggleSelection(instance.name)
                              }
                            />
                          </td>
                        )}
                        <td className="px-4 py-2">
                          <EllipsisText
                            text={
                              instance.title ||
                              extractInstanceResourceName(instance.name)
                            }
                          />
                        </td>
                        <td className="px-4 py-2">
                          <EnvironmentLabel
                            environmentName={instance.environment}
                          />
                        </td>
                        <td className="px-4 py-2">
                          {checked ? (
                            <Check className="size-4 text-success" />
                          ) : (
                            "-"
                          )}
                        </td>
                      </tr>
                    );
                  })
                )}
              </tbody>
            </Table>
          </div>

          <PagedTableFooter
            pageSize={PAGE_SIZE}
            pageSizeOptions={[PAGE_SIZE]}
            onPageSizeChange={() => {}}
            hasMore={Boolean(nextPage)}
            isFetchingMore={fetchingMore}
            onLoadMore={() => fetchInstances(false, nextPage)}
          />
        </SheetBody>
        <SheetFooter>
          <Button appearance="outline" onClick={() => onOpenChange(false)}>
            {t("common.cancel")}
          </Button>
          <Button disabled={confirmDisabled} onClick={handleConfirm}>
            {t("common.confirm")}
          </Button>
        </SheetFooter>
      </SheetContent>
    </Sheet>
  );
}
