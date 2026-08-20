import { ChevronDownIcon, CopyIcon } from "lucide-react";
import { useTranslation } from "react-i18next";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { useSelectionContext } from "./context";
import { formatAsCSV, formatAsSQL, formatAsText } from "./copy-formats";

export function CopyAllButton() {
  const { t } = useTranslation();
  const { copy, canCopyAsInsert } = useSelectionContext();

  return (
    <div className="flex items-center">
      <Button
        size="sm"
        appearance="outline"
        className="h-7 px-2 rounded-r-none border-r-0 text-control border-control-border hover:bg-control-bg-hover"
        onClick={() => copy("all", formatAsText)}
      >
        <CopyIcon className="size-4" />
        {t("common.copy-all")}
      </Button>
      <DropdownMenu>
        <DropdownMenuTrigger
          openOnHover
          delay={100}
          render={
            <Button
              size="sm"
              appearance="outline"
              aria-label={t("common.copy")}
              className="h-7 w-6 px-0 rounded-l-none text-control border-control-border hover:bg-control-bg-hover"
            >
              <ChevronDownIcon className="size-4" />
            </Button>
          }
        />
        <DropdownMenuContent align="end" className="min-w-0">
          <DropdownMenuItem
            onClick={() => copy("all", formatAsCSV)}
            className="px-2 py-1 text-xs gap-x-1.5"
          >
            <CopyIcon className="size-3" />
            {t("sql-editor.copy-all-rows-as-csv")}
          </DropdownMenuItem>
          {canCopyAsInsert && (
            <DropdownMenuItem
              onClick={() => copy("all", formatAsSQL)}
              className="px-2 py-1 text-xs gap-x-1.5"
            >
              <CopyIcon className="size-3" />
              {t("sql-editor.copy-all-rows-as-sql")}
            </DropdownMenuItem>
          )}
        </DropdownMenuContent>
      </DropdownMenu>
    </div>
  );
}
