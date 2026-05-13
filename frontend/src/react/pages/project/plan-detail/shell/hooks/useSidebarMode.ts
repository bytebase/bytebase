import { useEffect, useRef, useState } from "react";
import {
  MOBILE_BREAKPOINT_PX,
  SIDEBAR_WIDTH_NARROW_PX,
  SIDEBAR_WIDTH_WIDE_PX,
  WIDE_SIDEBAR_BREAKPOINT_PX,
} from "../constants";

export type PlanDetailSidebarMode = "NONE" | "DESKTOP" | "MOBILE";

export function useSidebarMode(pageHost: HTMLElement | null) {
  const [sidebarMode, setSidebarMode] = useState<PlanDetailSidebarMode>("NONE");
  const [containerWidth, setContainerWidth] = useState(0);
  const [sidebarWidth, setSidebarWidth] = useState(0);
  const [isMobileSidebarOpen, setMobileSidebarOpen] = useState(false);
  const isMobileSidebarOpenRef = useRef(isMobileSidebarOpen);
  isMobileSidebarOpenRef.current = isMobileSidebarOpen;

  useEffect(() => {
    if (!pageHost) {
      setSidebarMode("NONE");
      setContainerWidth(0);
      setSidebarWidth(0);
      setMobileSidebarOpen(false);
      return;
    }

    const updateSidebar = () => {
      const width = pageHost.getBoundingClientRect().width;
      const mode: PlanDetailSidebarMode =
        width === 0
          ? "NONE"
          : width < MOBILE_BREAKPOINT_PX
            ? "MOBILE"
            : "DESKTOP";
      setContainerWidth(width);
      setSidebarWidth(
        width < WIDE_SIDEBAR_BREAKPOINT_PX
          ? SIDEBAR_WIDTH_NARROW_PX
          : SIDEBAR_WIDTH_WIDE_PX
      );
      setSidebarMode(mode);
      if (mode !== "MOBILE") {
        setMobileSidebarOpen(false);
      }
    };

    updateSidebar();
    const observer = new ResizeObserver(() => updateSidebar());
    observer.observe(pageHost);

    return () => observer.disconnect();
  }, [pageHost]);

  return {
    sidebarMode,
    containerWidth,
    sidebarWidth,
    isMobileSidebarOpen,
    setMobileSidebarOpen,
  };
}
