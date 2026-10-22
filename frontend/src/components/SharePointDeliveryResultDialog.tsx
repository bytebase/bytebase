import { Code, ConnectError } from "@connectrpc/connect";
import { ExternalLink } from "lucide-react";
import { useTranslation } from "react-i18next";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogTitle,
} from "@/components/ui/dialog";
import type { SharePointDelivery } from "@/types/proto-es/v1/sql_service_pb";

export type SharePointDeliveryResult = {
  database: string;
} & (
  | { status: "DELIVERED"; folderName: string; webUrl: string }
  | { status: "FAILED" | "UNCONFIRMED"; reason: string }
);

// Falls back to the destination URL when the response omits the delivery.
export function toDeliveredSharePointResult(
  database: string,
  delivery: SharePointDelivery | undefined,
  destinationWebUrl: string
): SharePointDeliveryResult {
  return {
    database,
    status: "DELIVERED",
    folderName: delivery?.folderName ?? "",
    webUrl: delivery?.webUrl || destinationWebUrl,
  };
}

// The Export API reports a delivery it could not confirm as UNAVAILABLE.
export function toFailedSharePointDelivery(
  database: string,
  error: unknown
): SharePointDeliveryResult {
  if (error instanceof ConnectError) {
    return {
      database,
      status: error.code === Code.Unavailable ? "UNCONFIRMED" : "FAILED",
      reason: error.rawMessage,
    };
  }
  return { database, status: "FAILED", reason: String(error) };
}

interface Props {
  results: SharePointDeliveryResult[];
  onClose: () => void;
}

export function SharePointDeliveryResultDialog({ results, onClose }: Props) {
  const { t } = useTranslation();
  const allDelivered = results.every((r) => r.status === "DELIVERED");

  return (
    <Dialog
      open={results.length > 0}
      onOpenChange={(open) => {
        if (!open) onClose();
      }}
    >
      <DialogContent className="max-w-lg">
        <div className="flex flex-col gap-y-4">
          <div className="flex flex-col gap-y-1">
            <DialogTitle>
              {allDelivered
                ? t("export-data.sharepoint.delivered-title")
                : t("export-data.sharepoint.result-title")}
            </DialogTitle>
            <DialogDescription>
              {t("export-data.sharepoint.access-note")}
            </DialogDescription>
          </div>
          <ul className="flex flex-col divide-y divide-block-border rounded-sm border border-block-border">
            {results.map((result) => (
              <li
                key={result.database}
                className="flex flex-col gap-y-1 px-4 py-3 text-sm"
              >
                <div className="flex items-center justify-between gap-x-2">
                  <span className="truncate font-medium">
                    {result.database}
                  </span>
                  <DeliveryStatusBadge status={result.status} />
                </div>
                {result.status === "DELIVERED" ? (
                  <>
                    {result.folderName && (
                      <span className="break-all text-control-light">
                        {result.folderName}
                      </span>
                    )}
                    {result.webUrl && (
                      <a
                        href={result.webUrl}
                        target="_blank"
                        rel="noopener noreferrer"
                        className="inline-flex w-fit items-center gap-x-1 text-accent hover:underline"
                      >
                        {t("export-data.sharepoint.open")}
                        <ExternalLink className="size-3" />
                      </a>
                    )}
                  </>
                ) : (
                  <span className="break-words text-control-light">
                    {result.reason}
                  </span>
                )}
              </li>
            ))}
          </ul>
          <div className="flex justify-end">
            <Button onClick={onClose}>{t("common.close")}</Button>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  );
}

function DeliveryStatusBadge({
  status,
}: {
  status: SharePointDeliveryResult["status"];
}) {
  const { t } = useTranslation();
  switch (status) {
    case "DELIVERED":
      return (
        <Badge variant="success">
          {t("export-data.sharepoint.status.delivered")}
        </Badge>
      );
    case "UNCONFIRMED":
      return (
        <Badge variant="warning">
          {t("export-data.sharepoint.status.unconfirmed")}
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
