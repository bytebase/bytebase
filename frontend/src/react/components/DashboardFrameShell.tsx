import { LoaderCircle } from "lucide-react";
import { useEffect, useRef, useState } from "react";
import type { DashboardFrameShellProps } from "@/react/dashboard-shell";
import { useAppStore } from "@/react/stores/app";
import { useEnvironmentV1Store, useSettingV1Store } from "@/store";
import { Setting_SettingName } from "@/types/proto-es/v1/setting_service_pb";
import { BannersWrapper } from "./BannersWrapper";

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
  const [initialized, setInitialized] = useState(false);
  const loadEnvironmentList = useAppStore((state) => state.loadEnvironmentList);
  const loadWorkspaceProfile = useAppStore(
    (state) => state.loadWorkspaceProfile
  );

  useEffect(() => {
    let mounted = true;
    void Promise.all([
      loadEnvironmentList(),
      loadWorkspaceProfile(),
      loadLegacyDashboardState(),
    ])
      .catch(() => undefined)
      .then(() => {
        if (mounted) {
          setInitialized(true);
        }
      });
    return () => {
      mounted = false;
    };
  }, [loadEnvironmentList, loadWorkspaceProfile]);

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
