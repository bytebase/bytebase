import { Plus, X } from "lucide-react";
import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { Alert } from "@/react/components/ui/alert";
import { Button } from "@/react/components/ui/button";
import { Input } from "@/react/components/ui/input";
import { useEscapeKey } from "@/react/hooks/useEscapeKey";
import type { Database } from "@/types/proto-es/v1/database_service_pb";

export function LabelEditorDrawer({
  open,
  databases,
  onClose,
  onApply,
}: {
  open: boolean;
  databases: Database[];
  onClose: () => void;
  onApply: (labelsList: { [key: string]: string }[]) => Promise<void>;
}) {
  const { t } = useTranslation();
  const [labelsList, setLabelsList] = useState<{ [key: string]: string }[]>([]);
  const [applying, setApplying] = useState(false);
  const [newKey, setNewKey] = useState("");
  const [newValue, setNewValue] = useState("");

  useEscapeKey(open, onClose);
  useEffect(() => {
    if (open) {
      setLabelsList(databases.map((db) => ({ ...db.labels })));
      setApplying(false);
      setNewKey("");
      setNewValue("");
    }
  }, [open, databases]);

  if (!open) return null;

  const addLabelToAll = () => {
    if (!newKey.trim()) return;
    setLabelsList((prev) =>
      prev.map((labels) => ({ ...labels, [newKey.trim()]: newValue.trim() }))
    );
    setNewKey("");
    setNewValue("");
  };

  const removeLabel = (key: string) => {
    setLabelsList((prev) =>
      prev.map((labels) => {
        const next = { ...labels };
        delete next[key];
        return next;
      })
    );
  };

  const allKeys = Array.from(
    new Set(labelsList.flatMap((labels) => Object.keys(labels)))
  ).sort();

  const getDisplayValue = (key: string): { value: string; mixed: boolean } => {
    const values = new Set(
      labelsList.map((labels) => labels[key] ?? "").filter(Boolean)
    );
    if (values.size === 0) return { value: "", mixed: false };
    if (values.size === 1) return { value: [...values][0], mixed: false };
    return { value: t("database.mixed-label-values"), mixed: true };
  };

  const hasMixedValues = allKeys.some((key) => getDisplayValue(key).mixed);

  return (
    <div className="fixed inset-0 z-50 flex">
      <div className="fixed inset-0 bg-black/50" onClick={onClose} />
      <div className="ml-auto relative bg-white w-[28rem] max-w-[100vw] h-full shadow-lg flex flex-col">
        <div className="flex items-center justify-between px-6 py-4 border-b border-control-border">
          <h2 className="text-lg font-semibold">{t("database.edit-labels")}</h2>
          <button
            className="p-1 hover:bg-control-bg rounded-xs"
            onClick={onClose}
          >
            <X className="w-4 h-4" />
          </button>
        </div>
        <div className="flex-1 overflow-y-auto p-6">
          {hasMixedValues && (
            <Alert variant="warning" className="mb-4">
              {t("database.mixed-label-values-warning")}
            </Alert>
          )}
          <div className="flex items-center gap-x-2 mb-4">
            <Input
              placeholder={t("common.key")}
              value={newKey}
              onChange={(e) => setNewKey(e.target.value)}
              className="flex-1"
            />
            <span className="text-control-placeholder">:</span>
            <Input
              placeholder={t("common.value")}
              value={newValue}
              onChange={(e) => setNewValue(e.target.value)}
              className="flex-1"
              onKeyDown={(e) => {
                if (e.key === "Enter") addLabelToAll();
              }}
            />
            <Button size="sm" onClick={addLabelToAll} disabled={!newKey.trim()}>
              <Plus className="h-4 w-4" />
            </Button>
          </div>
          {allKeys.length > 0 ? (
            <div className="flex flex-col gap-y-2">
              {allKeys.map((key) => {
                const { value, mixed } = getDisplayValue(key);
                return (
                  <div
                    key={key}
                    className="flex items-center justify-between rounded-xs bg-gray-50 px-3 py-2"
                  >
                    <span className="text-sm">
                      {key}:
                      {mixed ? (
                        <span className="italic text-control-placeholder ml-1">
                          {value}
                        </span>
                      ) : (
                        value
                      )}
                    </span>
                    <button
                      className="p-0.5 hover:bg-gray-200 rounded-xs"
                      onClick={() => removeLabel(key)}
                    >
                      <X className="w-3 h-3" />
                    </button>
                  </div>
                );
              })}
            </div>
          ) : (
            <p className="text-sm text-control-placeholder">
              {t("common.no-data")}
            </p>
          )}
        </div>
        <div className="flex justify-end items-center gap-x-2 px-6 py-4 border-t border-control-border">
          <Button variant="ghost" onClick={onClose}>
            {t("common.cancel")}
          </Button>
          <Button
            disabled={applying}
            onClick={async () => {
              setApplying(true);
              try {
                await onApply(labelsList);
                onClose();
              } finally {
                setApplying(false);
              }
            }}
          >
            {t("common.update")}
          </Button>
        </div>
      </div>
    </div>
  );
}
