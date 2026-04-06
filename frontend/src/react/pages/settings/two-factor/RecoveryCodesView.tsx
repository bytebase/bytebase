import { Download, Info } from "lucide-react";
import { useCallback } from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@/react/components/ui/button";

interface RecoveryCodesViewProps {
  recoveryCodes: string[];
  onDownload?: () => void;
}

export function RecoveryCodesView({
  recoveryCodes,
  onDownload,
}: RecoveryCodesViewProps) {
  const { t } = useTranslation();

  const downloadRecoveryCodes = useCallback(() => {
    const content = recoveryCodes.join("\n");
    const blob = new Blob([content], { type: "text/plain" });
    const downloadLink = document.createElement("a");
    downloadLink.href = URL.createObjectURL(blob);
    downloadLink.download = "bytebase-recovery-codes.txt";
    document.body.appendChild(downloadLink);
    downloadLink.click();
    URL.revokeObjectURL(downloadLink.href);
    document.body.removeChild(downloadLink);
    onDownload?.();
  }, [recoveryCodes, onDownload]);

  return (
    <div className="w-full flex flex-col gap-y-4 my-8">
      <div className="flex gap-3 rounded-md border border-blue-200 bg-blue-50 p-4">
        <Info className="w-5 h-5 text-blue-600 shrink-0 mt-0.5" />
        <div>
          <p className="text-sm font-medium text-blue-800">
            {t("two-factor.setup-steps.download-recovery-codes.keep-safe.self")}
          </p>
          <div className="text-sm text-blue-700 mt-1">
            <p>{t("two-factor.setup-steps.download-recovery-codes.tips")}</p>
            <p>
              {t(
                "two-factor.setup-steps.download-recovery-codes.keep-safe.description"
              )}
            </p>
          </div>
        </div>
      </div>
      <div className="w-full mx-auto flex flex-col justify-start items-start">
        <ul className="w-full grid grid-cols-2 list-disc list-inside mx-auto gap-4 gap-x-24 p-8 px-12 border rounded-md bg-gray-50">
          {recoveryCodes.map((code) => (
            <li key={code}>
              <code className="ml-2">{code}</code>
            </li>
          ))}
        </ul>
      </div>
      <div className="w-full mx-auto flex flex-row justify-end items-center">
        <Button onClick={downloadRecoveryCodes}>
          <Download className="w-5 h-auto" />
          {t("common.download")}
        </Button>
      </div>
    </div>
  );
}
