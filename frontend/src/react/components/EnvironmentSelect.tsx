import { useMemo } from "react";
import { useTranslation } from "react-i18next";
import { EnvironmentLabel } from "@/react/components/EnvironmentLabel";
import { Combobox } from "@/react/components/ui/combobox";
import { useEnvironmentList } from "@/react/hooks/useAppState";
import { formatEnvironmentName } from "@/types";
import type { Environment } from "@/types/v1/environment";

interface SingleProps {
  multiple?: false;
  value: string;
  onChange: (value: string) => void;
  /** Extra content rendered next to the selected env in the trigger and rows. */
  renderSuffix?: (environment: Environment) => React.ReactNode;
}

interface MultiProps {
  multiple: true;
  value: string[];
  onChange: (values: string[]) => void;
}

export type EnvironmentSelectProps = (SingleProps | MultiProps) & {
  placeholder?: string;
  disabled?: boolean;
  className?: string;
  portal?: boolean;
  clearable?: boolean;
};

export function EnvironmentSelect(props: EnvironmentSelectProps) {
  const { t } = useTranslation();
  const environments = useEnvironmentList();
  const renderSuffix = !props.multiple ? props.renderSuffix : undefined;

  const options = useMemo(
    () =>
      environments.map((env) => ({
        value: formatEnvironmentName(env.id),
        label: env.title,
        render: () => (
          <div className="flex flex-col gap-0.5">
            <div className="flex items-center gap-x-1">
              <EnvironmentLabel environment={env} />
              {renderSuffix?.(env)}
            </div>
            <span className="text-xs text-control-placeholder">
              {formatEnvironmentName(env.id)}
            </span>
          </div>
        ),
      })),
    [environments, renderSuffix]
  );

  const placeholder = props.placeholder ?? t("environment.select");
  const clearable = props.clearable ?? true;

  if (props.multiple) {
    return (
      <Combobox
        multiple
        value={props.value}
        onChange={props.onChange}
        placeholder={placeholder}
        noResultsText={t("common.no-data")}
        disabled={props.disabled}
        className={props.className}
        portal={props.portal}
        clearable={clearable}
        options={options}
      />
    );
  }

  return (
    <Combobox
      value={props.value}
      onChange={props.onChange}
      placeholder={placeholder}
      noResultsText={t("common.no-data")}
      disabled={props.disabled}
      className={props.className}
      portal={props.portal}
      clearable={clearable}
      renderValue={(opt) => {
        const env = environments.find(
          (e) => formatEnvironmentName(e.id) === opt.value
        );
        if (!env) return opt.label;
        return (
          <div className="flex items-center gap-x-1">
            <EnvironmentLabel environment={env} />
            {renderSuffix?.(env)}
          </div>
        );
      }}
      options={options}
    />
  );
}
