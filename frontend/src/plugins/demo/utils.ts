import { assign, isNumber, isNaN } from "lodash-es";

// getStylePropertyValue returns the value of the given style property of the given element.
export const getStylePropertyValue = (
  element: HTMLElement,
  propertyName: string
) => {
  const propertyValue = window
    .getComputedStyle(element)
    .getPropertyValue(propertyName);

  return propertyValue;
};

// isElementFixed returns true if the element is fixed.
export const isElementFixed = (element: HTMLElement): boolean => {
  const parentNode = element.parentNode;

  if (!parentNode || parentNode.nodeName === "HTML") {
    return false;
  }

  if (getStylePropertyValue(element, "position") === "fixed") {
    return true;
  }

  return isElementFixed(parentNode as HTMLElement);
};

// getElementBounding returns the bounding rectangle and position of the element.
export const getElementBounding = (
  element: HTMLElement,
  relativeEl?: HTMLElement
) => {
  const scrollTop =
    window.pageYOffset ||
    document.documentElement.scrollTop ||
    document.body.scrollTop;
  const scrollLeft =
    window.pageXOffset ||
    document.documentElement.scrollLeft ||
    document.body.scrollLeft;

  relativeEl = relativeEl || document.body;

  const elementRect = element.getBoundingClientRect();
  const relativeElRect = relativeEl.getBoundingClientRect();
  const relativeElPosition = getStylePropertyValue(relativeEl, "position");

  const bounding = {
    width: elementRect.width,
    height: elementRect.height,
  };

  if (
    (relativeEl.tagName !== "BODY" && relativeElPosition === "relative") ||
    relativeElPosition === "sticky"
  ) {
    return assign(bounding, {
      top: elementRect.top - relativeElRect.top,
      left: elementRect.left - relativeElRect.left,
    });
  }

  if (isElementFixed(element)) {
    return assign(bounding, {
      top: elementRect.top,
      left: elementRect.left,
    });
  }

  return assign(bounding, {
    top: elementRect.top + scrollTop,
    left: elementRect.left + scrollLeft,
  });
};

// getElementMaxZIndex returns the max z-index of the element and its parents.
export const getElementMaxZIndex = (element: HTMLElement): number => {
  const zIndex = Number(getStylePropertyValue(element, "z-index"));

  if (element.parentElement && element.parentElement !== document.body) {
    if (isNumber(zIndex) && !isNaN(zIndex)) {
      return Math.max(zIndex, getElementMaxZIndex(element.parentElement));
    }
    return getElementMaxZIndex(element.parentElement);
  }

  return 0;
};

const getTargetElementWithSelector = (selector: string) => {
  let targetElement = null;
  try {
    targetElement = document.body.querySelector(selector) as HTMLElement;
  } catch (error) {
    // do nth
  }
  return targetElement;
};

// waitForTargetElement will wait for the target element to be available in the DOM.
export const waitForTargetElement = (
  selector: string
): Promise<HTMLElement> => {
  return new Promise((resolve) => {
    let targetElement = getTargetElementWithSelector(selector);
    if (targetElement) {
      return resolve(targetElement);
    }

    const observer = new MutationObserver(() => {
      targetElement = getTargetElementWithSelector(selector);
      if (targetElement) {
        observer.disconnect();
        return resolve(targetElement);
      }
    });

    observer.observe(document.body, {
      childList: true,
      subtree: true,
    });
  });
};

export const waitBodyLoaded = () => {
  return new Promise((resolve) => {
    let t = setTimeout(() => {
      resolve(undefined);
    }, 1000);

    const observer = new MutationObserver(() => {
      clearTimeout(t);
      t = setTimeout(() => {
        resolve(undefined);
      }, 1000);
    });

    observer.observe(document.body, {
      childList: true,
      subtree: true,
    });
  });
};

// checkUrlMatched is used to check if the given url's pathname is matched with the location pathname.
export const checkUrlPathnameMatched = (url: string) => {
  const urlObject = new URL(url);
  return urlObject.pathname === window.location.pathname;
};

export const checkUrlMatched = (url: string) => {
  const regex = new RegExp(url);
  return regex.test(window.location.href);
};

export const isNullOrUndefined = (value: any) => {
  return value === null || value === undefined;
};

export const getScrollParent = (
  element: HTMLElement | null | undefined
): HTMLElement => {
  if (!element) {
    return document.body;
  }

  if (element.scrollHeight > element.clientHeight) {
    return element;
  } else {
    return getScrollParent(element.parentElement);
  }
};
