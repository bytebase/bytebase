import { useLayoutEffect } from "react";

export const LAYER_ROOT_ID = {
  overlay: "bb-react-layer-overlay",
  agent: "bb-react-layer-agent",
  critical: "bb-react-layer-critical",
} as const;

export const LAYER_Z_INDEX = {
  // Legacy naive-ui popovers sit at ~2000 during the Vue->React migration.
  // React overlays mounted inside Vue popovers (e.g. a nested React Popover
  // inside a naive-ui share popover) must sit above their Vue ancestor.
  // 2500 keeps React overlays above naive-ui popovers but below naive-ui
  // modals (3000) and notifications (5000).
  overlay: 2500,
  agent: 2600,
  // Forced re-auth must sit above naive-ui notifications/messages.
  critical: 7000,
} as const;

export type LayerFamily = keyof typeof LAYER_ROOT_ID;

const ORDERED_FAMILIES: LayerFamily[] = ["overlay", "agent", "critical"];
const LAYER_ACCESSIBLE_ATTRIBUTES = [
  "aria-hidden",
  "inert",
  "data-base-ui-inert",
] as const;
const HIGHER_LAYER_FAMILIES: Record<LayerFamily, LayerFamily[]> = {
  overlay: ["agent", "critical"],
  agent: ["critical"],
  critical: [],
};

const getExistingLayerRoot = (family: LayerFamily) =>
  document.getElementById(LAYER_ROOT_ID[family]) as HTMLDivElement | null;

const hasActiveLayerContent = (family: LayerFamily) =>
  (getExistingLayerRoot(family)?.childElementCount ?? 0) > 0;

const ensureRoot = (family: LayerFamily) => {
  const id = LAYER_ROOT_ID[family];
  const existing = document.getElementById(id);
  if (existing) {
    return existing as HTMLDivElement;
  }

  const root = document.createElement("div");
  root.id = id;
  root.dataset.bbLayerFamily = family;
  root.style.position = "relative";
  root.style.zIndex = String(LAYER_Z_INDEX[family]);
  root.style.isolation = "isolate";

  const nextFamily = ORDERED_FAMILIES.slice(
    ORDERED_FAMILIES.indexOf(family) + 1
  ).find((candidate) => document.getElementById(LAYER_ROOT_ID[candidate]));

  if (nextFamily) {
    document.body.insertBefore(
      root,
      document.getElementById(LAYER_ROOT_ID[nextFamily])
    );
  } else {
    document.body.appendChild(root);
  }

  return root as HTMLDivElement;
};

export const getLayerRoot = (family: LayerFamily) => ensureRoot(family);

export const usePreserveHigherLayerAccess = (family: LayerFamily) => {
  useLayoutEffect(() => {
    const preserveableRoots = HIGHER_LAYER_FAMILIES[family]
      .map((higherFamily) => ({
        family: higherFamily,
        root: getExistingLayerRoot(higherFamily),
      }))
      .filter(
        (entry): entry is { family: LayerFamily; root: HTMLDivElement } =>
          entry.root instanceof HTMLDivElement
      );

    if (preserveableRoots.length === 0) {
      return;
    }

    const shouldRevealRoot = (targetFamily: LayerFamily) =>
      HIGHER_LAYER_FAMILIES[targetFamily].every(
        (higherFamily) => !hasActiveLayerContent(higherFamily)
      );

    const revealRoot = (targetFamily: LayerFamily, root: HTMLDivElement) => {
      if (!shouldRevealRoot(targetFamily)) {
        return;
      }

      for (const attribute of LAYER_ACCESSIBLE_ATTRIBUTES) {
        root.removeAttribute(attribute);
      }
    };

    preserveableRoots.forEach(({ family: targetFamily, root }) => {
      revealRoot(targetFamily, root);
    });

    const observer = new MutationObserver((records) => {
      for (const record of records) {
        if (
          record.attributeName &&
          LAYER_ACCESSIBLE_ATTRIBUTES.includes(
            record.attributeName as (typeof LAYER_ACCESSIBLE_ATTRIBUTES)[number]
          ) &&
          record.target instanceof HTMLDivElement
        ) {
          const targetFamily = record.target.dataset.bbLayerFamily as
            | LayerFamily
            | undefined;
          if (targetFamily) {
            revealRoot(targetFamily, record.target);
          }
        }
      }
    });

    for (const { root } of preserveableRoots) {
      observer.observe(root, {
        attributes: true,
        attributeFilter: [...LAYER_ACCESSIBLE_ATTRIBUTES],
      });
    }

    return () => {
      observer.disconnect();
    };
  }, [family]);
};

// Backdrops and popups share one intra-family stack level so nested modal
// layers can rely on portal mount order: child backdrop above parent surface,
// child popup above child backdrop.
export const LAYER_SURFACE_CLASS = "z-10";
export const LAYER_BACKDROP_CLASS = "z-10";
