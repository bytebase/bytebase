import { useId, useState } from "react";
import { useTranslation } from "react-i18next";
import { FormLabel } from "@/components/ui/form";
import { SegmentedControl } from "@/components/ui/segmented-control";
import { ValidationField, ValidationInput } from "./ValidationField";

type Identifier = { sid: string; serviceName: string };
type IdentifierMode = keyof Identifier;

const initialState = (source: Identifier) => ({
  source,
  drafts: source,
  mode: (source.sid ? "sid" : "serviceName") as IdentifierMode,
});

export function OracleConnectionFields({
  dataSourceId,
  instanceName,
  sid,
  serviceName,
  allowEdit,
  resetEvent,
  onChange,
}: Identifier & {
  dataSourceId: string;
  instanceName: string | undefined;
  allowEdit: boolean;
  resetEvent: number;
  onChange: (identifier: Identifier) => void;
}) {
  const { t } = useTranslation();
  const inputId = useId();
  const [cache, setCache] = useState(() => ({
    instanceName,
    resetEvent,
    entries: new Map([[dataSourceId, initialState({ sid, serviceName })]]),
  }));

  // Tab changes retain each data source's drafts. Revert and instance changes
  // discard all drafts, including those belonging to inactive tabs.
  const entries =
    cache.instanceName === instanceName && cache.resetEvent === resetEvent
      ? cache.entries
      : new Map<string, ReturnType<typeof initialState>>();
  const cached = entries.get(dataSourceId);
  const state =
    cached &&
    cached.source.sid === sid &&
    cached.source.serviceName === serviceName
      ? cached
      : initialState({ sid, serviceName });
  if (entries !== cache.entries || state !== cached) {
    setCache({
      instanceName,
      resetEvent,
      entries: new Map(entries).set(dataSourceId, state),
    });
  }

  const update = (mode: IdentifierMode, drafts: Identifier) => {
    const source = {
      sid: mode === "sid" ? drafts.sid : "",
      serviceName: mode === "serviceName" ? drafts.serviceName : "",
    };
    setCache({
      instanceName,
      resetEvent,
      entries: new Map(entries).set(dataSourceId, { mode, drafts, source }),
    });
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
