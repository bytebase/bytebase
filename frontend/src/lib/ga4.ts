const GA4_MEASUREMENT_ID = "G-4BZ4JH7449";
const GA4_SCRIPT_ID = "bytebase-ga4-tag";

type GAWindow = Window & {
  dataLayer?: unknown[][];
  gtag?: (...args: unknown[]) => void;
};

export function initializeGA4(isSaaSMode: boolean): void {
  if (!isSaaSMode || document.getElementById(GA4_SCRIPT_ID)) {
    return;
  }

  const script = document.createElement("script");
  script.id = GA4_SCRIPT_ID;
  script.async = true;
  script.src = `https://www.googletagmanager.com/gtag/js?id=${GA4_MEASUREMENT_ID}`;
  document.head.appendChild(script);

  const gaWindow = window as GAWindow;
  gaWindow.dataLayer = gaWindow.dataLayer || [];
  gaWindow.gtag = (...args: unknown[]) => {
    gaWindow.dataLayer?.push(args);
  };
  gaWindow.gtag("js", new Date());
  gaWindow.gtag("config", GA4_MEASUREMENT_ID);
}
