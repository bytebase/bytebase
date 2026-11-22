import { RotateCcw } from "lucide-react";
import { useEffect, useId, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@/components/ui/button";
import {
  FormControlGroup,
  FormControlRow,
  FormError,
  FormLabel,
} from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import { RadioGroup, RadioGroupItem } from "@/components/ui/radio-group";
import { Separator } from "@/components/ui/separator";
import { Switch } from "@/components/ui/switch";
import { formatTokenLimit, parseTokenLimit } from "../logic/tokenLimit";
import { useAgentStore } from "../store/agent";
import { DEFAULT_AGENT_PREFERENCES } from "../store/preferences";
import {
  AgentPopover,
  AgentPopoverContent,
  AgentPopoverTitle,
  AgentPopoverTrigger,
} from "./ui/AgentPopover";

const preferenceRowClassName =
  "flex flex-col gap-3 py-6 @min-[560px]/preferences:flex-row @min-[560px]/preferences:gap-6";
const preferenceControlClassName =
  "min-w-0 @min-[560px]/preferences:w-56 @min-[560px]/preferences:shrink-0";

export function AgentPreferences() {
  const { t } = useTranslation();
  const preferences = useAgentStore((state) => state.preferences);
  const storageError = useAgentStore((state) => state.preferencesStorageError);
  const sectionRef = useRef<HTMLElement>(null);
  const resetTriggerRef = useRef<HTMLButtonElement>(null);
  const [resetOpen, setResetOpen] = useState(false);
  const id = useId();
  const [tokenDraft, setTokenDraft] = useState(
    formatTokenLimit(preferences.tokenLimitPerRun)
  );
  const invalidTokenLimit = parseTokenLimit(tokenDraft) === null;

  useEffect(() => {
    sectionRef.current?.focus();
  }, []);

  useEffect(() => {
    setTokenDraft(formatTokenLimit(preferences.tokenLimitPerRun));
  }, [preferences.tokenLimitPerRun]);

  return (
    <section
      ref={sectionRef}
      tabIndex={-1}
      className="@container/agent-preferences size-full overflow-y-auto bg-background outline-none"
      data-agent-preferences
      aria-labelledby={`${id}-title`}
    >
      <div className="mx-auto w-full max-w-172 px-4 pt-2 pb-4 @min-[540px]/agent-preferences:px-6 @min-[540px]/agent-preferences:pb-6">
        <header className="flex min-w-0 flex-col gap-2 py-6">
          <h3 id={`${id}-title`} className="text-2xl font-bold text-main">
            {t("agent.preferences.title")}
          </h3>
          <p className="text-sm text-control-light">
            {t("agent.preferences.scope")}
          </p>
        </header>
        <Separator />
        <div className="@container/preferences">
          <div
            className="flex items-center justify-between gap-6 py-6"
            data-agent-preference-row="send"
          >
            <span className="flex min-w-0 flex-1 flex-col">
              <FormLabel htmlFor={`${id}-send`} className="w-fit text-main">
                {t("agent.preferences.send-with-enter")}
              </FormLabel>
              <span
                id={`${id}-shortcut-help`}
                className="text-xs text-control-light"
              >
                {preferences.sendWithEnter
                  ? t("agent.preferences.enter-help")
                  : t("agent.preferences.newline-help")}
              </span>
            </span>
            <span className="flex min-h-9 w-11 shrink-0 items-center justify-end pointer-coarse:min-h-11">
              <Switch
                id={`${id}-send`}
                checked={preferences.sendWithEnter}
                onCheckedChange={(sendWithEnter) =>
                  useAgentStore.getState().updatePreferences({ sendWithEnter })
                }
                aria-label={t("agent.preferences.send-with-enter")}
                aria-describedby={`${id}-shortcut-help`}
              />
            </span>
          </div>
          <Separator />

          <div
            className={preferenceRowClassName}
            data-agent-preference-row="tokens"
          >
            <FormLabel htmlFor={`${id}-tokens`} className="min-w-0 flex-1">
              {t("agent.preferences.token-limit")}
            </FormLabel>
            <div className={preferenceControlClassName}>
              <FormControlGroup>
                <FormControlRow>
                  <Input
                    id={`${id}-tokens`}
                    type="number"
                    inputMode="decimal"
                    min="0.001"
                    max={formatTokenLimit(Number.MAX_SAFE_INTEGER)}
                    step="0.001"
                    required
                    value={tokenDraft}
                    className="w-32 min-w-0 max-w-full pointer-coarse:text-base"
                    aria-invalid={Boolean(invalidTokenLimit)}
                    aria-describedby={
                      invalidTokenLimit
                        ? `${id}-tokens-unit ${id}-token-error`
                        : `${id}-tokens-unit`
                    }
                    onChange={(event) => {
                      const value = event.target.value;
                      setTokenDraft(value);
                      const parsed = parseTokenLimit(value);
                      if (parsed !== null) {
                        useAgentStore
                          .getState()
                          .updatePreferences({ tokenLimitPerRun: parsed });
                      }
                    }}
                  />
                  <span
                    id={`${id}-tokens-unit`}
                    className="shrink-0 text-sm text-control-light"
                  >
                    {t("agent.preferences.thousand-tokens")}
                  </span>
                </FormControlRow>
                {invalidTokenLimit && (
                  <FormError id={`${id}-token-error`}>
                    {t("agent.preferences.invalid-token-limit")}
                  </FormError>
                )}
              </FormControlGroup>
            </div>
          </div>
          <Separator />

          <div
            className={preferenceRowClassName}
            data-agent-preference-row="opening"
          >
            <span
              id={`${id}-opening`}
              className="min-w-0 flex-1 text-sm font-medium text-main"
            >
              {t("agent.preferences.when-opening")}
            </span>
            <div className={preferenceControlClassName}>
              <RadioGroup
                value={preferences.openingBehavior}
                onValueChange={(value) => {
                  if (value === "landing" || value === "continue") {
                    useAgentStore
                      .getState()
                      .updatePreferences({ openingBehavior: value });
                  }
                }}
                aria-labelledby={`${id}-opening`}
                className="flex-col items-start gap-2 pointer-coarse:gap-0"
              >
                <RadioGroupItem
                  value="landing"
                  className="pointer-coarse:min-h-11"
                >
                  {t("agent.preferences.show-landing")}
                </RadioGroupItem>
                <RadioGroupItem
                  value="continue"
                  className="pointer-coarse:min-h-11"
                >
                  {t("agent.preferences.continue-last-chat")}
                </RadioGroupItem>
              </RadioGroup>
            </div>
          </div>
          <Separator />
        </div>

        <div className="flex flex-col items-start gap-4 pt-4">
          <AgentPopover open={resetOpen} onOpenChange={setResetOpen}>
            <AgentPopoverTrigger
              render={
                <Button
                  ref={resetTriggerRef}
                  appearance="secondary"
                  size="sm"
                  className="-ml-2 text-control-light pointer-coarse:min-h-11"
                />
              }
            >
              <RotateCcw className="size-4" aria-hidden="true" />
              {t("agent.preferences.reset")}
            </AgentPopoverTrigger>
            <AgentPopoverContent
              finalFocus={resetTriggerRef}
              side="bottom"
              align="start"
              className="w-56 p-3"
              data-agent-reset-preferences-confirmation
            >
              <AgentPopoverTitle className="mb-3 font-normal">
                {t("agent.preferences.reset-confirmation")}
              </AgentPopoverTitle>
              <div className="flex justify-end gap-2">
                <Button
                  appearance="outline"
                  size="sm"
                  onClick={() => setResetOpen(false)}
                >
                  {t("common.cancel")}
                </Button>
                <Button
                  size="sm"
                  onClick={() => {
                    setResetOpen(false);
                    useAgentStore.getState().resetPreferences();
                    setTokenDraft(
                      formatTokenLimit(
                        DEFAULT_AGENT_PREFERENCES.tokenLimitPerRun
                      )
                    );
                  }}
                >
                  {t("common.reset")}
                </Button>
              </div>
            </AgentPopoverContent>
          </AgentPopover>
          {storageError && (
            <FormError>{t("agent.preferences.storage-failed")}</FormError>
          )}
        </div>
      </div>
    </section>
  );
}
