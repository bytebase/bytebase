import { useMemo } from "react";
import { useTranslation } from "react-i18next";
import { EnvironmentLabel } from "@/react/components/EnvironmentLabel";
import { Combobox } from "@/react/components/ui/combobox";
import { useEnvironmentList } from "@/react/hooks/useAppState";
import { formatEnvironmentName } from "@/types";
import type { Environment } from "@/types/v1/environment";

export interface EnvironmentSelectProps {
  value: string;
  onChange: (value: string) => void;
  placeholder?: string;
  disabled?: boolean;
  className?: string;
  portal?: boolean;
  clearable?: boolean;
  renderSuffix?: (environment: Environment) => React.ReactNode;
}

export function EnvironmentSelect({
  value,
  onChange,
  placeholder,
  disabled,
  className,
  portal,
  clearable = true,
  renderSuffix,
}: EnvironmentSelectProps) {
  const { t } = useTranslation();
  const environments = useEnvironmentList();

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

  return (
    <Combobox
      value={value}
      onChange={onChange}
      placeholder={placeholder ?? t("environment.select")}
      noResultsText={t("common.no-data")}
      disabled={disabled}
      className={className}
      portal={portal}
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
