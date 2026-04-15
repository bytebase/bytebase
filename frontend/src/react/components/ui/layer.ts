import { useLayoutEffect } from "react";

export const LAYER_ROOT_ID = {
  overlay: "bb-react-layer-overlay",
  agent: "bb-react-layer-agent",
  critical: "bb-react-layer-critical",
} as const;

export const LAYER_Z_INDEX = {
  overlay: 50,
  agent: 60,
  // Legacy Naive overlays start at 2000 and notifications/messages go higher,
  // so forced re-auth must sit above that during the Vue->React migration.
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
    const higherRoots = HIGHER_LAYER_FAMILIES[family]
      .map((higherFamily) =>
        document.getElementById(LAYER_ROOT_ID[higherFamily])
      )
      .filter((root): root is HTMLDivElement => root instanceof HTMLDivElement);

    if (higherRoots.length === 0) {
      return;
    }

    const revealRoot = (root: HTMLDivElement) => {
      for (const attribute of LAYER_ACCESSIBLE_ATTRIBUTES) {
        root.removeAttribute(attribute);
      }
    };

    higherRoots.forEach(revealRoot);

    const observer = new MutationObserver((records) => {
      for (const record of records) {
        if (
          record.attributeName &&
          LAYER_ACCESSIBLE_ATTRIBUTES.includes(
            record.attributeName as (typeof LAYER_ACCESSIBLE_ATTRIBUTES)[number]
          ) &&
          record.target instanceof HTMLDivElement
        ) {
          revealRoot(record.target);
        }
      }
    });

    for (const root of higherRoots) {
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
