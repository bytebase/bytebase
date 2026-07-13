import "./productIntro.css";

import { useAppStore } from "@/react/stores/app";

export const PRODUCT_INTRO_QUERY_KEY = "intro";
export const CONNECT_DATABASE_PRODUCT_INTRO = "connect-database";

export type ProductIntroOptions = {
  id: string;
  title: string;
  description: string;
  closeLabel: string;
};

const ACTIVE_TARGET_CLASS = "bb-product-intro-active";
const ROOT_CLASS = "bb-product-intro";
const STAGE_CLASS = "bb-product-intro-stage";
const POPOVER_CLASS = "bb-product-intro-popover";
const ARROW_CLASS = "bb-product-intro-arrow";
const REFRESH_DELAY_MS = 100;
const STAGE_PADDING = 6;
const POPOVER_GAP = 14;
const POPOVER_WIDTH = 320;
const POPOVER_HEIGHT = 128;
const VIEWPORT_PADDING = 16;

const getTargetSelector = (id: string) => `[data-product-intro-target="${id}"]`;

const isVisibleRect = (rect: DOMRect) => rect.width > 0 && rect.height > 0;

const resolveTargetElement = (selector: string): HTMLElement | undefined => {
  for (const element of document.querySelectorAll(selector)) {
    if (!(element instanceof HTMLElement)) {
      continue;
    }
    if (!element.isConnected) {
      continue;
    }
    if (isVisibleRect(element.getBoundingClientRect())) {
      return element;
    }
  }
  return undefined;
};

const findElement = async (
  selector: string
): Promise<HTMLElement | undefined> => {
  for (let i = 0; i < 10; i++) {
    const element = resolveTargetElement(selector);
    if (element) {
      return element;
    }
    await new Promise<void>((resolve) => window.setTimeout(resolve, 100));
  }
  return undefined;
};

const clamp = (value: number, min: number, max: number) =>
  Math.min(Math.max(value, min), max);

const removeActiveTargetClass = () => {
  for (const element of document.querySelectorAll(`.${ACTIVE_TARGET_CLASS}`)) {
    element.classList.remove(ACTIVE_TARGET_CLASS);
  }
};

const markActiveTarget = (element: HTMLElement) => {
  removeActiveTargetClass();
  element.classList.add(ACTIVE_TARGET_CLASS);
};

const createIntroElements = ({
  title,
  description,
  closeLabel,
}: Pick<ProductIntroOptions, "title" | "description" | "closeLabel">) => {
  const root = document.createElement("div");
  root.className = ROOT_CLASS;

  const stage = document.createElement("div");
  stage.className = STAGE_CLASS;

  const popover = document.createElement("div");
  popover.className = POPOVER_CLASS;
  popover.setAttribute("role", "dialog");
  popover.setAttribute("aria-modal", "false");

  const arrow = document.createElement("div");
  arrow.className = ARROW_CLASS;

  const titleElement = document.createElement("div");
  titleElement.className = "bb-product-intro-title";
  titleElement.textContent = title;

  const descriptionElement = document.createElement("div");
  descriptionElement.className = "bb-product-intro-description";
  descriptionElement.textContent = description;

  const closeButton = document.createElement("button");
  closeButton.className = "bb-product-intro-close";
  closeButton.type = "button";
  closeButton.setAttribute("aria-label", closeLabel);
  closeButton.textContent = "×";

  popover.append(arrow, closeButton, titleElement, descriptionElement);
  root.append(stage, popover);

  return { root, stage, popover, arrow, closeButton };
};

