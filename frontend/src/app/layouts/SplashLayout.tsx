import { createContext, type ReactNode, useContext, useState } from "react";
import { createPortal } from "react-dom";
import { Outlet, useMatches } from "react-router";
import { AUTH_SETUP_MODULE } from "@/app/router/handles";
import { useMediaQuery } from "@/hooks/useMediaQuery";
import { cn } from "@/lib/utils";

// The left column's element on a route that hands that column to its page.
export const SplashPanelContext = createContext<HTMLElement | null>(null);

// Renders `children` in the layout's left column. On a route that has no
// left column, it renders nothing.
export function SplashPanel({ children }: { readonly children: ReactNode }) {
  const slot = useContext(SplashPanelContext);
  return slot ? createPortal(children, slot) : null;
}

// Tailwind's `lg` breakpoint, from which the setup route's left column shows.
const LEFT_COLUMN_QUERY = "(min-width: 1024px)";

// Auth chrome: the routed auth page is centered in one column. The setup
// route alone splits in two from lg, with the wizard's own panel on the left
// and the page in the right half; below lg that column is hidden and the page
// gets no slot, so the panel is not mounted unseen.
//
// The column owns the gutter and centers the page, which fills it up to its
// own cap (`w-full max-w-*`) and adds no width fraction or side padding of its
// own, which would compound with the column's. The column is also the
// containing block for the page's absolutely positioned children: `pb-24`
// keeps the sign-in and sign-up `AuthFooter` (`absolute bottom-0`) clear of
// the form, which therefore sits 24px above true center on every route.
export function SplashLayout() {
  const matches = useMatches();
  const currentRouteName = (
    matches.at(-1)?.handle as { name?: string } | undefined
  )?.name;
  const pageOwnsPanel = currentRouteName === AUTH_SETUP_MODULE;
  const leftColumnShown = useMediaQuery(LEFT_COLUMN_QUERY);
  const [panelSlot, setPanelSlot] = useState<HTMLElement | null>(null);

  return (
    <div className="min-h-screen flex">
      {pageOwnsPanel && (
        <div
          ref={setPanelSlot}
          className="hidden bg-control-bg lg:flex w-0 flex-1 items-center justify-center p-12"
        />
      )}
      <div
        className={cn(
          "relative flex-1 flex flex-col items-center justify-center py-12 pb-24 px-4 sm:px-6",
          pageOwnsPanel && "lg:flex-none lg:px-20 lg:w-1/2 xl:px-24"
        )}
      >
        <SplashPanelContext.Provider
          value={pageOwnsPanel && leftColumnShown ? panelSlot : null}
        >
          <Outlet />
        </SplashPanelContext.Provider>
      </div>
    </div>
  );
}
