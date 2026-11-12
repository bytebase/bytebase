import { Code, ConnectError, createContextValues } from "@connectrpc/connect";
import type { TFunction } from "i18next";
import { ArrowRight, Building2, Folder, KeyRound } from "lucide-react";
import {
  type ReactNode,
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";
import { createPortal } from "react-dom";
import { useTranslation } from "react-i18next";
import { instanceServiceClientConnect } from "@/api";
import { silentContextKey } from "@/api/context-key";
import { EngineIcon } from "@/components/EngineIcon";
import {
  AlertDialog,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogTitle,
} from "@/components/ui/dialog";
import { ResponsiveFormLayout } from "@/components/ui/form";
import { StepIndicator } from "@/components/ui/step-indicator";
import { StickyActionFooter } from "@/components/ui/sticky-action-footer";
import { useUnsavedChangesGuard } from "@/hooks/useUnsavedChangesGuard";
import { useAppStore } from "@/stores/app";
import { Engine } from "@/types/proto-es/v1/common_pb";
import type {
  DiscoverNeonBranchesResponse,
  NeonProjectDiscoveryFailure,
} from "@/types/proto-es/v1/instance_service_pb";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import { NeonConfigureStep } from "./NeonConfigureStep";
import { type NeonConnectFailure, NeonConnectStep } from "./NeonConnectStep";
import { NeonImportStep } from "./NeonImportStep";
import { NeonRowSheet, useEnvironmentTitle } from "./NeonRowSheet";
import {
  type NeonDiscoveryStatus,
  type NeonProjectFailure,
  NeonSelectStep,
} from "./NeonSelectStep";
import {
  applyInstanceChecks,
  branchKey,
  chunkIndices,
  classifyRequestFailure,
  createImportRun,
  defaultSharedSettings,
  draftNeed,
  duplicateInstanceIds,
  effectiveSettings,
  entryRetryable,
  errorReason,
  formatClock,
  formatCountdown,
  importsActivated,
  instanceResourceName,
  mergeDiscovery,
  NEON_BATCH_SIZE,
  type NeonImportDraft,
  type NeonImportRun,
  type NeonInstanceCheck,
  type NeonRequestFailure,
  type NeonSharedKey,
  remainingInstanceCapacity,
  runImportBatches,
  scheduleMode,
  statusReason,
  summarizeRun,
  toImportRequest,
  unsentGroups,
  updateEntryDraft,
} from "./neon-import";

const PAGE_SIZE = 50;

type Step = 0 | 1 | 2 | 3;
type Confirm =
  | { kind: "import" | "stop" | "close" | "leave" }
  | { kind: "retry-unknown"; key: string };

export function NeonStepIndicator({ step }: { step: number }) {
  const { t } = useTranslation();
  return (
    // Clipped because the indicator measures its full-width row off screen,
    // which would otherwise widen the page's scroll area on a narrow screen.
    <div className="overflow-hidden">
      <StepIndicator
        fitOneRow
        ariaLabel={t("instance.neon.steps.label")}
        currentIndex={step}
        steps={[
          { title: t("instance.neon.steps.connect") },
          { title: t("instance.neon.steps.select") },
          { title: t("instance.neon.steps.configure") },
          { title: t("instance.neon.steps.import") },
        ]}
      />
    </div>
  );
}

export function neonDestinationLabel(
  t: TFunction,
  parent: string | undefined,
  project: Project | undefined
): string {
  return parent
    ? t("instance.neon.destination.project", {
        project: project?.title || parent,
      })
    : t("instance.neon.destination.workspace");
}

// Seconds left until a deadline, re-rendering once a second while it is ahead.
function useSecondsUntil(deadline: number | undefined): number {
  const [now, setNow] = useState(() => Date.now());
  useEffect(() => {
    if (deadline === undefined || deadline <= Date.now()) return;
    setNow(Date.now());
    const timer = window.setInterval(() => {
      setNow(Date.now());
      if (Date.now() >= deadline) window.clearInterval(timer);
    }, 1000);
    return () => window.clearInterval(timer);
  }, [deadline]);
  return deadline === undefined
    ? 0
    : Math.max(0, Math.ceil((deadline - now) / 1000));
}

const readLicense = () => {
  const state = useAppStore.getState();
  return {
    instanceCountLimit: state.instanceCountLimit(),
    totalInstanceCount: state.totalInstanceCount(),
    instanceLicenseCount: state.instanceLicenseCount(),
    activatedInstanceCount: state.activatedInstanceCount(),
  };
};

// Reads the instance counts again, so the remaining number reflects instances
// added elsewhere since the page loaded. A failed refresh keeps the last
// counts; the server enforces the limit either way.
const refreshCapacity = async (): Promise<number | undefined> => {
  await useAppStore
    .getState()
    .refreshServerInfo()
    .catch(() => undefined);
  return remainingInstanceCapacity(readLicense());
};

const retryDeadline = (failure: NeonRequestFailure) =>
  failure.kind === "rate-limited" && failure.retryAfterSeconds > 0
    ? Date.now() + failure.retryAfterSeconds * 1000
    : undefined;

const toProjectFailure = (
  failure: NeonProjectDiscoveryFailure
): NeonProjectFailure => ({
  projectId: failure.projectId,
  reason: statusReason(failure.status),
  message: failure.status?.message ?? "",
  retrying: false,
});

export function NeonImportForm({
  parent,
  project,
  active = true,
  footerContainer,
  actionBarContainer,
  onDismiss,
  onStepChange,
  onBusyChange,
  onDirtyChange,
}: {
  parent?: string;
  project?: Project;
  active?: boolean;
  // Where the wizard footer is mounted, so the page keeps one scroll region.
  footerContainer?: HTMLElement | null;
  actionBarContainer?: HTMLElement | null;
  onDismiss: () => void;
  onStepChange?: (step: number) => void;
  onBusyChange?: (busy: boolean) => void;
  // Reports whether closing now would discard a key, selections, or results.
  onDirtyChange?: (dirty: boolean) => void;
}) {
  const { t } = useTranslation();
  const environmentTitle = useEnvironmentTitle();
  const license = {
    instanceCountLimit: useAppStore((state) => state.instanceCountLimit()),
    totalInstanceCount: useAppStore((state) => state.totalInstanceCount()),
    instanceLicenseCount: useAppStore((state) => state.instanceLicenseCount()),
    activatedInstanceCount: useAppStore((state) =>
      state.activatedInstanceCount()
    ),
  };
  const activated = importsActivated(license);
  const intervalEditable = useAppStore((state) =>
    state.hasFeature(PlanFeature.FEATURE_CUSTOM_INSTANCE_SYNC_TIME)
  );
  const schedule = scheduleMode(activated, intervalEditable);
  const capacity = remainingInstanceCapacity(license);
  const destination = neonDestinationLabel(t, parent, project);
  const instancesHref = parent ? `/${parent}/instances` : "/instances";

  const [step, setStep] = useState<Step>(0);
  const [apiKey, setApiKey] = useState("");
  const [projectId, setProjectId] = useState("");
  const [connecting, setConnecting] = useState(false);
  const [connectFailure, setConnectFailure] = useState<NeonConnectFailure>();
  const [connectRetryAt, setConnectRetryAt] = useState<number>();

  const [drafts, setDrafts] = useState<NeonImportDraft[]>([]);
  const [failures, setFailures] = useState<NeonProjectFailure[]>([]);
  const [started, setStarted] = useState(false);
  const [nextPageToken, setNextPageToken] = useState("");
  const [loading, setLoading] = useState(false);
  const [pageFailure, setPageFailure] = useState<NeonRequestFailure>();
  const [pageRetryAt, setPageRetryAt] = useState<number>();
  const [shared, setShared] = useState(defaultSharedSettings);
  const [editingKey, setEditingKey] = useState<string>();

  const [run, setRun] = useState<NeonImportRun>();
  const [activity, setActivity] = useState<"auto" | "retry">();
  const [stopped, setStopped] = useState(false);
  const [checking, setChecking] = useState(false);
  const [confirm, setConfirm] = useState<Confirm>();
  const [closing, setClosing] = useState(false);
  const [preparing, setPreparing] = useState(false);

  // What the loaded rows were discovered with, to skip a repeat of the same
  // discovery.
  const scopeRef = useRef("");
  // Bumped whenever the rows are discarded, so a response that belongs to an
  // earlier traversal or arrives after setup closed is dropped.
  const traversalRef = useRef(0);
  const tokenRef = useRef("");
  const carriedRef = useRef(new Map<string, NeonImportDraft>());
  const draftsRef = useRef(drafts);
  draftsRef.current = drafts;
  const runRef = useRef(run);
  runRef.current = run;
  const connectAbortRef = useRef<AbortController | null>(null);
  const discoveryAbortRef = useRef<AbortController | null>(null);
  const importAbortRef = useRef<AbortController | null>(null);
  const mountedRef = useRef(true);
  useEffect(() => {
    mountedRef.current = true;
    return () => {
      mountedRef.current = false;
      traversalRef.current++;
      connectAbortRef.current?.abort();
      discoveryAbortRef.current?.abort();
      importAbortRef.current?.abort();
    };
  }, []);

  const connectRetryIn = useSecondsUntil(connectRetryAt);
  const pageRetryIn = useSecondsUntil(pageRetryAt);
  const selected = useMemo(
    () => drafts.filter((draft) => draft.selected),
    [drafts]
  );
  const duplicateIds = useMemo(
    () => duplicateInstanceIds(selected),
    [selected]
  );
  // On the Import step an edited instance ID must stay unique among every
  // entry of the run: two equal IDs in one retry are rejected as a whole.
  const runDuplicateIds = useMemo(
    () => duplicateInstanceIds(run?.entries.map((entry) => entry.draft) ?? []),
    [run]
  );
  const needCount = selected.filter((draft) =>
    draftNeed(draft, duplicateIds)
  ).length;
  const overLimit =
    capacity === undefined ? 0 : Math.max(0, selected.length - capacity);
  const summary = run ? summarizeRun(run) : undefined;
  const unresolved = summary
    ? summary.failed + summary.unknown + summary.unsent
    : 0;
  const busy = connecting || loading || !!activity || checking;
  const complete = started && !nextPageToken;

  useEffect(() => {
    if (active) onStepChange?.(step);
  }, [active, step, onStepChange]);
  useEffect(() => {
    if (step === 1 || step === 2) void refreshCapacity();
  }, [step]);
  useEffect(() => {
    onBusyChange?.(busy);
  }, [busy, onBusyChange]);

  const finished = !!run && unresolved === 0 && !activity;
  const dirty =
    !closing && !finished && (!!apiKey || selected.length > 0 || !!run);
  useUnsavedChangesGuard(active && dirty);
  useEffect(() => {
    onDirtyChange?.(dirty);
  }, [dirty, onDirtyChange]);
  // Leaving through this form's own controls has already been confirmed, so
  // the route guard must see a clean form before the navigation starts.
  useEffect(() => {
    if (closing) onDismiss();
  }, [closing, onDismiss]);
  const close = () => {
    traversalRef.current++;
    connectAbortRef.current?.abort();
    discoveryAbortRef.current?.abort();
    importAbortRef.current?.abort();
    setApiKey("");
    setConfirm(undefined);
    setClosing(true);
  };
  useEffect(() => {
    if (finished) setApiKey("");
  }, [finished]);

  const requestOptions = (signal: AbortSignal) => ({
    signal,
    contextValues: createContextValues().set(silentContextKey, true),
  });

  // ---- Discovery ----

  const discover = (
    signal: AbortSignal,
    neonProjectId: string,
    pageToken: string
  ) =>
    instanceServiceClientConnect.discoverNeonBranches(
      {
        parent,
        apiKey,
        projectId: neonProjectId,
        pageSize: PAGE_SIZE,
        pageToken,
      },
      requestOptions(signal)
    );

  const applyPage = (response: DiscoverNeonBranchesResponse) => {
    setDrafts((rows) =>
      mergeDiscovery(rows, response.branches, carriedRef.current)
    );
    setFailures((previous) => {
      const returned = new Set(
        response.branches.map((branch) => branch.projectId)
      );
      const byProject = new Map(
        previous
          .filter((failure) => !returned.has(failure.projectId))
          .map((failure) => [failure.projectId, failure])
      );
      for (const failure of response.failures) {
        byProject.set(failure.projectId, toProjectFailure(failure));
      }
      return [...byProject.values()];
    });
  };

  // Loads pages of the main traversal from its current token. A failed request
  // leaves the token in place, so retrying asks for the same page.
  const loadPages = async (all: boolean) => {
    const controller = new AbortController();
    discoveryAbortRef.current = controller;
    const traversal = traversalRef.current;
    const current = () =>
      mountedRef.current &&
      traversalRef.current === traversal &&
      !controller.signal.aborted;
    setLoading(true);
    setPageFailure(undefined);
    setPageRetryAt(undefined);
    try {
      do {
        const response = await discover(
          controller.signal,
          projectId.trim(),
          tokenRef.current
        );
        if (!current()) return true;
        applyPage(response);
        tokenRef.current = response.nextPageToken;
        setNextPageToken(response.nextPageToken);
        setStarted(true);
      } while (all && tokenRef.current);
      return true;
    } catch (error) {
      if (!current()) return true;
      const failure = classifyRequestFailure(error, !!projectId.trim());
      setPageFailure(failure);
      setPageRetryAt(retryDeadline(failure));
      return false;
    } finally {
      if (current()) setLoading(false);
    }
  };

  const startTraversal = (scope: string) => {
    discoveryAbortRef.current?.abort();
    traversalRef.current++;
    scopeRef.current = scope;
    tokenRef.current = "";
    carriedRef.current = new Map(
      draftsRef.current.map((draft) => [branchKey(draft.branch), draft])
    );
    setDrafts([]);
    setFailures([]);
    setNextPageToken("");
    setStarted(false);
    setLoading(false);
    setPageFailure(undefined);
    setPageRetryAt(undefined);
  };

  const connect = async () => {
    if (connecting || connectRetryIn > 0) return;
    if (!apiKey.trim()) {
      setConnectFailure({ kind: "key-empty" });
      return;
    }
    const scope = JSON.stringify([parent, projectId.trim(), apiKey]);
    if (scopeRef.current === scope && started) {
      setStep(1);
      return;
    }
    setConnecting(true);
    setConnectFailure(undefined);
    setConnectRetryAt(undefined);
    const controller = new AbortController();
    connectAbortRef.current?.abort();
    connectAbortRef.current = controller;
    try {
      const response = await discover(controller.signal, projectId.trim(), "");
      if (!mountedRef.current || controller.signal.aborted) return;
      // A Neon project filter that lists nothing is a mistake in the field
      // just filled in, so it is reported there and the step stays open.
      const filterFailure = response.failures.find(
        (failure) => failure.projectId === projectId.trim()
      );
      if (
        filterFailure &&
        response.branches.length === 0 &&
        !response.nextPageToken
      ) {
        setConnectFailure({
          kind: "project-id-unreadable",
          retryAfterSeconds: 0,
          message: filterFailure.status?.message ?? "",
        });
        return;
      }
      startTraversal(scope);
      applyPage(response);
      tokenRef.current = response.nextPageToken;
      setNextPageToken(response.nextPageToken);
      setStarted(true);
      setStep(1);
    } catch (error) {
      if (!mountedRef.current || controller.signal.aborted) return;
      // The step keeps what was entered and what was already discovered.
      const failure = classifyRequestFailure(error, !!projectId.trim());
      setConnectFailure(failure);
      setConnectRetryAt(retryDeadline(failure));
    } finally {
      if (mountedRef.current) setConnecting(false);
    }
  };

  const restart = () => {
    startTraversal(scopeRef.current);
    void loadPages(false);
  };

  // A Neon project the traversal could not enumerate is listed on its own, with
  // its own page tokens, and merged into the same rows.
  const retryProject = async (neonProjectId: string) => {
    const traversal = traversalRef.current;
    const controller = new AbortController();
    const current = () =>
      mountedRef.current &&
      traversalRef.current === traversal &&
      !controller.signal.aborted;
    const patch = (change: Partial<NeonProjectFailure>) =>
      setFailures((previous) =>
        previous.map((failure) =>
          failure.projectId === neonProjectId
            ? { ...failure, ...change }
            : failure
        )
      );
    patch({ retrying: true });
    try {
      let pageToken = "";
      do {
        const response = await discover(
          controller.signal,
          neonProjectId,
          pageToken
        );
        if (!current()) return;
        const failure = response.failures.find(
          (item) => item.projectId === neonProjectId
        );
        if (failure) {
          patch(toProjectFailure(failure));
          return;
        }
        setDrafts((rows) =>
          mergeDiscovery(rows, response.branches, carriedRef.current)
        );
        pageToken = response.nextPageToken;
      } while (pageToken);
      setFailures((previous) =>
        previous.filter((failure) => failure.projectId !== neonProjectId)
      );
    } catch (error) {
      if (!current()) return;
      const failure = classifyRequestFailure(error, true);
      patch({
        retrying: false,
        reason: errorReason(error),
        message: failure.message,
      });
    }
  };

  // ---- Rows ----

  const updateDrafts = useCallback(
    (
      keys: readonly string[],
      update: (draft: NeonImportDraft) => NeonImportDraft
    ) => {
      const targets = new Set(keys);
      setDrafts((rows) =>
        rows.map((row) =>
          targets.has(branchKey(row.branch)) ? update(row) : row
        )
      );
    },
    []
  );
  const resetOverride = (setting: NeonSharedKey) =>
    setDrafts((rows) => rows.map((row) => ({ ...row, [setting]: undefined })));
  const removeRows = (keys: readonly string[]) => {
    updateDrafts(keys, (row) => ({ ...row, selected: false }));
    if (selected.length <= keys.length) setStep(1);
  };

  // ---- Import ----

  const execute = async (
    base: NeonImportRun,
    groups: readonly (readonly number[])[],
    mode: "auto" | "retry"
  ) => {
    const controller = new AbortController();
    importAbortRef.current = controller;
    setActivity(mode);
    const result = await runImportBatches(
      base,
      groups,
      (entries, signal) =>
        instanceServiceClientConnect.importInstances(
          toImportRequest(
            parent,
            apiKey,
            entries.map((entry) => entry.draft),
            schedule
          ),
          requestOptions(signal ?? controller.signal)
        ),
      (next) => {
        if (mountedRef.current) setRun(next);
      },
      { signal: controller.signal, retry: mode === "retry" }
    );
    if (!mountedRef.current) return;
    setRun(result.run);
    setActivity(undefined);
    // Nothing is left to stop, so a stop confirmation opened earlier closes.
    setConfirm((current) => (current?.kind === "stop" ? undefined : current));
    if (result.run.entries.some((entry) => entry.state === "created")) {
      void useAppStore
        .getState()
        .refreshServerInfo()
        .catch(() => undefined);
    }
  };

  // The limit is checked against fresh counts before the confirmation opens;
  // an import that no longer fits stays on Configure with the reason shown.
  const prepareImport = async () => {
    if (preparing) return;
    setPreparing(true);
    const latest = await refreshCapacity();
    if (!mountedRef.current) return;
    setPreparing(false);
    if (latest === undefined || selected.length <= latest) {
      setConfirm({ kind: "import" });
    }
  };
  const startImport = () => {
    const next = createImportRun(selected, shared);
    setConfirm(undefined);
    setEditingKey(undefined);
    setStopped(false);
    setRun(next);
    setStep(3);
    void execute(next, unsentGroups(next), "auto");
  };
  const stopImport = () => {
    setConfirm(undefined);
    setStopped(true);
    importAbortRef.current?.abort();
  };
  const continueImport = () => {
    const current = runRef.current;
    if (!current || activity) return;
    setStopped(false);
    void execute(current, unsentGroups(current), "auto");
  };
  const retryFailed = () => {
    const current = runRef.current;
    if (!current || activity) return;
    const duplicates = duplicateInstanceIds(
      current.entries.map((entry) => entry.draft)
    );
    const indices = current.entries.flatMap((entry, index) =>
      entryRetryable(entry, duplicates) ? [index] : []
    );
    void execute(current, chunkIndices(indices), "retry");
  };
  const retryUnknown = (key: string) => {
    const current = runRef.current;
    setConfirm(undefined);
    if (!current || activity) return;
    const index = current.entries.findIndex(
      (entry) =>
        entry.key === key && entry.state === "unknown" && entry.checkedAt
    );
    if (index >= 0) void execute(current, [[index]], "retry");
  };
  const checkUnknown = async () => {
    const current = runRef.current;
    if (!current || checking) return;
    setChecking(true);
    const controller = new AbortController();
    const checks = await Promise.all(
      current.entries
        .filter((entry) => entry.state === "unknown")
        .map(async (entry): Promise<NeonInstanceCheck> => {
          try {
            const instance = await instanceServiceClientConnect.getInstance(
              { name: instanceResourceName(parent, entry.draft.instanceId) },
              requestOptions(controller.signal)
            );
            return { key: entry.key, result: "found", instance };
          } catch (error) {
            return {
              key: entry.key,
              result:
                error instanceof ConnectError && error.code === Code.NotFound
                  ? "missing"
                  : "error",
            };
          }
        })
    );
    if (!mountedRef.current) return;
    setRun((latest) =>
      latest ? applyInstanceChecks(latest, checks, new Date()) : latest
    );
    setChecking(false);
    if (checks.some((check) => check.result === "found")) {
      void useAppStore
        .getState()
        .refreshServerInfo()
        .catch(() => undefined);
    }
  };

  // ---- Presentation ----

  const editingDraft =
    step === 3
      ? run?.entries.find(
          (entry) => entry.key === editingKey && entry.state === "failed"
        )?.draft
      : selected.find((draft) => branchKey(draft.branch) === editingKey);
  const selectedProjects = new Set(
    selected.map((draft) => draft.branch.projectId)
  ).size;
  const instances = (count: number) =>
    t("instance.neon.count.instances", { count });
  const branchCount = (count: number) =>
    t("instance.neon.count.branches", { count });

  const environmentSummary = () => {
    const counts = new Map<string, number>();
    for (const draft of selected) {
      const title = environmentTitle(
        effectiveSettings(draft, shared).environment
      );
      counts.set(title, (counts.get(title) ?? 0) + 1);
    }
    return [...counts.entries()]
      .map(([title, count]) => `${title} ${count}`)
      .join(", ");
  };

  const status: NeonDiscoveryStatus = {
    started,
    loading,
    complete,
    pageFailure,
    retryInSeconds: pageRetryIn,
    retryAt: pageRetryIn > 0 ? pageRetryAt : undefined,
  };

  const contextBar = (
    <div className="flex flex-wrap items-center gap-x-4 gap-y-1 text-xs leading-4 text-control-light">
      <span className="inline-flex items-center gap-1.5">
        <EngineIcon engine={Engine.POSTGRES} className="size-4 shrink-0" />
        {t("instance.neon.context.engine")}
      </span>
      <span className="inline-flex items-center gap-1.5">
        {parent ? (
          <Folder className="size-3.5 shrink-0" />
        ) : (
          <Building2 className="size-3.5 shrink-0" />
        )}
        {t("instance.neon.context.destination")}
        <span className="font-medium text-main">{destination}</span>
      </span>
      {!finished && (
        <span className="inline-flex items-center gap-1.5">
          <KeyRound className="size-3.5 shrink-0" />
          {t("instance.neon.context.key")}
        </span>
      )}
    </div>
  );

  const heading = (title: string, description: string, context = true) => (
    <div className="flex flex-col gap-4">
      <NeonStepIndicator step={step} />
      <div className="flex flex-col gap-1">
        <h1
          data-step-heading
          tabIndex={-1}
          className="text-lg font-semibold leading-7 text-main focus:outline-hidden"
        >
          {title}
        </h1>
        <p className="max-w-3xl text-sm leading-5 text-control-light">
          {description}
        </p>
      </div>
      {context && contextBar}
    </div>
  );

  const footerNote = (text: string, tone = "text-control-light") => (
    <span className={`text-xs leading-4 ${tone}`}>{text}</span>
  );
  const footer = ((): ReactNode => {
    switch (step) {
      case 0:
        return (
          <StickyActionFooter
            left={
              <Button
                appearance="outline"
                disabled={connecting}
                onClick={() =>
                  selected.length > 0 ? setConfirm({ kind: "leave" }) : close()
                }
              >
                {t("common.cancel")}
              </Button>
            }
            right={
              <Button
                disabled={connecting || connectRetryIn > 0 || capacity === 0}
                onClick={() => void connect()}
              >
                {connecting
                  ? t("instance.neon.connect.finding")
                  : connectRetryIn > 0
                    ? t("instance.neon.connect.retry-in", {
                        time: formatCountdown(connectRetryIn),
                      })
                    : t("instance.neon.connect.find")}
              </Button>
            }
          />
        );
      case 1:
        return (
          <StickyActionFooter
            left={
              <Button appearance="outline" onClick={() => setStep(0)}>
                {t("common.back")}
              </Button>
            }
            right={
              <>
                {overLimit > 0
                  ? footerNote(
                      t("instance.neon.select.footer-over", {
                        branches: branchCount(overLimit),
                      }),
                      "text-error"
                    )
                  : footerNote(
                      [
                        selected.length > 0
                          ? t("instance.neon.select.footer-selected", {
                              branches: branchCount(selected.length),
                              projects: t("instance.neon.count.neon-projects", {
                                count: selectedProjects,
                              }),
                            })
                          : t("instance.neon.select.footer-none"),
                        capacity !== undefined &&
                          t("instance.neon.select.footer-limit", {
                            count: capacity,
                          }),
                      ]
                        .filter(Boolean)
                        .join(" · ")
                    )}
                <Button
                  disabled={selected.length === 0 || overLimit > 0}
                  onClick={() => setStep(2)}
                >
                  {selected.length > 0
                    ? t("instance.neon.select.configure", {
                        count: selected.length,
                      })
                    : t("instance.neon.select.configure-none")}
                  <ArrowRight className="size-4" />
                </Button>
              </>
            }
          />
        );
      case 2:
        return (
          <StickyActionFooter
            left={
              <Button appearance="outline" onClick={() => setStep(1)}>
                {t("common.back")}
              </Button>
            }
            right={
              <>
                {overLimit > 0
                  ? footerNote(
                      t("instance.neon.configure.footer-over", {
                        instances: instances(overLimit),
                      }),
                      "text-error"
                    )
                  : needCount > 0
                    ? footerNote(
                        t("instance.neon.configure.footer-needs", {
                          count: needCount,
                        }),
                        "text-warning-hover"
                      )
                    : footerNote(
                        `${instances(selected.length)} · ${environmentSummary()}`
                      )}
                <Button
                  disabled={
                    selected.length === 0 ||
                    needCount > 0 ||
                    overLimit > 0 ||
                    preparing
                  }
                  onClick={() => void prepareImport()}
                >
                  {t("instance.neon.import-count", { count: selected.length })}
                </Button>
              </>
            }
          />
        );
      default:
        return activity === "auto" ? (
          <StickyActionFooter
            left={
              <Button
                variant="destructive"
                appearance="outline"
                onClick={() => setConfirm({ kind: "stop" })}
              >
                {t("instance.neon.import.stop")}
              </Button>
            }
            right={footerNote(t("instance.neon.import.stop-note"))}
          />
        ) : (
          <StickyActionFooter
            right={
              <>
                {unresolved > 0 &&
                  footerNote(
                    t("instance.neon.import.unresolved", { count: unresolved }),
                    "text-warning-hover"
                  )}
                <a
                  className="text-sm font-medium text-accent hover:underline max-sm:hidden"
                  href={instancesHref}
                  target="_blank"
                  rel="noreferrer"
                >
                  {t("instance.neon.import.view-instances")}
                </a>
                <Button
                  appearance={unresolved > 0 ? "outline" : "solid"}
                  disabled={!!activity || checking}
                  onClick={() =>
                    unresolved > 0 ? setConfirm({ kind: "close" }) : close()
                  }
                >
                  {t("instance.neon.import.close")}
                </Button>
              </>
            }
          />
        );
    }
  })();

  const retryEntry =
    confirm?.kind === "retry-unknown"
      ? run?.entries.find((entry) => entry.key === confirm.key)
      : undefined;
  const scheduledCount = selected.filter(
    (draft) => effectiveSettings(draft, shared).syncSeconds > 0
  ).length;
  const batchSizes = chunkIndices(selected.map((_, index) => index)).map(
    (chunk) => chunk.length
  );

  return (
    <div hidden={!active} data-testid="neon-import" className="w-full">
      <ResponsiveFormLayout
        layout="vertical"
        density="compact"
        className="flex flex-col gap-4 text-sm text-control"
      >
        {step === 0 && (
          <div className="w-full max-w-5xl">
            <NeonConnectStep
              apiKey={apiKey}
              projectId={projectId}
              busy={connecting}
              failure={connectFailure}
              retryAt={connectRetryIn > 0 ? connectRetryAt : undefined}
              destination={destination}
              onApiKeyChange={(value) => {
                setApiKey(value);
                if (
                  connectFailure?.kind === "key-empty" ||
                  connectFailure?.kind === "key-invalid"
                ) {
                  setConnectFailure(undefined);
                }
              }}
              onProjectIdChange={(value) => {
                setProjectId(value);
                if (
                  connectFailure?.kind === "project-id-required" ||
                  connectFailure?.kind === "project-id-invalid" ||
                  connectFailure?.kind === "project-id-unreadable"
                ) {
                  setConnectFailure(undefined);
                }
              }}
              onSubmit={() => void connect()}
            />
          </div>
        )}
        {step === 1 && (
          <>
            {heading(
              t("instance.neon.steps.select"),
              t("instance.neon.select.description", { destination })
            )}
            <NeonSelectStep
              drafts={drafts}
              status={status}
              failures={failures}
              capacity={capacity}
              onSelect={(keys, on) =>
                updateDrafts(keys, (row) => ({ ...row, selected: on }))
              }
              onLoadMore={() => void loadPages(false)}
              onLoadAll={() => void loadPages(true)}
              onRetryProject={(neonProjectId) =>
                void retryProject(neonProjectId)
              }
              onRestart={restart}
            />
          </>
        )}
        {step === 2 && (
          <>
            {heading(
              t("instance.neon.configure.title", { count: selected.length }),
              t("instance.neon.configure.description"),
              // The table is what this step is for; the destination is stated
              // again in the confirmation.
              false
            )}
            <NeonConfigureStep
              drafts={selected}
              shared={shared}
              capacity={capacity}
              schedule={schedule}
              actionBarContainer={actionBarContainer}
              onSharedChange={(change) =>
                setShared((previous) => ({ ...previous, ...change }))
              }
              onDraftsChange={updateDrafts}
              onResetOverride={resetOverride}
              onRemove={removeRows}
              onOpenRow={setEditingKey}
            />
          </>
        )}
        {step === 3 && run && (
          <>
            <NeonStepIndicator step={step} />
            <NeonImportStep
              run={run}
              stopped={stopped}
              checking={checking}
              schedule={schedule}
              duplicateIds={runDuplicateIds}
              destination={destination}
              instancesHref={instancesHref}
              context={contextBar}
              onCheck={() => void checkUnknown()}
              onRetryUnknown={(key) =>
                setConfirm({ kind: "retry-unknown", key })
              }
              onRetryFailed={retryFailed}
              onContinue={continueImport}
              onEntryChange={(key, update) =>
                setRun((latest) =>
                  latest ? updateEntryDraft(latest, key, update) : latest
                )
              }
              onEditEntry={setEditingKey}
            />
          </>
        )}
      </ResponsiveFormLayout>

      {active &&
        (footerContainer === undefined
          ? footer
          : footerContainer && createPortal(footer, footerContainer))}

      <NeonRowSheet
        open={!!editingDraft}
        draft={editingDraft}
        shared={step === 3 ? undefined : shared}
        duplicateIds={step === 3 ? runDuplicateIds : duplicateIds}
        schedule={schedule}
        instanceScope={
          parent
            ? t("instance.neon.destination.project-instances", {
                project: project?.title || parent,
              })
            : t("instance.neon.destination.workspace-instances")
        }
        onChange={(update) => {
          if (!editingKey) return;
          if (step === 3) {
            setRun((latest) =>
              latest
                ? updateEntryDraft(latest, editingKey, (draft) => ({
                    ...draft,
                    ...update(draft),
                  }))
                : latest
            );
          } else {
            updateDrafts([editingKey], update);
          }
        }}
        onClose={() => setEditingKey(undefined)}
      />

      <Dialog
        open={confirm?.kind === "import"}
        onOpenChange={(open) => !open && setConfirm(undefined)}
      >
        <DialogContent className="w-[32rem] max-w-[calc(100vw-2rem)]">
          <div className="flex flex-col gap-4">
            <DialogTitle>
              {t("instance.neon.confirm.title", { count: selected.length })}
            </DialogTitle>
            <dl className="flex flex-col gap-3 text-sm leading-5 text-control">
              <ConfirmRow term={t("instance.neon.confirm.destination")}>
                {t("instance.neon.confirm.destination-value", { destination })}
              </ConfirmRow>
              <ConfirmRow term={t("common.instances")}>
                {t("instance.neon.confirm.instances-value", {
                  instances: t("instance.neon.count.postgres-instances", {
                    count: selected.length,
                  }),
                  projects: t("instance.neon.count.neon-projects", {
                    count: selectedProjects,
                  }),
                })}
              </ConfirmRow>
              <ConfirmRow term={t("instance.neon.sheet.sync")}>
                {schedule === "off"
                  ? t("instance.neon.sync.unscheduled-help")
                  : scheduledCount === selected.length
                    ? t("instance.neon.confirm.sync-all-scheduled")
                    : [
                        t("instance.neon.confirm.sync-activated"),
                        scheduledCount > 0 &&
                          t("instance.neon.confirm.sync-scheduled", {
                            count: scheduledCount,
                          }),
                        t("instance.neon.confirm.sync-manual", {
                          count: selected.length - scheduledCount,
                        }),
                      ]
                        .filter(Boolean)
                        .join(" ")}
              </ConfirmRow>
              <ConfirmRow term={t("instance.neon.confirm.credentials")}>
                {t("instance.neon.confirm.credentials-value")}
              </ConfirmRow>
            </dl>
            <DialogDescription className="text-xs leading-4 text-control-light">
              {batchSizes.length > 1
                ? t("instance.neon.confirm.batch-split", {
                    batches: batchSizes.length,
                    sizes: batchSizes.join(", "),
                    max: NEON_BATCH_SIZE,
                  })
                : t("instance.neon.confirm.batch-single")}{" "}
              {t("instance.neon.confirm.remain")}
            </DialogDescription>
            <div className="flex flex-wrap justify-end gap-2">
              <Button
                appearance="outline"
                onClick={() => setConfirm(undefined)}
              >
                {t("common.cancel")}
              </Button>
              <Button onClick={startImport}>
                {t("instance.neon.import-count", { count: selected.length })}
              </Button>
            </div>
          </div>
        </DialogContent>
      </Dialog>

      <AlertDialog
        open={confirm?.kind === "stop"}
        onOpenChange={(open) => !open && setConfirm(undefined)}
      >
        <AlertDialogContent>
          <AlertDialogTitle>{t("instance.neon.stop.title")}</AlertDialogTitle>
          <AlertDialogDescription render={<div />}>
            <ul className="flex list-disc flex-col gap-2 pl-5">
              <li>
                {t("instance.neon.stop.created", {
                  count: summary?.created ?? 0,
                })}
              </li>
              <li>
                {t("instance.neon.stop.awaiting", {
                  count: summary?.awaiting ?? 0,
                })}
              </li>
              <li>
                {t("instance.neon.stop.unsent", {
                  count: summary?.unsent ?? 0,
                })}
              </li>
            </ul>
          </AlertDialogDescription>
          <AlertDialogFooter>
            <Button appearance="outline" onClick={() => setConfirm(undefined)}>
              {t("instance.neon.stop.keep")}
            </Button>
            <Button variant="destructive" onClick={stopImport}>
              {t("instance.neon.import.stop")}
            </Button>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      <AlertDialog
        open={confirm?.kind === "close" || confirm?.kind === "leave"}
        onOpenChange={(open) => !open && setConfirm(undefined)}
      >
        <AlertDialogContent>
          <AlertDialogTitle>
            {confirm?.kind === "close"
              ? t("instance.neon.close.title")
              : t("instance.neon.leave.title")}
          </AlertDialogTitle>
          <AlertDialogDescription>
            {confirm?.kind === "close"
              ? t("instance.neon.close.body", { count: unresolved })
              : t("instance.neon.leave.body")}
          </AlertDialogDescription>
          <AlertDialogFooter>
            <Button appearance="outline" onClick={() => setConfirm(undefined)}>
              {t("instance.neon.close.stay")}
            </Button>
            <Button variant="destructive" onClick={close}>
              {confirm?.kind === "close"
                ? t("instance.neon.import.close")
                : t("instance.neon.leave.confirm")}
            </Button>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      <AlertDialog
        open={confirm?.kind === "retry-unknown"}
        onOpenChange={(open) => !open && setConfirm(undefined)}
      >
        <AlertDialogContent>
          <AlertDialogTitle>
            {t("instance.neon.retry-unknown.title", {
              name: retryEntry?.draft.title,
            })}
          </AlertDialogTitle>
          <AlertDialogDescription>
            {t("instance.neon.retry-unknown.body", {
              id: retryEntry?.draft.instanceId,
              time: retryEntry?.checkedAt
                ? formatClock(retryEntry.checkedAt)
                : "",
            })}
          </AlertDialogDescription>
          <AlertDialogFooter>
            <Button appearance="outline" onClick={() => setConfirm(undefined)}>
              {t("common.cancel")}
            </Button>
            <Button
              onClick={() =>
                confirm?.kind === "retry-unknown" && retryUnknown(confirm.key)
              }
            >
              {t("instance.neon.import.unknown.retry")}
            </Button>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}

function ConfirmRow({ term, children }: { term: string; children: ReactNode }) {
  return (
    <div className="grid grid-cols-1 gap-x-4 gap-y-1 sm:grid-cols-[7rem_1fr]">
      <dt className="font-medium text-main">{term}</dt>
      <dd>{children}</dd>
    </div>
  );
}
