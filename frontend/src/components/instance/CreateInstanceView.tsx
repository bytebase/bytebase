import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
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
  const canPrepareSampleProjectInstance =
    isSaaSMode && !!parent && isValidProjectName(parent);

  // Check instance limit on mount
  useEffect(() => {
    if (canPrepareSampleProjectInstance) return;
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
          onCreated={onCreated}
          parent={parent}
        />
      </InstanceFormProvider>
    </div>
  );
}

function CreateInstanceFormInner({
  canPrepareSampleProjectInstance,
  onCreated,
  parent,
}: {
  canPrepareSampleProjectInstance: boolean;
  onCreated: (instance: Instance) => void;
  parent?: string;
}) {
  const { t } = useTranslation();
  const ctx = useInstanceFormContext();
  const currentEngine = ctx.basicInfo.engine;
  const prepareSampleProjectInstance = useAppStore(
    (state) => state.prepareSampleProjectInstance
  );
  const [
    isPreparingSampleProjectInstance,
    setIsPreparingSampleProjectInstance,
  ] = useState(false);

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
      host: t("instance.host-or-socket"),
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
    if (!parent || isPreparingSampleProjectInstance) return;
    setIsPreparingSampleProjectInstance(true);
    try {
      onCreated(await prepareSampleProjectInstance(parent));
    } catch {
      pushNotification({
        module: "bytebase",
        style: "CRITICAL",
        title: t("instance.prepare-sample-instance-failed"),
      });
    } finally {
      setIsPreparingSampleProjectInstance(false);
    }
  }, [
    isPreparingSampleProjectInstance,
    onCreated,
    parent,
    prepareSampleProjectInstance,
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
        <div className="flex-1 min-h-0 overflow-auto">
          <div className="px-4 py-4 sm:px-6">
            {canPrepareSampleProjectInstance && (
              <Alert
                title={t("instance.sample-project-instance-title")}
                description={t("instance.sample-project-instance-description")}
              >
                <Button
                  className="mt-3"
                  disabled={isPreparingSampleProjectInstance}
                  onClick={handlePrepareSampleProjectInstance}
                >
                  {isPreparingSampleProjectInstance
                    ? t("instance.preparing-sample-instance")
                    : t("instance.use-sample-instance")}
                </Button>
              </Alert>
            )}
            <InstanceFormBody onOpenInfoPanel={handleOpenInfoPanel} />
          </div>
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
