import {
  forwardRef,
  useCallback,
  useEffect,
  useImperativeHandle,
  useRef,
  useState,
} from "react";
import { useTranslation } from "react-i18next";
import { Input } from "@/react/components/ui/input";
import type { ValidatedMessage } from "@/types";
import { randomString } from "@/utils";

const VALID_CHARS = "abcdefghijklmnopqrstuvwxyz1234567890-";
const RESOURCE_ID_PATTERN = /^[a-z]([a-z0-9-]{0,61}[a-z0-9])?$/;

type ResourceType =
  | "environment"
  | "instance"
  | "project"
  | "idp"
  | "role"
  | "database-group"
  | "review-config";

export interface ResourceIdFieldRef {
  resourceId: string;
  isValidated: boolean;
  addValidationError: (message: string) => void;
}

interface ResourceIdFieldProps {
  value: string;
  resourceType: ResourceType;
  resourceName: string;
  resourceTitle?: string;
  suffix?: boolean;
  readonly?: boolean;
  validate?: (resourceId: string) => Promise<ValidatedMessage[]>;
  onChange?: (value: string) => void;
  onValidationChange?: (isValid: boolean) => void;
}

function randomCharacter(ch?: string): string {
  const letters = "abcdefghijklmnopqrstuvwxyz";
  const index = ch
    ? ch.charCodeAt(0) % letters.length
    : Math.floor(Math.random() * letters.length);
  return letters.charAt(index);
}

function escapeTitle(str: string): string {
  return str
    .toLowerCase()
    .split("")
    .map((char) => {
      if (char === " ") return "-";
      if (char.match(/\s/)) return "";
      if (VALID_CHARS.includes(char)) return char;
      return randomCharacter(char);
    })
    .join("");
}

export const ResourceIdField = forwardRef<
  ResourceIdFieldRef,
  ResourceIdFieldProps
>(function ResourceIdField(
  {
    value,
    resourceType: _resourceType, // reserved for future per-type behavior
    resourceName,
    resourceTitle,
    suffix = false,
    readonly = false,
    validate,
    onChange,
    onValidationChange,
  },
  ref
) {
  const { t } = useTranslation();
  const [manualEdit, setManualEdit] = useState(false);
  const [messages, setMessages] = useState<ValidatedMessage[]>([]);
  const randomSuffixRef = useRef(randomString(4).toLowerCase());
  const initializedRef = useRef(false);
  // Use refs for callbacks to avoid re-triggering the auto-generate effect
  const onChangeRef = useRef(onChange);
  onChangeRef.current = onChange;
  const validateRef = useRef(validate);
  validateRef.current = validate;
  const onValidationChangeRef = useRef(onValidationChange);
  onValidationChangeRef.current = onValidationChange;

  const updateMessages = useCallback((msgs: ValidatedMessage[]) => {
    setMessages(msgs);
    onValidationChangeRef.current?.(msgs.length === 0);
  }, []);

  const validateId = useCallback(
    async (id: string): Promise<ValidatedMessage[]> => {
      const result: ValidatedMessage[] = [];

      if (id === "") {
        result.push({
          type: "error",
          message: t("resource-id.validation.empty", {
            resource: resourceName,
          }),
        });
      } else if (id.length > 64) {
        result.push({
          type: "error",
          message: t("resource-id.validation.overflow", {
            resource: resourceName,
          }),
        });
      } else if (!RESOURCE_ID_PATTERN.test(id)) {
        result.push({
          type: "error",
          message: t("resource-id.validation.pattern", {
            resource: resourceName,
          }),
        });
      }

      if (validateRef.current && result.length === 0) {
        const custom = await validateRef.current(id);
        if (Array.isArray(custom)) {
          result.push(...custom);
        }
      }

      return result;
    },
    [resourceName, t]
  );

  // Auto-generate from title
  useEffect(() => {
    if (readonly || manualEdit) return;

    const parts: string[] = [];
    if (resourceTitle) {
      const escaped = escapeTitle(resourceTitle);
      if (suffix) {
        parts.push(escaped, randomSuffixRef.current);
      } else if (escaped) {
        parts.push(escaped);
      } else {
        parts.push(randomString(4).toLowerCase());
      }
    }
    const name = parts.join("-");

    onChangeRef.current?.(name);
    validateId(name).then((msgs) => {
      if (!initializedRef.current && msgs.length > 0) {
        const fallback = name + "-" + randomString(4).toLowerCase();
        onChangeRef.current?.(fallback);
        validateId(fallback).then(updateMessages);
      } else {
        updateMessages(msgs);
      }
      initializedRef.current = true;
    });
  }, [resourceTitle, readonly, manualEdit, suffix, validateId, updateMessages]);

  const handleManualInput = (newValue: string) => {
    onChange?.(newValue);
    setMessages([]);
    validateId(newValue).then(updateMessages);
  };

  useImperativeHandle(
    ref,
    () => ({
      get resourceId() {
        return value;
      },
      get isValidated() {
        return messages.length === 0;
      },
      addValidationError(message: string) {
        const newMessages: ValidatedMessage[] = [
          ...messages,
          { type: "error", message },
        ];
        setMessages(newMessages);
        onValidationChangeRef.current?.(false);
      },
    }),
    [value, messages]
  );

  const visible = readonly ? !!value : true;
  if (!visible) return null;

  return (
    <div>
      {readonly || !manualEdit ? (
        <div className="textinfolabel text-sm flex items-start flex-wrap gap-x-1">
          <div className="flex items-center gap-x-1">
            {t("resource-id.self", { resource: resourceName })}:
            {value ? (
              <span className="text-gray-600 font-medium mr-1">{value}</span>
            ) : (
              <span className="text-control-placeholder italic">-</span>
            )}
          </div>
          {!readonly && (
            <div>
              <span>{t("resource-id.cannot-be-changed-later")}</span>
              <button
                type="button"
                className="text-accent font-medium cursor-pointer hover:opacity-80 ml-1"
                onClick={() => setManualEdit(true)}
              >
                {t("common.edit")}
              </button>
            </div>
          )}
        </div>
      ) : (
        <div className="mt-1">
          <label className="textlabel flex items-center">
            {t("resource-id.self", { resource: resourceName })}
            <span className="ml-0.5 text-error">*</span>
          </label>
          <p className="textinfolabel mb-2 mt-1">
            {t("resource-id.description", { resource: resourceName })}
          </p>
          <Input
            value={value}
            onChange={(e) => handleManualInput(e.target.value)}
            placeholder={t("resource-id.self", { resource: resourceName })}
            className={
              messages.some((m) => m.type === "error") ? "border-error" : ""
            }
          />
        </div>
      )}
      {messages.length > 0 && (
        <ul className="w-full my-2 flex flex-col gap-y-2 list-disc list-outside pl-4">
          {messages.map((msg) => (
            <li
              key={msg.message}
              className={`break-words w-full text-xs ${
                msg.type === "warning" ? "text-yellow-600" : "text-red-600"
              }`}
            >
              {msg.message}
            </li>
          ))}
        </ul>
      )}
    </div>
  );
});