export const showProductIntroOnce = async ({
  id,
  title,
  description,
  closeLabel,
}: ProductIntroOptions): Promise<boolean> => {
  const target = getTargetSelector(id);
  const store = useAppStore.getState();

  if (store.getIntroStateByKey(id)) {
    return false;
  }

  const element = await findElement(target);
  if (!element) {
    return false;
  }

  let activeElement = element;
  let destroyed = false;
  let refreshTimer: number | undefined;
  const { root, stage, popover, arrow, closeButton } = createIntroElements({
    title,
    description,
    closeLabel,
  });

  const render = () => {
    const rect = activeElement.getBoundingClientRect();
    stage.style.top = `${rect.top - STAGE_PADDING}px`;
    stage.style.left = `${rect.left - STAGE_PADDING}px`;
    stage.style.width = `${rect.width + STAGE_PADDING * 2}px`;
    stage.style.height = `${rect.height + STAGE_PADDING * 2}px`;

    const maxLeft = window.innerWidth - POPOVER_WIDTH - VIEWPORT_PADDING;
    const left = clamp(
      rect.left + rect.width / 2 - POPOVER_WIDTH / 2,
      VIEWPORT_PADDING,
      Math.max(VIEWPORT_PADDING, maxLeft)
    );
    const canShowBelow =
      rect.bottom + POPOVER_GAP + POPOVER_HEIGHT <= window.innerHeight;
    const top = canShowBelow
      ? rect.bottom + POPOVER_GAP
      : Math.max(VIEWPORT_PADDING, rect.top - POPOVER_GAP - POPOVER_HEIGHT);

    popover.style.left = `${left}px`;
    popover.style.top = `${top}px`;
    arrow.classList.toggle("bb-product-intro-arrow-top", canShowBelow);
    arrow.classList.toggle("bb-product-intro-arrow-bottom", !canShowBelow);
    arrow.style.left = `${clamp(
      rect.left + rect.width / 2 - left - 6,
      12,
      POPOVER_WIDTH - 24
    )}px`;
  };

  const cleanup = () => {
    if (destroyed) {
      return;
    }
    destroyed = true;
    if (refreshTimer !== undefined) {
      window.clearTimeout(refreshTimer);
      refreshTimer = undefined;
    }
    closeButton.removeEventListener("click", handleCloseClick);
    document.removeEventListener("click", handleDocumentClick, true);
    document.removeEventListener("keydown", handleDocumentKeydown, true);
    window.removeEventListener("resize", scheduleRefreshTarget);
    window.removeEventListener("scroll", scheduleRefreshTarget, true);
    root.remove();
    removeActiveTargetClass();
    store.saveIntroStateByKey({ key: id, newState: true });
  };

  const refreshTarget = () => {
    refreshTimer = undefined;
    const currentElement = resolveTargetElement(target);
    if (!currentElement) {
      cleanup();
      return;
    }
    if (currentElement !== activeElement) {
      activeElement.classList.remove(ACTIVE_TARGET_CLASS);
      activeElement = currentElement;
      markActiveTarget(activeElement);
    }
    render();
  };

  const scheduleRefreshTarget = () => {
    if (refreshTimer !== undefined) {
      window.clearTimeout(refreshTimer);
    }
    refreshTimer = window.setTimeout(refreshTarget, REFRESH_DELAY_MS);
  };

  function handleCloseClick(event: MouseEvent) {
    event.preventDefault();
    event.stopPropagation();
    cleanup();
  }

  function handleDocumentClick(event: MouseEvent) {
    const currentElement = resolveTargetElement(target);
    if (!currentElement) {
      cleanup();
      return;
    }
    if (event.target instanceof Node && currentElement.contains(event.target)) {
      cleanup();
    }
  }

  function handleDocumentKeydown(event: KeyboardEvent) {
    if (event.key === "Escape") {
      cleanup();
    }
  }

  markActiveTarget(activeElement);
  document.body.appendChild(root);
  render();
  closeButton.addEventListener("click", handleCloseClick);
  document.addEventListener("click", handleDocumentClick, true);
  document.addEventListener("keydown", handleDocumentKeydown, true);
  window.addEventListener("resize", scheduleRefreshTarget);
  window.addEventListener("scroll", scheduleRefreshTarget, true);
  return true;
};
