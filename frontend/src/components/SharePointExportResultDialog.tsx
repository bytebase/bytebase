import { ErrorInfoSchema } from "@buf/googleapis_googleapis.bufbuild_es/google/rpc/error_details_pb";
import { Code, ConnectError } from "@connectrpc/connect";
import type { TFunction } from "i18next";
import { ChevronRight, ExternalLink, RotateCw } from "lucide-react";
import { useTranslation } from "react-i18next";
import { Alert } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Collapsible,
  CollapsiblePanel,
  CollapsibleTrigger,
} from "@/components/ui/collapsible";
import { Dialog, DialogContent, DialogTitle } from "@/components/ui/dialog";
import { Progress } from "@/components/ui/progress";

// SharePoint URLs arrive percent-encoded; show them as people typed them.
export function formatSharePointUrl(url: string): string {
  try {
    return decodeURI(url);
  } catch {
    return url;
  }
}

// Why a SharePoint export failed, as the Export API reported it.
export type SharePointExportFailure = {
  // The ErrorInfo reason, when the server classified the failure.
  reason?: string;
  // The export folder where some files may already exist.
  folder?: string;
  // The server's message, which can include Microsoft Graph's error code.
  message: string;
};

export type SharePointExportDatabase = {
  // The database resource name, which a retry exports again.
  name: string;
  label: string;
};

export type SharePointExportResult = {
  name: string;
  database: string;
} & (
  | { status: "EXPORTED"; uri: string }
  | { status: "FAILED" | "UNCONFIRMED"; failure: SharePointExportFailure }
  // CANCELED: canceled while this database was exporting, so files may exist.
  // NOT_STARTED: canceled before this database's export was requested.
  | { status: "CANCELED" | "NOT_STARTED" }
);

const SHAREPOINT_ERROR_DOMAIN = "bytebase.com";

// `uri` is the folder created for this export; falls back to the destination
// URL when the response omits it.
export function toExportedSharePointResult(
  database: SharePointExportDatabase,
  uri: string,
  destinationUrl: string
): SharePointExportResult {
  return {
    name: database.name,
    database: database.label,
    status: "EXPORTED",
    uri: uri || destinationUrl,
  };
}

// The Export API reports an upload it could not confirm as UNAVAILABLE. An
// aborted request rejects with CANCELED.
export function toFailedSharePointExport(
  database: SharePointExportDatabase,
  error: unknown
): SharePointExportResult {
  const row = { name: database.name, database: database.label };
  if (!(error instanceof ConnectError)) {
    return { ...row, status: "FAILED", failure: { message: String(error) } };
  }
  if (error.code === Code.Canceled) {
    return { ...row, status: "CANCELED" };
  }
  const info = error
    .findDetails(ErrorInfoSchema)
    .find((detail) => detail.domain === SHAREPOINT_ERROR_DOMAIN);
  return {
    ...row,
    status: error.code === Code.Unavailable ? "UNCONFIRMED" : "FAILED",
    failure: {
      reason: info?.reason,
      folder: info?.metadata.folder || undefined,
      message: error.rawMessage,
    },
  };
}

export function toNotStartedSharePointExport(
  database: SharePointExportDatabase
): SharePointExportResult {
  return {
    name: database.name,
    database: database.label,
    status: "NOT_STARTED",
  };
}

export function isRetryableSharePointExport(
  result: SharePointExportResult
): boolean {
  return result.status === "FAILED" || result.status === "UNCONFIRMED";
}

// Replaces each retried database's row in place, keeping the others.
export function mergeSharePointResults(
  results: SharePointExportResult[],
  retried: SharePointExportResult[]
): SharePointExportResult[] {
  const byName = new Map(retried.map((result) => [result.name, result]));
  return results.map((result) => byName.get(result.name) ?? result);
}

// A retry in progress: which databases it exports again, and for how long.
export type SharePointExportRetry = {
  names: ReadonlySet<string>;
  elapsedSeconds?: number;
};

interface Props {
  results: SharePointExportResult[];
  retry?: SharePointExportRetry;
  // Exports the named databases again with the same options.
  onRetry?: (names: string[]) => void;
  onCancelRetry?: () => void;
  onClose: () => void;
}

