import type { ComponentType } from "react";

// Adapter for react-router's `lazy` route field.
//
// Page components were authored for the vue layer, which injected route
// params as props (e.g. `{ projectId: string }`). react-router renders a
// route's `Component` with no props — params are read via `useParams()`. The
// per-page migration to `useParams()` happens in a later phase; until then the
// resolved component is widened to `ComponentType` (no required props) so the
// route table type-checks. This is the single, intentional seam where that
// widening lives, instead of scattering casts across every route.
export const lazyPage =
  <T extends Record<string, unknown>>(
    loader: () => Promise<T>,
    pick: (m: T) => unknown
  ) =>
  async (): Promise<{ Component: ComponentType }> => {
    const m = await loader();
    return { Component: pick(m) as ComponentType };
  };
