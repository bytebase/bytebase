import { LoaderCircle } from "lucide-react";
import { useEffect, useRef, useState } from "react";
import type { DashboardFrameShellProps } from "@/react/dashboard-shell";
import { useEnsureWorkspaceCommonData } from "@/react/hooks/useEnsureWorkspaceCommonData";
import { useEnvironmentV1Store, useSettingV1Store } from "@/store";
import { Setting_SettingName } from "@/types/proto-es/v1/setting_service_pb";
import { BannersWrapper } from "./BannersWrapper";

// Legacy Pinia bootstrap kept alongside the app-store bootstrap because the
// remaining Vue / shared-util surfaces still read from these Pinia stores
// (e.g. expr.ts environment options, DashboardSidebar's app-feature profile).
// The app store is loaded separately by useEnsureWorkspaceCommonData. Once
// those Pinia readers are migrated this block can go away.
const loadLegacyDashboardState = () => {
  return Promise.all([
    useEnvironmentV1Store().fetchEnvironments(),
    useSettingV1Store().getOrFetchSettingByName(
      Setting_SettingName.WORKSPACE_PROFILE
    ),
  ]);
};

export function DashboardFrameShell({ onReady }: DashboardFrameShellProps) {
  const bannerRef = useRef<HTMLDivElement>(null);
  const bodyRef = useRef<HTMLDivElement>(null);
  const [legacyReady, setLegacyReady] = useState(false);
  const commonDataReady = useEnsureWorkspaceCommonData();

  useEffect(() => {
    let mounted = true;
    void loadLegacyDashboardState()
      .catch(() => undefined)
      .then(() => {
        if (mounted) {
          setLegacyReady(true);
        }
      });
    return () => {
      mounted = false;
    };
  }, []);

  const initialized = commonDataReady && legacyReady;

  useEffect(() => {
    if (!initialized) return;
    onReady?.({
      banner: bannerRef.current,
      body: bodyRef.current,
    });
  }, [initialized, onReady]);

  return (
    <div className="relative flex h-screen flex-col overflow-hidden">
      <div ref={bannerRef}>
        <BannersWrapper />
      </div>
      <div ref={bodyRef} className="min-h-0 flex-1" />
      {!initialized ? (
        <div className="absolute inset-0 z-10 flex items-center justify-center bg-white">
          <LoaderCircle className="size-6 animate-spin text-accent" />
        </div>
      ) : null}
    </div>
  );
}
