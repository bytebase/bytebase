import { Outlet } from "react-router-dom";
import { AuthGate } from "@/react/app/AuthGate";
import { SessionExpiredSurfaceGate } from "@/react/app/SessionExpiredSurfaceGate";
import { Toaster } from "@/react/components/ui/toaster";
import { Watermark } from "@/react/components/Watermark";
import { AgentWindow } from "@/react/plugins/agent/components/AgentWindow";

// Root route element for the react-router app shell. Hosts the global overlays
// that previously lived in `ReactApp.tsx` (Watermark / Toaster / AgentWindow /
// SessionExpiredSurfaceGate) and wraps the routed tree in `<AuthGate>` so the
// session lifecycle (load gate, poll, cross-tab switch, inactivity reminder)
// runs around every page.
export function RootLayout() {
  return (
    <>
      <Watermark />
      <Toaster />
      <AgentWindow />
      <SessionExpiredSurfaceGate />
      <AuthGate>
        <Outlet />
      </AuthGate>
    </>
  );
}
