// The header advance when it is the current user's turn to review: a "Review"
// button that opens the same Comment / Approve / Reject composer used by the
// review section. The Review section stays the source of detail — this just
// brings the action to the top-of-page lifecycle slot (BYT-9722).
import { ChevronDown } from "lucide-react";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@/react/components/ui/button";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/react/components/ui/popover";
import type { Issue } from "@/types/proto-es/v1/issue_service_pb";
import { ReviewActionPopover } from "../review/ReviewActionPopover";

export function ReviewYourTurnAction({ issue }: { issue: Issue }) {
  const { t } = useTranslation();
  const [open, setOpen] = useState(false);

  return (
    <Popover onOpenChange={setOpen} open={open}>
      <PopoverTrigger render={<Button className="shrink-0 gap-x-1.5" />}>
        {t("plan.review.action")}
        <ChevronDown className="size-4" />
      </PopoverTrigger>
      <PopoverContent align="end" className="px-4 py-4">
        <ReviewActionPopover issue={issue} onClose={() => setOpen(false)} />
      </PopoverContent>
    </Popover>
  );
}
