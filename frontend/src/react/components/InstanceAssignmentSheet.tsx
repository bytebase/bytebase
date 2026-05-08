import { create } from "@bufbuild/protobuf";
import { FieldMaskSchema } from "@bufbuild/protobuf/wkt";
import { Check } from "lucide-react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { EnvironmentLabel } from "@/react/components/EnvironmentLabel";
import { LearnMoreLink } from "@/react/components/LearnMoreLink";
import { Button } from "@/react/components/ui/button";
import { Checkbox } from "@/react/components/ui/checkbox";
import { EllipsisText } from "@/react/components/ui/ellipsis-text";
import {
  Sheet,
  SheetBody,
  SheetContent,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from "@/react/components/ui/sheet";
import {
  useServerState,
  useSubscriptionState,
} from "@/react/hooks/useAppState";
import { PagedTableFooter } from "@/react/hooks/usePagedData";
import { useAppStore } from "@/react/stores/app";
import {
  pushNotification,
  useDatabaseV1Store,
  useInstanceV1Store,
} from "@/store";
import { isValidInstanceName } from "@/types";
import type {
  Instance,
  UpdateInstanceRequest,
} from "@/types/proto-es/v1/instance_service_pb";
import { UpdateInstanceRequestSchema } from "@/types/proto-es/v1/instance_service_pb";
import { PlanType } from "@/types/proto-es/v1/subscription_service_pb";
import { extractInstanceResourceName, hasWorkspacePermissionV2 } from "@/utils";

const PAGE_SIZE = 50;

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
  const instanceStore = useInstanceV1Store();
  const databaseStore = useDatabaseV1Store();
  const refreshServerInfo = useAppStore((state) => state.refreshServerInfo);

  const { instanceLicenseCount, currentPlan } = useSubscriptionState();
  const { activatedInstanceCount } = useServerState();

  const [instances, setInstances] = useState<Instance[]>([]);
  const [selectedNames, setSelectedNames] = useState<Set<string>>(new Set());
  const [nextPageToken, setNextPageToken] = useState("");
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
    async (refresh: boolean, pageToken = "") => {
      const fetchId = ++fetchIdRef.current;
      if (refresh) {
        setLoading(true);
      } else {
        setFetchingMore(true);
      }
      try {
        const result = await instanceStore.fetchInstanceList({
          pageSize: PAGE_SIZE,
          pageToken: refresh ? "" : pageToken,
        });
        if (fetchId !== fetchIdRef.current) {
          return;
        }
        setInstances((prev) =>
          refresh ? result.instances : [...prev, ...result.instances]
        );
        setNextPageToken(result.nextPageToken ?? "");
        applyActivatedSelection(result.instances);
      } finally {
        if (fetchId === fetchIdRef.current) {
          setLoading(false);
          setFetchingMore(false);
        }
      }
    },
    [applyActivatedSelection, instanceStore]
  );

  useEffect(() => {
    if (!open) {
      setInstances([]);
      setSelectedNames(new Set());
      setNextPageToken("");
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
          (await instanceStore.getOrFetchInstanceByName(instanceName)) ??
          instanceStore.getInstanceByName(instanceName);
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

      const updated = await instanceStore.batchUpdateInstances(requests);
      for (const instance of updated) {
        databaseStore.updateDatabaseInstance(instance);
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
    databaseStore,
    instanceStore,
    instances,
    onOpenChange,
    onUpdated,
    processing,
    selectedNames,
    t,
    refreshServerInfo,
  ]);

  const confirmDisabled =
    !canManageSubscription ||
    processing ||
    selectedNames.size > instanceLicenseCount;

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent width="standard">
        <SheetHeader>
          <SheetTitle>
            {t("subscription.instance-assignment.manage-license")}
          </SheetTitle>
        </SheetHeader>
        <SheetBody className="gap-y-5">
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
            <table className="w-full min-w-[36rem] text-sm">
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
            </table>
          </div>

          <PagedTableFooter
            pageSize={PAGE_SIZE}
            pageSizeOptions={[PAGE_SIZE]}
            onPageSizeChange={() => {}}
            hasMore={Boolean(nextPageToken)}
            isFetchingMore={fetchingMore}
            onLoadMore={() => fetchInstances(false, nextPageToken)}
          />
        </SheetBody>
        <SheetFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
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
