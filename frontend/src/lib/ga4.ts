const GA4_SCRIPT_ID = "bytebase-ga4-tag";

type GAWindow = Window & {
  dataLayer?: unknown[][];
  gtag?: (...args: unknown[]) => void;
};

export function initializeGA4(isSaaSMode: boolean): void {
  const measurementId = import.meta.env.BB_GA4_MEASUREMENT_ID as
    | string
    | undefined;
  if (!isSaaSMode || !measurementId || document.getElementById(GA4_SCRIPT_ID)) {
    return;
  }

  const script = document.createElement("script");
  script.id = GA4_SCRIPT_ID;
  script.async = true;
  script.src = `https://www.googletagmanager.com/gtag/js?id=${measurementId}`;
  document.head.appendChild(script);

  const gaWindow = window as GAWindow;
  gaWindow.dataLayer = gaWindow.dataLayer || [];
  gaWindow.gtag = (...args: unknown[]) => {
    gaWindow.dataLayer?.push(args);
  };
  gaWindow.gtag("js", new Date());
  gaWindow.gtag("config", measurementId, {
    page_location: `${window.location.origin}${window.location.pathname}`,
    page_path: window.location.pathname,
  });
}
