import { X } from "lucide-react";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { cn } from "@/lib/utils";
import {
  addInviteAddresses,
  describeInviteAddresses,
  endsAnAddress,
} from "./inviteEmails";

export function InviteEmailsInput({
  id,
  addresses,
  ownerEmail,
  describedBy,
  disabled = false,
  onChange,
}: {
  readonly id: string;
  readonly addresses: readonly string[];
  readonly ownerEmail: string;
  readonly describedBy: string;
  readonly disabled?: boolean;
  readonly onChange: (addresses: string[]) => void;
}) {
  const { t } = useTranslation();
  const [pending, setPending] = useState("");
  const entries = describeInviteAddresses(addresses, ownerEmail);

  const add = (text: string) => {
    onChange(addInviteAddresses(addresses, text));
    setPending("");
  };
  const commit = () => {
    if (pending.trim()) add(pending);
  };

  return (
    <div className="flex min-h-9 w-full min-w-0 flex-wrap items-center gap-1 rounded-xs border border-control-border bg-background px-3 py-1 text-sm leading-5 focus-within:border-accent">
      {entries.map((entry) => {
        const problem =
          entry.problem === "invalid"
            ? t("workspace-setup.team.address-invalid")
            : entry.problem === "own"
              ? t("workspace-setup.team.address-own")
              : undefined;
        return (
          <span
            key={entry.address}
            title={problem}
            className="inline-flex min-w-0 max-w-full items-center gap-1 rounded-xs bg-control-bg px-1.5 py-0.5 text-xs"
          >
            <span
              className={cn(
                "min-w-0 truncate",
                problem && "text-control-light line-through"
              )}
            >
              {entry.address}
            </span>
            {problem && <span className="sr-only">{problem}</span>}
            <Button
              type="button"
              appearance="secondary"
              size="xs"
              className="text-control-placeholder hover:text-control-light"
              aria-label={t("workspace-setup.team.remove-address", {
                email: entry.address,
              })}
              disabled={disabled}
              onClick={() =>
                onChange(
                  addresses.filter((address) => address !== entry.address)
                )
              }
            >
              <X className="size-3" />
            </Button>
          </span>
        );
      })}
      <Input
        id={id}
        size="xs"
        type="text"
        inputMode="email"
        autoComplete="off"
        value={pending}
        disabled={disabled}
        aria-describedby={describedBy}
        placeholder={
          addresses.length > 0
            ? t("workspace-setup.team.address-placeholder-more")
            : t("workspace-setup.team.address-placeholder")
        }
        className="h-5 min-w-40 flex-1 border-0 bg-transparent p-0 text-sm leading-5 shadow-none outline-none placeholder:text-control-placeholder focus:border-0 focus:ring-0"
        onChange={(event) => {
          const value = event.target.value;
          if (endsAnAddress(value)) add(value);
          else setPending(value);
        }}
        // A single-line input drops the line breaks of a pasted column, which
        // would join its addresses, so a pasted list is read from the clipboard.
        onPaste={(event) => {
          const text = event.clipboardData.getData("text");
          if (!endsAnAddress(text.trim())) return;
          event.preventDefault();
          // The list takes the selection's place, and the text typed around it
          // stays a recipient of its own.
          const { selectionStart, selectionEnd } = event.currentTarget;
          add(
            [
              pending.slice(0, selectionStart ?? pending.length),
              text,
              pending.slice(selectionEnd ?? pending.length),
            ].join("\n")
          );
        }}
        onKeyDown={(event) => {
          // While an input method composes text, its keys are its own.
          if (event.nativeEvent.isComposing) return;
          if (event.key === "Enter") {
            // Enter adds the address; it must not submit the wizard.
            event.preventDefault();
            commit();
          } else if (
            event.key === "Backspace" &&
            !pending &&
            addresses.length > 0
          ) {
            onChange(addresses.slice(0, -1));
          }
        }}
        onBlur={commit}
      />
    </div>
  );
}
