import { isEqual } from "lodash-es";
import { SlidersHorizontal } from "lucide-react";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { Badge } from "@/react/components/ui/badge";
import { Button } from "@/react/components/ui/button";
import {
  Sheet,
  SheetBody,
  SheetContent,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from "@/react/components/ui/sheet";
import { GhostFlagsForm } from "./GhostFlagsForm";

interface GhostFlagsButtonProps {
  /** Current gh-ost flags parsed from the directive; `{}` = enabled, all defaults. */
  value: Record<string, string>;
  onChange: (next: Record<string, string>) => void;
  disabled?: boolean;
}

/**
 * "Configure" entry point for online-migration (gh-ost) parameters. Opens a Sheet
 * that edits the flags carried by the `-- gh-ost = {...}` directive; only
 * non-default values are written, matching the backend
 * (`backend/component/ghost/config.go`).
 *
 * Edits are buffered in a local draft and the directive is written (via
 * `onChange`) only when the user clicks Save — persisting on every keystroke would
 * refetch and re-render the whole plan, flickering the page. Cancel/Escape/scrim
 * discard the draft.
 */
export function GhostFlagsButton({
  value,
  onChange,
  disabled,
}: GhostFlagsButtonProps) {
  const { t } = useTranslation();
  const [open, setOpen] = useState(false);
  const [draft, setDraft] = useState(value);
  const overrides = Object.keys(value).length;

  const openSheet = () => {
    // Seed the draft from the latest persisted value each time we open.
    setDraft(value);
    setOpen(true);
  };
  const save = () => {
    onChange(draft);
    setOpen(false);
  };

  return (
    <>
      <Button
        variant="outline"
        size="sm"
        disabled={disabled}
        onClick={openSheet}
      >
        <SlidersHorizontal className="size-3.5" />
        {t("plan.ghost.configure")}
        {overrides > 0 && (
          <Badge variant="secondary" className="px-1.5 py-0 text-xs">
            {overrides}
          </Badge>
        )}
      </Button>
      <Sheet open={open} onOpenChange={setOpen}>
        <SheetContent width="panel">
          <SheetHeader>
            <SheetTitle>{t("plan.ghost.parameters")}</SheetTitle>
          </SheetHeader>
          <SheetBody>
            <GhostFlagsForm value={draft} onChange={setDraft} />
          </SheetBody>
          <SheetFooter className="justify-between">
            <Button
              variant="ghost"
              size="sm"
              disabled={Object.keys(draft).length === 0}
              onClick={() => setDraft({})}
            >
              {t("plan.ghost.reset-to-defaults")}
            </Button>
            <div className="flex gap-x-2">
              <Button
                variant="outline"
                size="sm"
                onClick={() => setOpen(false)}
              >
                {t("common.cancel")}
              </Button>
              <Button size="sm" disabled={isEqual(draft, value)} onClick={save}>
                {t("common.save")}
              </Button>
            </div>
          </SheetFooter>
        </SheetContent>
      </Sheet>
    </>
  );
}
