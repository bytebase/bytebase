import { useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import type {
  AgentAskUserOption,
  AgentAskUserResponse,
  ToolCall,
} from "../logic/types";

interface Props {
  toolCall: ToolCall;
  result?: string;
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

export function ToolCallCard({ toolCall, result }: Props) {
  const { t } = useTranslation();
  const [expanded, setExpanded] = useState(false);

  const resultText = result ?? "";
  const parsedArguments = useMemo(
    () => parseJson(toolCall.arguments),
    [toolCall.arguments]
  );
  const parsedResult = useMemo(() => parseJson(resultText), [resultText]);
  const isAskUser = toolCall.name === "ask_user";
  const isDone = toolCall.name === "done";

  const askPrompt = useMemo(() => {
    if (
      typeof parsedArguments === "object" &&
      parsedArguments &&
      "prompt" in parsedArguments &&
      typeof (parsedArguments as Record<string, unknown>).prompt === "string"
    ) {
      return (parsedArguments as Record<string, unknown>).prompt as string;
    }
    return "";
  }, [parsedArguments]);

  const askKind = useMemo(() => {
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
  }, [parsedArguments]);

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
      typeof parsedArguments === "object" &&
      parsedArguments &&
      "text" in parsedArguments &&
      typeof (parsedArguments as Record<string, unknown>).text === "string"
    ) {
      return (parsedArguments as Record<string, unknown>).text as string;
    }
    return "";
  }, [parsedArguments]);

  const doneSuccess = useMemo(() => {
    if (
      typeof parsedArguments === "object" &&
      parsedArguments &&
      "success" in parsedArguments
    ) {
      return (parsedArguments as Record<string, unknown>).success !== false;
    }
    return true;
  }, [parsedArguments]);

  const renderStatus = () => {
    if (isAskUser) {
      return resultText ? (
        <span className="text-green-500">
          {t("agent.tool-response-submitted")}
        </span>
      ) : (
        <span className="text-amber-600">{t("agent.tool-ask-user")}</span>
      );
    }
    if (isDone) {
      return (
        <span className={doneSuccess ? "text-green-500" : "text-red-500"}>
          {doneSuccess ? t("agent.tool-completed") : t("agent.tool-failed")}
        </span>
      );
    }
    return resultText ? (
      <span className="text-green-500">&#10003;</span>
    ) : (
      <span className="animate-pulse text-gray-400">&#9679;</span>
    );
  };

  const renderExpandedContent = () => {
    if (isAskUser) {
      return (
        <>
          <div className="text-gray-500">
            {askKind === "confirm"
              ? t("agent.tool-ask-user-confirm")
              : askKind === "choose"
                ? t("agent.tool-ask-user-choose")
                : t("agent.tool-ask-user-input")}
          </div>
          <div className="text-gray-500">{t("agent.tool-prompt")}</div>
          <pre className="whitespace-pre-wrap break-all text-gray-700">
            {askPrompt}
          </pre>
          {askDefaultValue && (
            <>
              <div className="text-gray-500">
                {t("agent.tool-default-value")}
              </div>
              <pre className="whitespace-pre-wrap break-all text-gray-700">
                {askDefaultValue}
              </pre>
            </>
          )}
          {askKind === "choose" && askOptions.length > 0 && (
            <>
              <div className="text-gray-500">{t("agent.tool-options")}</div>
              <div className="space-y-1">
                {askOptions.map((option) => (
                  <div
                    key={option.value}
                    className="rounded border border-gray-200 bg-white px-2 py-1"
                  >
                    <div className="font-medium text-gray-700">
                      {option.label}
                    </div>
                    <div className="text-gray-500">{option.value}</div>
                    {option.description && (
                      <div className="text-gray-500">{option.description}</div>
                    )}
                  </div>
                ))}
              </div>
            </>
          )}
          {askResponse && (
            <>
              <div className="text-gray-500">{t("agent.tool-answer")}</div>
              <pre className="whitespace-pre-wrap break-all text-gray-700">
                {askResponse.answer}
              </pre>
              {askResponse.kind === "choose" && (
                <>
                  <div className="text-gray-500">{t("agent.tool-value")}</div>
                  <pre className="whitespace-pre-wrap break-all text-gray-700">
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
          <div className="text-gray-500">{t("agent.result")}</div>
          <pre className="whitespace-pre-wrap break-all text-gray-700">
            {doneText}
          </pre>
        </>
      );
    }

    return (
      <>
        <div className="text-gray-500">{t("agent.args")}</div>
        <pre className="whitespace-pre-wrap break-all text-gray-700">
          {formatJson(toolCall.arguments)}
        </pre>
        {resultText && (
          <>
            <div className="text-gray-500">{t("agent.result")}</div>
            <pre className="max-h-32 overflow-y-auto whitespace-pre-wrap break-all text-gray-700">
              {formatJson(resultText)}
            </pre>
          </>
        )}
      </>
    );
  };

  return (
    <div className="rounded border bg-gray-50 text-xs">
      <div
        className="flex cursor-pointer items-center gap-x-2 px-2 py-1.5"
        onClick={() => setExpanded(!expanded)}
      >
        <span className="font-mono text-gray-600">{toolCall.name}</span>
        {renderStatus()}
        <span className="ml-auto text-gray-400">
          {expanded ? "\u25BE" : "\u25B8"}
        </span>
      </div>

      {expanded && (
        <div className="space-y-1 border-t px-2 py-1.5">
          {renderExpandedContent()}
        </div>
      )}
    </div>
  );
}
