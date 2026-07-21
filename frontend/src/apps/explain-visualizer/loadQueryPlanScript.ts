// Dynamically load `html-query-plan` from `/libs/qp.min.js` and resolve
// when `window.QP.showPlan` is callable. The script attaches `QP` as a
// global the first time it loads; subsequent calls short-circuit.

declare global {
  interface Window {
    QP: {
      showPlan: (
        container: Element,
        planXml: string,
        options?: Record<string, unknown>
      ) => void;
    };
  }
}

const SCRIPT_ID = "html-query-plan-script";

export const loadQueryPlanScript = (): Promise<void> => {
  return new Promise((resolve, reject) => {
    if (window.QP) {
      resolve();
      return;
    }
    if (document.getElementById(SCRIPT_ID)) {
      resolve();
      return;
    }
    const script = document.createElement("script");
    script.id = SCRIPT_ID;
    script.src = "/libs/qp.min.js";
    script.onload = () => resolve();
    script.onerror = () =>
      reject(new Error("Failed to load html-query-plan script"));
    document.head.appendChild(script);
  });
};
