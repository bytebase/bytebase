import { useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { Alert } from "@/components/ui/alert";
import {
  Collapsible,
  CollapsiblePanel,
  CollapsibleTrigger,
} from "@/components/ui/collapsible";
import { Separator } from "@/components/ui/separator";
import type { AgentApiAttempt } from "../logic/apiAttempt";
import { parseCallApiResult } from "../logic/callApiResult";
import { parseAgentQueryResult } from "../logic/queryResult";
import { apiCallNeedsConfirmation } from "../logic/tools/apiMethodClass";
import type {
  AgentAskUserOption,
  AgentAskUserResponse,
  ToolCall,
} from "../logic/types";
import { AgentApiOutcome } from "./AgentApiOutcome";
import { AgentQueryResult } from "./AgentQueryResult";

interface Props {
  toolCall: ToolCall;
  result?: string;
  missingResultAttempt?: AgentApiAttempt;
  hasRecoveryDraft?: boolean;
  hideQueryPreview?: boolean;
}

const parseJson = (value?: string): unknown => {
  if (!value) return undefined;
  try {
    return JSON.parse(value);
  } catch {
    return value;
  }
};

const formatJson = (value?: string): string => {
  if (!value) return "";
  try {
    return JSON.stringify(JSON.parse(value), null, 2);
  } catch {
    return value;
  }
};

const isQueryOperation = (value: unknown): boolean =>
  value === "SQLService/Query" || value === "bytebase.v1.SQLService.Query";

const parseAskUserOption = (value: unknown): AgentAskUserOption | null => {
  if (!value || typeof value !== "object") return null;

  const option = value as Record<string, unknown>;
  const label =
    typeof option.label === "string"
      ? option.label.trim()
      : typeof option.value === "string"
        ? option.value.trim()
        : "";
  const optionValue =
    typeof option.value === "string" ? option.value.trim() : "";

  if (!label || !optionValue) return null;

  return {
    label,
    value: optionValue,
    description:
      typeof option.description === "string" && option.description.trim()
        ? option.description.trim()
        : undefined,
  };
};

export function ToolCallCard({
  toolCall,
  result,
  missingResultAttempt,
  hasRecoveryDraft,
  hideQueryPreview = false,
}: Props) {
  const { t } = useTranslation();
  const [expanded, setExpanded] = useState(false);

  const resultText = result ?? "";
  const parsedArguments = useMemo(
    () => parseJson(toolCall.arguments),
    [toolCall.arguments]
  );
  const parsedResult = useMemo(() => parseJson(resultText), [resultText]);
  const resultRecord =
    typeof parsedResult === "object" &&
    parsedResult !== null &&
    !Array.isArray(parsedResult)
      ? (parsedResult as Record<string, unknown>)
      : undefined;
  const errorMessage =
    typeof resultRecord?.error === "string"
      ? resultRecord.error
      : resultRecord?.kind === "error" &&
          typeof resultRecord.message === "string"
        ? resultRecord.message
        : undefined;
  const hasErrorResult =
    errorMessage !== undefined || resultRecord?.kind === "error";
  const unmaskMethodExplanation = useMemo(() => {
    if (
      toolCall.name !== "ask_unmask_method" ||
      hasErrorResult ||
      resultRecord?.needsScope !== false ||
      (resultRecord.method !== "just-in-time" &&
        resultRecord.method !== "masking-exemption")
    ) {
      return undefined;
    }
    const explanation = resultRecord.methodExplanation;
    if (
      typeof explanation !== "object" ||
      explanation === null ||
      Array.isArray(explanation)
    ) {
      return undefined;
    }
    const { label, description, alternativeReason } = explanation as Record<
      string,
      unknown
    >;
    if (
      typeof label !== "string" ||
      !label.trim() ||
      typeof description !== "string" ||
      !description.trim() ||
      typeof alternativeReason !== "string" ||
      !alternativeReason.trim()
    ) {
      return undefined;
    }
    return { label, description, alternativeReason };
  }, [hasErrorResult, resultRecord, toolCall.name]);
  const operationId =
    toolCall.name === "call_api" &&
    typeof parsedArguments === "object" &&
    parsedArguments !== null &&
    !Array.isArray(parsedArguments) &&
    typeof (parsedArguments as Record<string, unknown>).operationId === "string"
      ? ((parsedArguments as Record<string, unknown>).operationId as string)
      : undefined;
  const callApi =
    toolCall.name === "call_api" ||
    (toolCall.name === "dom_action" &&
      resultRecord?.agentExecution !== undefined)
      ? parseCallApiResult(resultText, operationId)
      : undefined;
  const queryPreview = useMemo(() => {
    if (
      toolCall.name !== "call_api" ||
      typeof parsedArguments !== "object" ||
      parsedArguments === null ||
      !isQueryOperation(
        (parsedArguments as Record<string, unknown>).operationId
      )
    ) {
      return undefined;
    }
    const preview = parseAgentQueryResult(resultText);
    return preview?.resultSets.some(
      (resultSet) => resultSet.columns.length > 0 || resultSet.error
    )
      ? preview
      : undefined;
  }, [parsedArguments, resultText, toolCall.name]);
  const runtimeQuestionKey = {
    ask_masking_exemption_scope: "agent.masking.exemption-scope.question",
    ask_masking_rule_scope: "agent.masking.rule-scope.question",
    ask_masking_semantic_type: "agent.masking.semantic-type-choice.question",
  }[toolCall.name];
  const isAskUser = toolCall.name === "ask_user" || !!runtimeQuestionKey;
  const askError = isAskUser
    ? hasErrorResult
      ? errorMessage
      : resultText.startsWith("Error:")
        ? resultText
        : undefined
    : undefined;
  const isDone = toolCall.name === "done";
  const isCanceledAsk =
    isAskUser &&
    typeof parsedResult === "object" &&
    parsedResult !== null &&
    !Array.isArray(parsedResult) &&
    (parsedResult as Record<string, unknown>).kind === "cancelled";
  const isSkippedAsk =
    isAskUser &&
    typeof parsedResult === "object" &&
    parsedResult !== null &&
    !Array.isArray(parsedResult) &&
    (parsedResult as Record<string, unknown>).skipped === true;

  const askPrompt = useMemo(() => {
    if (runtimeQuestionKey) {
      return t(runtimeQuestionKey);
    }
    if (
      typeof parsedArguments === "object" &&
      parsedArguments &&
      "prompt" in parsedArguments &&
      typeof (parsedArguments as Record<string, unknown>).prompt === "string"
    ) {
      return (parsedArguments as Record<string, unknown>).prompt as string;
    }
    return "";
  }, [runtimeQuestionKey, parsedArguments, t]);

  const askKind = useMemo(() => {
    if (runtimeQuestionKey) return "choose" as const;
    if (
      typeof parsedArguments === "object" &&
      parsedArguments &&
      "kind" in parsedArguments
    ) {
      const kind = (parsedArguments as Record<string, unknown>).kind;
      if (kind === "confirm") return "confirm" as const;
      if (kind === "choose") return "choose" as const;
    }
    return "input" as const;
  }, [runtimeQuestionKey, parsedArguments]);

  const askDefaultValue = useMemo(() => {
    if (
      typeof parsedArguments === "object" &&
      parsedArguments &&
      "defaultValue" in parsedArguments &&
      typeof (parsedArguments as Record<string, unknown>).defaultValue ===
        "string"
    ) {
      return (parsedArguments as Record<string, unknown>)
        .defaultValue as string;
    }
    return "";
  }, [parsedArguments]);

  const askOptions = useMemo<AgentAskUserOption[]>(() => {
    if (
      typeof parsedArguments !== "object" ||
      !parsedArguments ||
      !("options" in parsedArguments) ||
      !Array.isArray((parsedArguments as Record<string, unknown>).options)
    ) {
      return [];
    }
    return ((parsedArguments as Record<string, unknown>).options as unknown[])
      .map((option) => parseAskUserOption(option))
      .filter((option): option is AgentAskUserOption => !!option);
  }, [parsedArguments]);

  const askResponse = useMemo<AgentAskUserResponse | null>(() => {
    if (
      typeof parsedResult !== "object" ||
      !parsedResult ||
      !("kind" in parsedResult) ||
      typeof (parsedResult as Record<string, unknown>).kind !== "string" ||
      !("answer" in parsedResult) ||
      typeof (parsedResult as Record<string, unknown>).answer !== "string"
    ) {
      return null;
    }

    const response = parsedResult as Record<string, unknown>;
    if (response.kind === "confirm") {
      if (typeof response.confirmed !== "boolean") return null;
      return response as unknown as AgentAskUserResponse;
    }
    if (response.kind === "choose") {
      if (typeof response.value !== "string") return null;
      return response as unknown as AgentAskUserResponse;
    }
    if (response.kind === "input") {
      return response as unknown as AgentAskUserResponse;
    }
    return null;
  }, [parsedResult]);

  const doneText = useMemo(() => {
    if (
      typeof parsedResult === "object" &&
      parsedResult !== null &&
      !Array.isArray(parsedResult) &&
      typeof (parsedResult as Record<string, unknown>).text === "string"
    ) {
      return (parsedResult as Record<string, unknown>).text as string;
    }
    if (
      typeof parsedArguments === "object" &&
      parsedArguments &&
      "text" in parsedArguments &&
      typeof (parsedArguments as Record<string, unknown>).text === "string"
    ) {
      return (parsedArguments as Record<string, unknown>).text as string;
    }
    return "";
  }, [parsedArguments, parsedResult]);

  const doneSuccess = useMemo(() => {
    if (
      typeof parsedResult === "object" &&
      parsedResult !== null &&
      !Array.isArray(parsedResult) &&
      typeof (parsedResult as Record<string, unknown>).success === "boolean"
    ) {
      return (parsedResult as Record<string, unknown>).success as boolean;
    }
    if (
      typeof parsedArguments === "object" &&
      parsedArguments &&
      "success" in parsedArguments
    ) {
      return (parsedArguments as Record<string, unknown>).success !== false;
    }
    return true;
  }, [parsedArguments, parsedResult]);
  const doneHasApiWarnings =
    doneSuccess &&
    typeof parsedResult === "object" &&
    parsedResult !== null &&
    !Array.isArray(parsedResult) &&
    typeof (parsedResult as Record<string, unknown>).failedApiCallCount ===
      "number" &&
    ((parsedResult as Record<string, unknown>).failedApiCallCount as number) >
      0;
  const doneHasRecoveryDraft =
    isDone &&
    hasRecoveryDraft &&
    resultRecord?.success === false &&
    typeof resultRecord.text === "string" &&
    resultRecord.text.trim().length > 0 &&
    resultRecord.skipped !== true;
  const renderStatus = () => {
    if (!resultText && missingResultAttempt) {
      return (
        <span className="text-warning">
          <AgentApiOutcome execution={missingResultAttempt.execution} />
        </span>
      );
    }
    if (callApi?.execution && apiCallNeedsConfirmation(operationId ?? "")) {
      return (
        <span
          className={callApi.requiresRecovery ? "text-warning" : "text-control"}
        >
          <AgentApiOutcome execution={callApi.execution} />
        </span>
      );
    }
    if (toolCall.name === "dom_action" && resultRecord?.success === false) {
      const manual = resultRecord.code === "manual-required";
      return (
        <span className={manual ? "text-warning" : "text-error"}>
          {manual ? t("agent.tool-manual-required") : t("agent.tool-failed")}
        </span>
      );
    }
    if (hasErrorResult || callApi?.error || askError) {
      return <span className="text-error">{t("agent.tool-failed")}</span>;
    }
    if (isAskUser) {
      if (isCanceledAsk) {
        return (
          <span className="text-control-light">
            {t("agent.tool-request-canceled")}
          </span>
        );
      }
      if (isSkippedAsk) {
        return (
          <span className="text-control-light">{t("common.skipped")}</span>
        );
      }
      return askResponse ? (
        <span className="text-success">
          {t("agent.tool-response-submitted")}
        </span>
      ) : resultText ? (
        <span className="text-control-light">{t("agent.tool-completed")}</span>
      ) : (
        <span className="text-warning">{t("agent.tool-ask-user")}</span>
      );
    }
    if (isDone) {
      if (doneHasApiWarnings || doneHasRecoveryDraft) {
        return (
          <span className="text-warning">
            {t("agent.tool-completed-with-warnings")}
          </span>
        );
      }
      return (
        <span className={doneSuccess ? "text-success" : "text-error"}>
          {doneSuccess ? t("agent.tool-completed") : t("agent.tool-failed")}
        </span>
      );
    }
    return resultText ? (
      <span className="text-success">&#10003;</span>
    ) : (
      <span className="animate-pulse text-control-placeholder">&#9679;</span>
    );
  };

  const renderExpandedContent = () => {
    if (isAskUser) {
      return (
        <>
          <div className="text-control-light">
            {askKind === "confirm"
              ? t("agent.tool-ask-user-confirm")
              : askKind === "choose"
                ? t("agent.tool-ask-user-choose")
                : t("agent.tool-ask-user-input")}
          </div>
          <div className="text-control-light">{t("agent.tool-prompt")}</div>
          <pre className="whitespace-pre-wrap break-all text-control">
            {askPrompt}
          </pre>
          {askDefaultValue && (
            <>
              <div className="text-control-light">
                {t("agent.tool-default-value")}
              </div>
              <pre className="whitespace-pre-wrap break-all text-control">
                {askDefaultValue}
              </pre>
            </>
          )}
          {askKind === "choose" && askOptions.length > 0 && (
            <>
              <div className="text-control-light">
                {t("agent.tool-options")}
              </div>
              <div className="flex flex-col gap-1">
                {askOptions.map((option) => (
                  <div
                    key={option.value}
                    className="rounded-sm border border-block-border bg-background px-2 py-1"
                  >
                    <div className="font-medium text-control">
                      {option.label}
                    </div>
                    <div className="text-control-light">{option.value}</div>
                    {option.description && (
                      <div className="text-control-light">
                        {option.description}
                      </div>
                    )}
                  </div>
                ))}
              </div>
            </>
          )}
          {askResponse && (
            <>
              <div className="text-control-light">{t("agent.tool-answer")}</div>
              <pre className="whitespace-pre-wrap break-all text-control">
                {askResponse.answer}
              </pre>
              {askResponse.kind === "choose" && (
                <>
                  <div className="text-control-light">
                    {t("agent.tool-value")}
                  </div>
                  <pre className="whitespace-pre-wrap break-all text-control">
                    {askResponse.value}
                  </pre>
                </>
              )}
            </>
          )}
        </>
      );
    }

    if (isDone) {
      return (
        <>
          <div className="text-control-light">{t("agent.result")}</div>
          <pre className="whitespace-pre-wrap break-all text-control">
            {doneText}
          </pre>
        </>
      );
    }

    return (
      <>
        <div className="text-control-light">{t("agent.args")}</div>
        <pre className="whitespace-pre-wrap break-all text-control">
          {formatJson(toolCall.arguments)}
        </pre>
        {resultText && (
          <>
            <div className="text-control-light">{t("agent.result")}</div>
            {queryPreview?.previewLevel &&
              queryPreview.rowsPartiallyRetained && (
                <div className="text-control-light">
                  {queryPreview.previewLevel === "omitted"
                    ? t("agent.query-result.preview-expired")
                    : t("agent.query-result.preview-partial")}
                </div>
              )}
            <pre className="max-h-32 overflow-y-auto whitespace-pre-wrap break-all text-control">
              {formatJson(resultText)}
            </pre>
          </>
        )}
        {!resultText && missingResultAttempt && (
          <p className="text-control-light">
            {t("agent.api-recovery.no-replay")}
          </p>
        )}
      </>
    );
  };

  return (
    <Collapsible
      open={expanded}
      onOpenChange={setExpanded}
      className="rounded-sm border border-block-border bg-control-bg text-xs"
    >
      <CollapsibleTrigger className="px-2 py-1.5">
        <span className="font-mono text-control-light">{toolCall.name}</span>
        {renderStatus()}
        <span className="ml-auto text-control-placeholder">
          {expanded ? "\u25BE" : "\u25B8"}
        </span>
      </CollapsibleTrigger>

      {askError && (
        <Alert
          variant="error"
          appearance="caption"
          aria-label={t("agent.tool-failed")}
          className="max-h-32 overflow-y-auto border-t border-block-border px-2 py-1.5 whitespace-pre-wrap break-words"
        >
          {askError}
        </Alert>
      )}

      {queryPreview && !hideQueryPreview && (
        <AgentQueryResult preview={queryPreview} />
      )}

      {unmaskMethodExplanation && (
        <>
          <Separator />
          <div
            className="flex flex-col gap-1 px-2 py-1.5 break-words"
            data-agent-unmask-method-explanation
          >
            <p className="font-medium text-main">
              {unmaskMethodExplanation.label}
            </p>
            <p className="text-control">
              {unmaskMethodExplanation.description}
            </p>
            <p className="text-control-light">
              {unmaskMethodExplanation.alternativeReason}
            </p>
          </div>
        </>
      )}

      <CollapsiblePanel className="flex flex-col gap-1 border-t border-block-border px-2 py-1.5">
        {renderExpandedContent()}
      </CollapsiblePanel>
    </Collapsible>
  );
}
