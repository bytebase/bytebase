import { createContext, type ReactNode, useContext, useState } from "react";
import { createPortal } from "react-dom";
import { Outlet, useMatches } from "react-router";
import { AUTH_SETUP_MODULE, AUTH_SIGNUP_MODULE } from "@/app/router/handles";
import signinImage from "@/assets/illustration/signin.webp";
import signupImage from "@/assets/illustration/signup.webp";
import { useAppStore } from "@/stores/app";
import { PlanType } from "@/types/proto-es/v1/subscription_service_pb";

// The left column's element on a route that hands that column to its page.
export const SplashPanelContext = createContext<HTMLElement | null>(null);

// Renders `children` in the layout's left column. On a route that keeps the
// branding illustration there, it renders nothing.
export function SplashPanel({ children }: { readonly children: ReactNode }) {
  const slot = useContext(SplashPanelContext);
  return slot ? createPortal(children, slot) : null;
}

// Two-column auth chrome: the left column holds the branding illustration
// (shown unless the workspace is on a non-trialing enterprise plan), or on the
// setup route the wizard's own panel; the routed auth page is on the right.
//
// The right column owns the gutter and the page's share of the viewport (all
// of it below lg, half from lg). A routed page fills the column up to its own
// cap (`w-full max-w-*`) and adds no width fraction or side padding of its
// own, which would compound with the column's.
export function SplashLayout() {
  const matches = useMatches();
  const currentRouteName = (
    matches.at(-1)?.handle as { name?: string } | undefined
  )?.name;
  const pageOwnsPanel = currentRouteName === AUTH_SETUP_MODULE;
  const [panelSlot, setPanelSlot] = useState<HTMLElement | null>(null);

  const [showBrandingImage] = useState(() => {
    const store = useAppStore.getState();
    const currentPlan = store.currentPlan();
    return currentPlan !== PlanType.ENTERPRISE || store.isTrialing();
  });

  return (
    <div className="min-h-screen overflow-hidden flex">
      {pageOwnsPanel ? (
        <div
          ref={setPanelSlot}
          className="hidden bg-control-bg lg:flex relative w-0 flex-1 items-center justify-center p-12"
        />
      ) : showBrandingImage ? (
        <div className="hidden bg-background lg:block relative w-0 flex-1">
          <img
            className="absolute inset-0 h-full w-full object-cover"
            src={
              currentRouteName === AUTH_SIGNUP_MODULE
                ? signupImage
                : signinImage
            }
            alt=""
          />
        </div>
      ) : null}
      <div className="relative mx-auto flex-1 flex flex-col justify-center py-12 pb-24 px-4 sm:px-6 lg:flex-none lg:px-20 lg:w-1/2 xl:px-24">
        <SplashPanelContext.Provider value={pageOwnsPanel ? panelSlot : null}>
          <Outlet />
        </SplashPanelContext.Provider>
      </div>
    </div>
  );
}
