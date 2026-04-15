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

// Backdrops and popups share one intra-family stack level so nested modal
// layers can rely on portal mount order: child backdrop above parent surface,
// child popup above child backdrop.
export const LAYER_SURFACE_CLASS = "z-10";
export const LAYER_BACKDROP_CLASS = "z-10";
