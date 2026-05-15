import { Toaster } from "@/react/components/ui/toaster";
import { Watermark } from "@/react/components/Watermark";
import { AgentWindow } from "@/react/plugins/agent/components/AgentWindow";
import { SessionExpiredSurfaceGate } from "./SessionExpiredSurfaceGate";

export function ReactApp() {
  return (
    <>
      <Watermark />
      <Toaster />
      <AgentWindow />
      <SessionExpiredSurfaceGate />
    </>
  );
}
