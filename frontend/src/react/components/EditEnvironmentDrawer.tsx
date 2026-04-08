import { X } from "lucide-react";
import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { EnvironmentLabel } from "@/react/components/EnvironmentLabel";
import { Button } from "@/react/components/ui/button";
import { useEscapeKey } from "@/react/hooks/useEscapeKey";
import { useVueState } from "@/react/hooks/useVueState";
import { cn } from "@/react/lib/utils";
import { useEnvironmentV1Store } from "@/store";

export function EditEnvironmentDrawer({
  open,
  onClose,
  onUpdate,
}: {
  open: boolean;
  onClose: () => void;
  onUpdate: (environment: string) => Promise<void>;
}) {
  const { t } = useTranslation();
  const environmentStore = useEnvironmentV1Store();
  const environments = useVueState(
    () => environmentStore.environmentList ?? []
  );
  const [selected, setSelected] = useState("");
  const [updating, setUpdating] = useState(false);
  useEscapeKey(open, onClose);
  useEffect(() => {
    if (open) {
      setSelected("");
      setUpdating(false);
    }
  }, [open]);

  if (!open) return null;

  return (
    <div className="fixed inset-0 z-50 flex">
      <div className="fixed inset-0 bg-black/50" onClick={onClose} />
      <div className="ml-auto relative bg-white w-[24rem] max-w-[100vw] h-full shadow-lg flex flex-col">
        <div className="flex items-center justify-between px-6 py-4 border-b border-control-border">
          <h2 className="text-lg font-semibold">
            {t("database.edit-environment")}
          </h2>
          <button
            className="p-1 hover:bg-control-bg rounded-xs"
            onClick={onClose}
          >
            <X className="w-4 h-4" />
          </button>
        </div>
        <div className="flex-1 overflow-y-auto p-6">
          <div className="flex flex-col gap-y-1">
            {environments.map((env) => (
              <label
                key={env.name}
                className={cn(
                  "flex items-center gap-x-3 px-3 py-2.5 rounded-sm cursor-pointer border transition-colors",
                  selected === env.name
                    ? "border-accent bg-accent/5"
                    : "border-transparent hover:bg-gray-50"
                )}
              >
                <input
                  type="radio"
                  name="environment"
                  checked={selected === env.name}
                  onChange={() => setSelected(env.name)}
                  className="accent-accent"
                />
                <EnvironmentLabel environment={env} />
              </label>
            ))}
          </div>
        </div>
        <div className="flex justify-end items-center gap-x-2 px-6 py-4 border-t border-control-border">
          <Button variant="ghost" onClick={onClose}>
            {t("common.cancel")}
          </Button>
          <Button
            disabled={!selected || updating}
            onClick={async () => {
              setUpdating(true);
              try {
                await onUpdate(selected);
                onClose();
              } finally {
                setUpdating(false);
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
