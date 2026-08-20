import { Play } from "lucide-react";
import type { ComponentProps } from "react";
import { Button } from "@/components/ui/button";
import { Tooltip } from "@/components/ui/tooltip";
import { cn } from "@/lib/utils";
import { useSQLEditorEditorState } from "@/modules/sql-editor/store/editor";
import { QueryContextSettingPopover } from "./QueryContextSettingPopover";

type Props = {
  readonly disabled?: boolean;
  readonly settingsDisabled?: boolean;
  readonly size?: "sm" | "md";
  readonly type?: ComponentProps<typeof Button>["type"];
  readonly onClick?: ComponentProps<typeof Button>["onClick"];
};

export function RunQueryButton({
  disabled = false,
  settingsDisabled = disabled,
  size = "sm",
  type = "button",
  onClick,
}: Props) {
  const resultRowsLimit = useSQLEditorEditorState((s) => s.resultRowsLimit);

  return (
    <div className="inline-flex">
      <Tooltip content="" side="bottom">
        <Button
          type={type}
          variant="default"
          size={size}
          className={cn(
            "px-1.5 gap-1 rounded-r-none text-sm",
            size === "sm" && "h-7"
          )}
          disabled={disabled}
          onClick={onClick}
        >
          <Play className="size-4 fill-current" />
          <span className="inline-flex items-center">
            (limit&nbsp;{resultRowsLimit})
          </span>
        </Button>
      </Tooltip>
      <QueryContextSettingPopover disabled={settingsDisabled} size={size} />
    </div>
  );
}
