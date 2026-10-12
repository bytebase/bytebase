import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { router } from "@/app/router";
import { PROJECT_V1_ROUTE_DATABASES } from "@/app/router/handles";
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
import { ResponsiveFormLayout } from "@/components/ui/form";
import { useUnsavedChangesGuard } from "@/hooks/useUnsavedChangesGuard";
import { pushNotification } from "@/stores";
import { useAppStore } from "@/stores/app";
import type { Instance } from "@/types/proto-es/v1/instance_service_pb";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import { isValidProjectName } from "@/types/v1/project";

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
  const sampleInstanceName = useAppStore(
    (state) => state.serverInfo?.sample?.instances[0]?.instance
  );
  const totalInstanceCount = useAppStore((state) => state.totalInstanceCount());
  const canPrepareSampleProjectInstance =
    sampleAvailable &&
    !sampleInstanceName &&
    totalInstanceCount === 0 &&
    !!parent &&
    isValidProjectName(parent);
  // A project-owned sample is reached through its project's databases.
  const sampleProjectId = /^projects\/([^/]+)\/instances\//.exec(
    sampleInstanceName ?? ""
  )?.[1];
  const openSampleInstance = useCallback(() => {
    if (!sampleProjectId) return;
    router.push({
      name: PROJECT_V1_ROUTE_DATABASES,
      params: { projectId: sampleProjectId },
    });
  }, [sampleProjectId]);

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
        onSampleInstancePrepared={
          canPrepareSampleProjectInstance ? onCreated : undefined
        }
        onOpenSampleInstance={sampleProjectId ? openSampleInstance : undefined}
      >
        <CreateInstanceFormInner
          isSaaSMode={isSaaSMode}
          onCreated={onCreated}
        />
      </InstanceFormProvider>
    </div>
  );
}

function CreateInstanceFormInner({
  isSaaSMode,
  onCreated,
}: {
  isSaaSMode: boolean;
  onCreated: (instance: Instance) => void;
}) {
  const { t } = useTranslation();
  const { basicInfo, state, sampleInstance } = useInstanceFormContext();
  const currentEngine = basicInfo.engine;

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

  return (
    <div
      ref={layoutRef}
      className="grid h-full w-full min-h-0 min-w-0"
      style={layoutStyle}
      onClick={handleDockedInfoPanelOutsideClick}
    >
      <div className="min-w-0 min-h-0 flex-1 flex flex-col">
        {/* Body */}
        <div className="flex-1 min-h-0 overflow-auto">
          <ResponsiveFormLayout className="px-4 py-4 sm:px-6 gap-y-4 flex flex-col">
            {sampleInstance?.kind === "prepare" && (
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
                  disabled={sampleInstance.pending || state.isRequesting}
                  onClick={() => sampleInstance.run("instance_creation")}
                >
                  {sampleInstance.pending
                    ? t("instance.preparing-sample-instance")
                    : isSaaSMode
                      ? t("instance.use-sample-instance")
                      : t("instance.use-sample-instance-self-host")}
                </Button>
              </Alert>
            )}
            <InstanceFormBody onOpenInfoPanel={handleOpenInfoPanel} />
          </ResponsiveFormLayout>
        </div>

        <InstanceFormButtons onCreated={onCreated} />
        <UnsavedChangesGuard />
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

function UnsavedChangesGuard() {
  const { state, valueChanged } = useInstanceFormContext();
  useUnsavedChangesGuard(valueChanged && !state.isRequesting);
  return null;
}
