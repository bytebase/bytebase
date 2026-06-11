import { useEffect } from "react";
import { useTranslation } from "react-i18next";
import { useRouteError } from "react-router-dom";
import { Button } from "@/react/components/ui/button";
import { cn } from "@/react/lib/utils";

/**
 * `errorElement` for uncaught render/loader exceptions, instead of
 * react-router's developer-facing default error screen. Shows generic
 * recovery copy; raw error details (stack, loader error text) can leak
 * bundle paths, component names, or backend messages, so they render only
 * in dev builds — production logs them to the console for diagnostics.
 *
 * Default renders full-screen (the root-route last resort). Pass `inline`
 * when mounting on a layout-seam route so the panel fills the layout's
 * content area and the navigation chrome stays alive.
 */
export function RouteErrorPage({
  inline = false,
}: Readonly<{ inline?: boolean }>) {
  const { t } = useTranslation();
  const error = useRouteError();
  const message = error instanceof Error ? error.message : String(error);
  const stack = error instanceof Error ? error.stack : undefined;
  const showDetails = import.meta.env.DEV;

  // react-router does not log errors that reach a custom errorElement —
  // without this, production errors would be invisible everywhere.
  useEffect(() => {
    console.error("[RouteErrorPage] uncaught route error:", error);
  }, [error]);

  return (
    <div
      className={cn(
        "flex w-full flex-col items-center justify-center gap-y-4 px-4",
        inline ? "h-full py-16" : "h-screen"
      )}
    >
      <h1 className="text-2xl font-semibold text-main">
        {t("error-page.something-went-wrong")}
      </h1>
      <p className="text-sm text-control-light">
        {t("error-page.unexpected-error-description")}
      </p>
      <div className="flex gap-x-2">
        <Button onClick={() => globalThis.location.reload()}>
          {t("common.refresh")}
        </Button>
        <Button
          variant="outline"
          onClick={() => globalThis.location.assign("/")}
        >
          {t("error-page.go-back-home")}
        </Button>
      </div>
      {showDetails && (
        <details className="w-full max-w-2xl text-sm text-control-light">
          <summary className="cursor-pointer select-none">
            {t("error-page.error-details")}
          </summary>
          <pre className="mt-2 max-h-64 overflow-auto whitespace-pre-wrap rounded-sm border border-control-border bg-control-bg p-3 text-xs">
            {stack ?? message}
          </pre>
        </details>
      )}
    </div>
  );
}
