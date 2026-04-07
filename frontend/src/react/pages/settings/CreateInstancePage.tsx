import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import {
  InfoPanel,
  InfoPanelContent,
  InstanceFormBody,
  InstanceFormButtons,
  InstanceFormProvider,
  useInstanceFormContext,
} from "@/react/components/instance";
import type { InfoSection } from "@/react/components/instance/info-content";
import { router } from "@/router";
import { INSTANCE_ROUTE_DASHBOARD } from "@/router/dashboard/workspaceRoutes";
import {
  pushNotification,
  useActuatorV1Store,
  useSubscriptionV1Store,
} from "@/store";

const MIN_DOCKED_MAIN_WIDTH = 700;
const DOCKED_INFO_RAIL_WIDTH = 500;
const DOCKED_INFO_RAIL_GAP = 16;
const MIN_DOCKED_LAYOUT_WIDTH =
  MIN_DOCKED_MAIN_WIDTH + DOCKED_INFO_RAIL_WIDTH + DOCKED_INFO_RAIL_GAP;

export function CreateInstancePage() {
  const { t } = useTranslation();
  const subscriptionStore = useSubscriptionV1Store();
  const actuatorStore = useActuatorV1Store();

  // Check instance limit on mount
  useEffect(() => {
    if (
      subscriptionStore.instanceCountLimit <=
      actuatorStore.activatedInstanceCount
    ) {
      pushNotification({
        module: "bytebase",
        style: "CRITICAL",
        title: t("subscription.usage.instance-count.title"),
        description: t("subscription.usage.instance-count.runoutof", {
          total: subscriptionStore.instanceCountLimit,
        }),
      });
      router.push({ name: INSTANCE_ROUTE_DASHBOARD });
    }
  }, [subscriptionStore, actuatorStore, t]);

  const goBack = useCallback(() => {
    router.push({ name: INSTANCE_ROUTE_DASHBOARD });
  }, []);

  return (
    <div className="h-full overflow-hidden px-4 sm:px-6">
      <InstanceFormProvider onDismiss={goBack}>
        <CreateInstanceFormInner />
      </InstanceFormProvider>
    </div>
  );
}

function CreateInstanceFormInner() {
  const { t } = useTranslation();
  const ctx = useInstanceFormContext();
  const currentEngine = ctx.basicInfo.engine;

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
        {/* Header */}
        <div className="sticky top-0 z-10 bg-white border-b border-block-border py-4">
          <h1 className="text-lg font-medium">
            {t("quick-action.add-instance")}
          </h1>
        </div>

        {/* Body */}
        <div className="flex-1 min-h-0 overflow-auto py-4">
          <InstanceFormBody onOpenInfoPanel={handleOpenInfoPanel} />
        </div>

        {/* Sticky footer */}
        <div className="sticky bottom-0 z-10 bg-white">
          <InstanceFormButtons />
        </div>
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
