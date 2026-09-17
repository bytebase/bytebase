import { useId, useState } from "react";
import { useTranslation } from "react-i18next";
import { FormLabel } from "@/components/ui/form";
import { SegmentedControl } from "@/components/ui/segmented-control";
import { ValidationField, ValidationInput } from "./ValidationField";

type Identifier = { sid: string; serviceName: string };
type IdentifierMode = keyof Identifier;

const initialState = (source: Identifier, resetEvent: number) => ({
  source,
  resetEvent,
  drafts: source,
  mode: (source.sid ? "sid" : "serviceName") as IdentifierMode,
});

export function OracleConnectionFields({
  sid,
  serviceName,
  allowEdit,
  resetEvent,
  onChange,
}: Identifier & {
  allowEdit: boolean;
  resetEvent: number;
  onChange: (identifier: Identifier) => void;
}) {
  const { t } = useTranslation();
  const inputId = useId();
  const [state, setState] = useState(() =>
    initialState({ sid, serviceName }, resetEvent)
  );

  // Revert must discard inactive drafts even when the saved identifier is
  // already active. Our own updates preserve drafts by recording their source.
  if (
    state.resetEvent !== resetEvent ||
    state.source.sid !== sid ||
    state.source.serviceName !== serviceName
  ) {
    setState(initialState({ sid, serviceName }, resetEvent));
  }

  const update = (mode: IdentifierMode, drafts: Identifier) => {
    const source = {
      sid: mode === "sid" ? drafts.sid : "",
      serviceName: mode === "serviceName" ? drafts.serviceName : "",
    };
    setState({ mode, drafts, source, resetEvent });
    onChange(source);
  };

  const label = t(
    state.mode === "sid" ? "instance.sid" : "instance.service-name"
  );

  return (
    <>
      <ValidationField
        className="sm:col-span-3 sm:col-start-1"
        title={t("instance.connect-using")}
      >
        <SegmentedControl
          size="sm"
          ariaLabel={t("instance.connect-using")}
          value={state.mode}
          options={[
            { value: "serviceName", label: t("instance.service-name") },
            { value: "sid", label: t("instance.sid") },
          ]}
          disabled={!allowEdit}
          onValueChange={(mode) => update(mode, state.drafts)}
        />
      </ValidationField>
      <ValidationField
        validationField="serviceName"
        className="sm:col-span-3 sm:col-start-1"
      >
        <FormLabel htmlFor={inputId}>{label}</FormLabel>
        <ValidationInput
          id={inputId}
          value={state.drafts[state.mode]}
          placeholder={
            state.mode === "sid"
              ? t("instance.sid-placeholder")
              : t("instance.service-name-placeholder")
          }
          disabled={!allowEdit}
          onChange={(event) =>
            update(state.mode, {
              ...state.drafts,
              [state.mode]: event.target.value,
            })
          }
        />
      </ValidationField>
    </>
  );
}
