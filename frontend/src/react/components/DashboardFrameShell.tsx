import { useEffect, useRef } from "react";
import type { DashboardFrameShellProps } from "@/react/dashboard-shell";

export function DashboardFrameShell({ onReady }: DashboardFrameShellProps) {
  const bannerRef = useRef<HTMLDivElement>(null);
  const bodyRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    onReady?.({
      banner: bannerRef.current,
      body: bodyRef.current,
    });
  }, [onReady]);

  return (
    <div className="relative flex h-screen flex-col overflow-hidden">
      <div ref={bannerRef} />
      <div ref={bodyRef} className="min-h-0 flex-1" />
    </div>
  );
}