export function SharePointExportResultDialog({
  results,
  retry,
  onRetry,
  onCancelRetry,
  onClose,
}: Props) {
  const { t } = useTranslation();
  const allExported = results.every((r) => r.status === "EXPORTED");
  const retryable = results.filter(isRetryableSharePointExport);
  const warnUnconfirmed =
    !retry && retryable.some((r) => r.status === "UNCONFIRMED");

  return (
    <Dialog
      open={results.length > 0}
      onOpenChange={(open) => {
        // A running retry is canceled with its own button, so it cannot be
        // dismissed while its results are still on the way.
        if (!open && !retry) onClose();
      }}
    >
      <DialogContent className="max-w-lg">
        <div className="flex flex-col gap-y-4">
          <DialogTitle>
            {allExported
              ? t("export-data.sharepoint.exported-title")
              : t("export-data.sharepoint.result-title")}
          </DialogTitle>
          <ul className="flex flex-col divide-y divide-block-border rounded-sm border border-block-border">
            {results.map((result) => (
              <li
                key={result.name}
                className="flex flex-col gap-y-1 px-4 py-3 text-sm"
              >
                <div className="flex items-center justify-between gap-x-2">
                  <span className="truncate font-medium">
                    {result.database}
                  </span>
                  {retry?.names.has(result.name) ? (
                    <Badge variant="default">
                      {t("export-data.sharepoint.status.retrying")}
                    </Badge>
                  ) : (
                    <ExportStatusBadge status={result.status} />
                  )}
                </div>
                <ExportResultDetail result={result} />
              </li>
            ))}
          </ul>
          {warnUnconfirmed && (
            <Alert
              variant="warning"
              description={t(
                "export-data.sharepoint.retry-unconfirmed-warning"
              )}
            />
          )}
          {retry ? (
            <div className="flex items-end justify-end gap-x-2">
              <Progress
                className="min-w-0 flex-1"
                value={null}
                label={
                  <>
                    {t("export-data.sharepoint.retrying")}
                    {retry.elapsedSeconds !== undefined && (
                      <>
                        {" "}
                        <span className="tabular-nums">
                          {`${retry.elapsedSeconds}s`}
                        </span>
                      </>
                    )}
                  </>
                }
              />
              <Button appearance="outline" onClick={onCancelRetry}>
                {t("common.cancel")}
              </Button>
            </div>
          ) : (
            <div className="flex justify-end gap-x-2">
              {onRetry && retryable.length > 0 && (
                <Button
                  appearance="outline"
                  onClick={() => onRetry(retryable.map((r) => r.name))}
                >
                  <RotateCw className="size-4" />
                  {t("export-data.sharepoint.retry-failed")}
                </Button>
              )}
              <Button onClick={onClose}>{t("common.close")}</Button>
            </div>
          )}
        </div>
      </DialogContent>
    </Dialog>
  );
}

function ExportResultDetail({ result }: { result: SharePointExportResult }) {
  const { t } = useTranslation();
  switch (result.status) {
    case "EXPORTED":
      return (
        result.uri && (
          <>
            <span className="break-all text-control-light">
              {formatSharePointUrl(result.uri)}
            </span>
            <a
              href={result.uri}
              target="_blank"
              rel="noopener noreferrer"
              className="inline-flex w-fit items-center gap-x-1 text-accent hover:underline"
            >
              {t("export-data.sharepoint.open")}
              <ExternalLink className="size-3" />
            </a>
          </>
        )
      );
    case "CANCELED":
      return (
        <span className="break-words text-control-light">
          {t("export-data.sharepoint.canceled-description")}
        </span>
      );
    case "NOT_STARTED":
      return (
        <span className="break-words text-control-light">
          {t("export-data.sharepoint.not-started-description")}
        </span>
      );
    default:
      return <FailureDetail failure={result.failure} />;
  }
}

function FailureDetail({ failure }: { failure: SharePointExportFailure }) {
  const { t } = useTranslation();
  const explanation = explainSharePointFailure(t, failure.reason);
  // Without a known reason the failure did not come from SharePoint, such as
  // a query error, and the server's message is the explanation.
  if (!explanation) {
    return (
      <span className="break-words text-control-light">{failure.message}</span>
    );
  }
  return (
    <>
      <span className="break-words text-control-light">{explanation}</span>
      {failure.folder && (
        <span className="break-words text-control-light">
          {t("export-data.sharepoint.error.files-may-exist", {
            folder: failure.folder,
          })}
        </span>
      )}
      <Collapsible>
        <CollapsibleTrigger className="group w-fit text-xs text-control-light hover:text-control">
          <ChevronRight className="size-3 transition-transform group-data-[panel-open]:rotate-90" />
          {t("error-page.error-details")}
        </CollapsibleTrigger>
        <CollapsiblePanel className="pt-1">
          <p className="break-words text-xs text-control-light">
            {failure.message}
          </p>
        </CollapsiblePanel>
      </Collapsible>
    </>
  );
}

function explainSharePointFailure(
  t: TFunction,
  reason: string | undefined
): string | undefined {
  switch (reason) {
    case "SHAREPOINT_NOT_CONFIGURED":
      return t("export-data.sharepoint.error.not-configured");
    case "SHAREPOINT_CREDENTIAL_REJECTED":
      return t("export-data.sharepoint.error.credential-rejected");
    case "SHAREPOINT_DESTINATION_NOT_FOUND":
      return t("export-data.sharepoint.error.destination-not-found");
    case "SHAREPOINT_DESTINATION_NOT_FOLDER":
      return t("export-data.sharepoint.error.destination-not-folder");
    case "SHAREPOINT_STORAGE_FULL":
      return t("export-data.sharepoint.error.storage-full");
    case "SHAREPOINT_UPLOAD_UNCONFIRMED":
      return t("export-data.sharepoint.error.upload-unconfirmed");
    case "SHAREPOINT_REQUEST_FAILED":
      return t("export-data.sharepoint.error.request-failed");
    default:
      return undefined;
  }
}

function ExportStatusBadge({
  status,
}: {
  status: SharePointExportResult["status"];
}) {
  const { t } = useTranslation();
  switch (status) {
    case "EXPORTED":
      return (
        <Badge variant="success">
          {t("export-data.sharepoint.status.exported")}
        </Badge>
      );
    case "UNCONFIRMED":
      return (
        <Badge variant="warning">
          {t("export-data.sharepoint.status.unconfirmed")}
        </Badge>
      );
    case "CANCELED":
      return (
        <Badge variant="warning">
          {t("export-data.sharepoint.status.canceled")}
        </Badge>
      );
    case "NOT_STARTED":
      return (
        <Badge variant="default">
          {t("export-data.sharepoint.status.not-started")}
        </Badge>
      );
    case "FAILED":
      return (
        <Badge variant="destructive">
          {t("export-data.sharepoint.status.failed")}
        </Badge>
      );
  }
}
