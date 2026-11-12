import { Building2, Folder } from "lucide-react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { createBehaviorMetric } from "@/app/analytics/behavior";
import { behaviorAnalytics } from "@/app/analytics/provider";
import { EngineIcon } from "@/components/EngineIcon";
import {
  InfoPanel,
  InfoPanelContent,
  InstanceFormBody,
  InstanceFormButtons,
  InstanceFormProvider,
  useInstanceFormContext,
} from "@/components/instance";
import type { InfoSection } from "@/components/instance/info-content";
import { Alert } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { FormField, ResponsiveFormLayout } from "@/components/ui/form";
import { RadioGroup } from "@/components/ui/radio-group";
import { useUnsavedChangesGuard } from "@/hooks/useUnsavedChangesGuard";
import { openAgentWithPrompt } from "@/modules/agent/logic/instanceConnectionHelp";
import { useAgentStore } from "@/modules/agent/store/agent";
import { pushNotification } from "@/stores";
import { useAppStore } from "@/stores/app";
import { Engine } from "@/types/proto-es/v1/common_pb";
import { type Instance, Vendor } from "@/types/proto-es/v1/instance_service_pb";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import { isValidProjectName } from "@/types/v1/project";
import { extractGrpcErrorMessage } from "@/utils/connect";
import { ChoiceCard } from "./ChoiceCard";
import {
  NeonImportForm,
  NeonStepIndicator,
  neonDestinationLabel,
} from "./NeonImportForm";
import { NeonWarning } from "./NeonWarning";
import { remainingInstanceCapacity } from "./neon-import";

const MIN_DOCKED_MAIN_WIDTH = 700;
const DOCKED_INFO_RAIL_WIDTH = 500;
const DOCKED_INFO_RAIL_GAP = 16;
const MIN_DOCKED_LAYOUT_WIDTH =
  MIN_DOCKED_MAIN_WIDTH + DOCKED_INFO_RAIL_WIDTH + DOCKED_INFO_RAIL_GAP;

interface CreateInstanceViewProps {
  parent?: string;
  project?: Project;
  onDismiss: () => void;
  onCreated: (instance: Instance) => void;
}

export function CreateInstanceView({
  parent,
  project,
  onDismiss,
  onCreated,
}: CreateInstanceViewProps) {
  const { t } = useTranslation();
  const isSaaSMode = useAppStore((state) => state.isSaaSMode());
  const sampleAvailable = useAppStore(
    (state) => state.serverInfo?.sample?.available ?? false
  );
  const sampleProvisioned = useAppStore(
    (state) => (state.serverInfo?.sample?.instances.length ?? 0) > 0
  );
  const totalInstanceCount = useAppStore((state) => state.totalInstanceCount());
  const canPrepareSampleProjectInstance =
    sampleAvailable &&
    !sampleProvisioned &&
    totalInstanceCount === 0 &&
    !!parent &&
    isValidProjectName(parent);

  // Check instance limit on mount
  useEffect(() => {
    const store = useAppStore.getState();
    if (store.instanceCountLimit() <= store.activatedInstanceCount()) {
      pushNotification({
        module: "bytebase",
        style: "CRITICAL",
        title: t("subscription.usage.instance-count.title"),
        description: t("subscription.usage.instance-count.runoutof", {
          total: store.instanceCountLimit(),
        }),
      });
      onDismiss();
    }
  }, [canPrepareSampleProjectInstance, onDismiss, t]);

  return (
    <div className="h-full overflow-hidden">
      <InstanceFormProvider
        parent={parent}
        project={project}
        onDismiss={onDismiss}
      >
        <CreateInstanceFormInner
          canPrepareSampleProjectInstance={canPrepareSampleProjectInstance}
          isSaaSMode={isSaaSMode}
          onCreated={onCreated}
          parent={parent}
        />
      </InstanceFormProvider>
    </div>
  );
}

