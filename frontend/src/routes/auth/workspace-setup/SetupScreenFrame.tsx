import { type ReactNode, useEffect, useRef } from "react";
import { useTranslation } from "react-i18next";
import { Separator } from "@/components/ui/separator";
import { StepIndicator } from "@/components/ui/step-indicator";
import { cn } from "@/lib/utils";
import type { SetupScreen } from "./draft";

export function SetupScreenFrame({
  screen,
  title,
  description,
  children,
  footer,
  onSubmit,
}: {
  readonly screen: SetupScreen;
  readonly title: string;
  readonly description: string;
  readonly children: ReactNode;
  readonly footer: ReactNode;
  readonly onSubmit: () => void;
}) {
  const { t } = useTranslation();
  const steps = [
    { key: "workspace", title: t("workspace-setup.steps.workspace") },
    { key: "goal", title: t("workspace-setup.steps.goal") },
    { key: "project", title: t("workspace-setup.steps.project") },
    { key: "team", title: t("workspace-setup.steps.team") },
  ];

  // Each screen is a new form, and the button that led here is gone, so the
  // keyboard would be left on the page body. It goes to the screen's first
  // answer: a text field, or the chosen option of a choice.
  const formRef = useRef<HTMLFormElement>(null);
  useEffect(() => {
    const first = formRef.current?.querySelector<HTMLElement>(
      'input:not([type="radio"]):not([type="hidden"]):not(:disabled), [role="radio"]'
    );
    const chosen =
      first?.getAttribute("role") === "radio"
        ? first
            .closest('[role="radiogroup"]')
            ?.querySelector<HTMLElement>('[role="radio"][aria-checked="true"]')
        : undefined;
    (chosen ?? first)?.focus();
  }, []);

  return (
    <form
      ref={formRef}
      className="mx-auto flex w-full max-w-lg flex-col gap-y-6"
      noValidate
      onSubmit={(event) => {
        event.preventDefault();
        onSubmit();
      }}
    >
      <StepIndicator steps={steps} currentKey={screen} />
      <div className="flex flex-col gap-y-1">
        <h1 className="text-xl font-semibold text-main">{title}</h1>
        <p className="text-sm text-control-light">{description}</p>
      </div>
      {children}
      <Separator />
      <div className="flex items-center justify-between gap-x-2">{footer}</div>
    </form>
  );
}

// The note under a field or question: a hint, or the reason Continue or Finish
// is disabled, which must sit with what it is about. A control references it
// by `aria-describedby`.
export function SetupNote({
  id,
  children,
}: {
  readonly id?: string;
  readonly children: ReactNode;
}) {
  return (
    <p id={id} className="text-sm text-control-light">
      {children}
    </p>
  );
}

export const choiceCardClassName = (selected: boolean) =>
  cn(
    "items-start rounded-sm border px-4 py-3",
    selected
      ? "border-accent bg-accent/5"
      : "border-control-border hover:bg-control-bg"
  );