function CreateInstanceFormInner({
  canPrepareSampleProjectInstance,
  isSaaSMode,
  onCreated,
  parent,
}: {
  canPrepareSampleProjectInstance: boolean;
  isSaaSMode: boolean;
  onCreated: (instance: Instance) => void;
  parent?: string;
}) {
  const { t } = useTranslation();
  const { basicInfo, state, setState, project, onDismiss } =
    useInstanceFormContext();
  const [method, setMethod] = useState<"discover" | "manual">("discover");
  const [neonStep, setNeonStep] = useState(0);
  const [neonBusy, setNeonBusy] = useState(false);
  const [neonDirty, setNeonDirty] = useState(false);
  const [pickerOpen, setPickerOpen] = useState(false);
  const [footerSlot, setFooterSlot] = useState<HTMLDivElement | null>(null);
  const [actionBarSlot, setActionBarSlot] = useState<HTMLDivElement | null>(
    null
  );
  const isNeon =
    basicInfo.engine === Engine.POSTGRES && basicInfo.vendor === Vendor.NEON;
  const showNeonImport = isNeon && method === "discover";
  const neonCapacity = remainingInstanceCapacity({
    instanceCountLimit: useAppStore((state) => state.instanceCountLimit()),
    totalInstanceCount: useAppStore((state) => state.totalInstanceCount()),
    instanceLicenseCount: useAppStore((state) => state.instanceLicenseCount()),
    activatedInstanceCount: useAppStore((state) =>
      state.activatedInstanceCount()
    ),
  });
  // Each wizard step opens at its top, whatever the previous one was scrolled
  // to, with focus on its heading so the new step is announced.
  const scrollRef = useRef<HTMLDivElement>(null);
  const shownStepRef = useRef(neonStep);
  useEffect(() => {
    const scroller = scrollRef.current;
    if (!scroller) return;
    scroller.scrollTop = 0;
    if (shownStepRef.current === neonStep) return;
    shownStepRef.current = neonStep;
    [...scroller.querySelectorAll<HTMLElement>("[data-step-heading]")]
      .find((heading) => !heading.closest("[hidden]"))
      ?.focus({ preventScroll: true });
  }, [neonStep]);
  // The discovery wizard stays mounted once opened, so its API key and
  // selections survive a switch to another method or hosting choice.
  const [neonVisited, setNeonVisited] = useState(false);
  useEffect(() => {
    if (!isNeon) return;
    setNeonVisited(true);
    setPickerOpen(false);
  }, [isNeon]);
  const currentEngine = basicInfo.engine;
  const prepareSampleProjectInstance = useAppStore(
    (state) => state.prepareSampleProjectInstance
  );
  const [
    isPreparingSampleProjectInstance,
    setIsPreparingSampleProjectInstance,
  ] = useState(false);

  const handleAskAgent = useCallback((prompt?: string) => {
    if (prompt) {
      openAgentWithPrompt(prompt);
    } else {
      useAgentStore.getState().open();
    }
  }, []);

  const [activeInfoSection, setActiveInfoSection] = useState<
    InfoSection | undefined
  >();
  const layoutRef = useRef<HTMLDivElement>(null);
  const [layoutWidth, setLayoutWidth] = useState(0);

  useEffect(() => {
    const el = layoutRef.current;
    if (!el) return;
    const observer = new ResizeObserver((entries) => {
      for (const entry of entries) {
        setLayoutWidth(entry.contentRect.width);
      }
    });
    observer.observe(el);
    return () => observer.disconnect();
  }, []);

  const canUseDockedInfoLayout = layoutWidth >= MIN_DOCKED_LAYOUT_WIDTH;
  const showDockedInfoPanel = !!activeInfoSection && canUseDockedInfoLayout;
  const showOverlayInfoPanel = !!activeInfoSection && !canUseDockedInfoLayout;

  const layoutStyle = useMemo(() => {
    if (!canUseDockedInfoLayout || !showDockedInfoPanel) {
      return {
        gridTemplateColumns: "minmax(0, 1fr)",
        columnGap: "0rem",
      };
    }
    return {
      gridTemplateColumns: `minmax(${MIN_DOCKED_MAIN_WIDTH}px, 1fr) ${DOCKED_INFO_RAIL_WIDTH}px`,
      columnGap: `${DOCKED_INFO_RAIL_GAP}px`,
    };
  }, [canUseDockedInfoLayout, showDockedInfoPanel]);

  const infoPanelTitle = useMemo(() => {
    if (!activeInfoSection) return "";
    const titleMap: Record<InfoSection, string> = {
      host: t("instance.hostname"),
      port: t("instance.port"),
      authentication: t("instance.connection-info"),
      ssl: t("data-source.ssl-connection"),
      ssh: t("data-source.ssh-connection"),
      database: t("common.database"),
      "sync-databases": t("instance.sync-databases.sync-all"),
    };
    return titleMap[activeInfoSection] ?? "";
  }, [activeInfoSection, t]);

  const handleDockedInfoPanelOutsideClick = useCallback(
    (event: React.MouseEvent) => {
      if (!showDockedInfoPanel) return;
      const target = event.target;
      if (!(target instanceof Element)) return;
      if (target.closest("[data-info-panel-docked='true']")) return;
      setActiveInfoSection(undefined);
    },
    [showDockedInfoPanel]
  );

  const handleOpenInfoPanel = useCallback((section: InfoSection) => {
    setActiveInfoSection(section);
  }, []);

  const handlePrepareSampleProjectInstance = useCallback(async () => {
    if (!parent || isPreparingSampleProjectInstance || state.isRequesting) {
      return;
    }
    behaviorAnalytics.captureMetric(
      createBehaviorMetric("sample instance requested", {
        properties: { source: "instance_creation" },
      })
    );
    setIsPreparingSampleProjectInstance(true);
    setState((prev) => ({ ...prev, isRequesting: true }));
    try {
      onCreated(await prepareSampleProjectInstance(parent));
    } catch (error) {
      pushNotification({
        module: "bytebase",
        style: "CRITICAL",
        title: t("instance.prepare-sample-instance-failed"),
        description: extractGrpcErrorMessage(error),
      });
    } finally {
      setIsPreparingSampleProjectInstance(false);
      setState((prev) => ({ ...prev, isRequesting: false }));
    }
  }, [
    isPreparingSampleProjectInstance,
    onCreated,
    parent,
    prepareSampleProjectInstance,
    setState,
    state.isRequesting,
    t,
  ]);

  return (
    <div
      ref={layoutRef}
      className="grid h-full w-full min-h-0 min-w-0"
      style={layoutStyle}
      onClick={handleDockedInfoPanelOutsideClick}
    >
      <div className="min-w-0 min-h-0 flex-1 flex flex-col">
        {/* Body */}
        <div ref={scrollRef} className="flex-1 min-h-0 overflow-auto">
          <ResponsiveFormLayout className="px-4 sm:px-6 py-4 gap-y-4 flex flex-col">
            {canPrepareSampleProjectInstance && !isNeon && (
              <Alert
                title={t("instance.sample-project-instance-title")}
                description={
                  isSaaSMode
                    ? t("instance.sample-project-instance-description")
                    : t(
                        "instance.sample-project-instance-description-self-host"
                      )
                }
              >
                <Button
                  className="mt-3"
                  disabled={
                    isPreparingSampleProjectInstance || state.isRequesting
                  }
                  onClick={handlePrepareSampleProjectInstance}
                >
                  {isPreparingSampleProjectInstance
                    ? t("instance.preparing-sample-instance")
                    : isSaaSMode
                      ? t("instance.use-sample-instance")
                      : t("instance.use-sample-instance-self-host")}
                </Button>
              </Alert>
            )}
            <div
              hidden={showNeonImport && neonStep > 0}
              className="w-full max-w-5xl flex flex-col gap-4"
            >
              {isNeon && (
                <>
                  {showNeonImport && <NeonStepIndicator step={0} />}
                  <div className="flex flex-col gap-1">
                    <h1
                      data-step-heading
                      tabIndex={-1}
                      className="text-lg font-semibold leading-7 text-main focus:outline-hidden"
                    >
                      {t("instance.neon.title")}
                    </h1>
                    <p className="flex flex-wrap items-center gap-x-2 gap-y-1 text-sm leading-5 text-control-light">
                      <span className="inline-flex items-center gap-1.5">
                        <EngineIcon
                          engine={Engine.POSTGRES}
                          className="size-4 shrink-0"
                        />
                        {t("instance.neon.entry.summary")}
                      </span>
                      {!pickerOpen && (
                        <Button
                          appearance="link"
                          size="sm"
                          className="h-auto p-0 text-sm"
                          disabled={neonBusy || state.isRequesting}
                          onClick={() => setPickerOpen(true)}
                        >
                          {t("instance.neon.entry.change")}
                        </Button>
                      )}
                    </p>
                  </div>
                </>
              )}
              <div hidden={isNeon && !pickerOpen}>
                <InstanceFormBody
                  engineOnly
                  disabled={neonBusy || state.isRequesting}
                />
              </div>
              {isNeon && (
                <>
                  <div className="flex items-start gap-3 rounded-sm border border-block-border p-3">
                    {parent ? (
                      <Folder className="mt-0.5 size-4 shrink-0 text-control-light" />
                    ) : (
                      <Building2 className="mt-0.5 size-4 shrink-0 text-control-light" />
                    )}
                    <div className="min-w-0">
                      <div className="text-sm font-medium text-main">
                        {t("instance.neon.destination.adding", {
                          destination: neonDestinationLabel(t, parent, project),
                        })}
                      </div>
                      <p className="max-w-3xl text-xs leading-4 text-control-light">
                        {parent
                          ? t("instance.neon.destination.project-help", {
                              project: project?.title || parent,
                            })
                          : t("instance.neon.destination.workspace-help")}{" "}
                        {t("instance.neon.destination.neon-project-note")}
                      </p>
                    </div>
                  </div>
                  {neonCapacity === 0 && (
                    <NeonWarning
                      title={t("instance.neon.limit-reached.title")}
                      description={t("instance.neon.limit-reached.body")}
                    />
                  )}
                  <FormField
                    layout="vertical"
                    title={t("instance.neon.method.title")}
                  >
                    <RadioGroup
                      aria-label={t("instance.neon.method.title")}
                      value={method}
                      disabled={neonBusy || state.isRequesting}
                      onValueChange={(value) =>
                        setMethod(value === "manual" ? "manual" : "discover")
                      }
                      className="grid w-full grid-cols-1 items-stretch gap-3 sm:grid-cols-2"
                    >
                      <ChoiceCard
                        value="discover"
                        title={t("instance.neon.method.discover")}
                        description={t("instance.neon.method.discover-help")}
                      />
                      <ChoiceCard
                        value="manual"
                        title={t("instance.neon.method.manual")}
                        description={t("instance.neon.method.manual-help")}
                      />
                    </RadioGroup>
                  </FormField>
                </>
              )}
            </div>
            <div hidden={showNeonImport}>
              <InstanceFormBody
                hideEngine
                onOpenInfoPanel={handleOpenInfoPanel}
                onAskAgent={() => handleAskAgent()}
              />
            </div>
            {(isNeon || neonVisited) && (
              <div hidden={!showNeonImport}>
                <NeonImportForm
                  parent={parent}
                  project={project}
                  active={showNeonImport}
                  footerContainer={footerSlot}
                  actionBarContainer={actionBarSlot}
                  onDismiss={() => onDismiss?.()}
                  onStepChange={setNeonStep}
                  onBusyChange={setNeonBusy}
                  onDirtyChange={setNeonDirty}
                />
              </div>
            )}
          </ResponsiveFormLayout>
        </div>

        <div ref={setActionBarSlot} className="relative" />
        <div ref={setFooterSlot} />
        {!showNeonImport && (
          <InstanceFormButtons
            onCreated={onCreated}
            onAskAgent={handleAskAgent}
          />
        )}
        {!showNeonImport && <UnsavedChangesGuard hasImportDraft={neonDirty} />}
      </div>

      {/* Docked info panel */}
      <InfoPanel
        visible={showDockedInfoPanel}
        mode="docked"
        title={infoPanelTitle}
        onClose={() => setActiveInfoSection(undefined)}
      >
        {activeInfoSection && (
          <InfoPanelContent
            engine={currentEngine}
            section={activeInfoSection}
          />
        )}
      </InfoPanel>

      {/* Overlay info panel */}
      <InfoPanel
        visible={showOverlayInfoPanel}
        mode="overlay"
        title={infoPanelTitle}
        onClose={() => setActiveInfoSection(undefined)}
      >
        {activeInfoSection && (
          <InfoPanelContent
            engine={currentEngine}
            section={activeInfoSection}
          />
        )}
      </InfoPanel>
    </div>
  );
}

function UnsavedChangesGuard({
  hasImportDraft = false,
}: {
  hasImportDraft?: boolean;
}) {
  const { state, valueChanged } = useInstanceFormContext();
  useUnsavedChangesGuard(
    (valueChanged || hasImportDraft) && !state.isRequesting
  );
  return null;
}
