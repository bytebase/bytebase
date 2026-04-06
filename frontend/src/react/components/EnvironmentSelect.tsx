import { useMemo } from "react";
import { useTranslation } from "react-i18next";
import { EnvironmentLabel } from "@/react/components/EnvironmentLabel";
import { Combobox } from "@/react/components/ui/combobox";
import { useVueState } from "@/react/hooks/useVueState";
import { useEnvironmentV1Store } from "@/store";
import { formatEnvironmentName } from "@/types";

export interface EnvironmentSelectProps {
  value: string;
  onChange: (value: string) => void;
  placeholder?: string;
  disabled?: boolean;
  className?: string;
}

export function EnvironmentSelect({
  value,
  onChange,
  placeholder,
  disabled,
  className,
}: EnvironmentSelectProps) {
  const { t } = useTranslation();
  const environmentStore = useEnvironmentV1Store();

  const environments = useVueState(
    () => environmentStore.environmentList ?? []
  );

  const options = useMemo(
    () =>
      environments.map((env) => ({
        value: formatEnvironmentName(env.id),
        label: env.title,
        render: () => (
          <div className="flex flex-col gap-0.5">
            <EnvironmentLabel environment={env} />
            <span className="text-xs text-control-placeholder">
              {formatEnvironmentName(env.id)}
            </span>
          </div>
        ),
      })),
    [environments]
  );

  return (
    <Combobox
      value={value}
      onChange={onChange}
      placeholder={placeholder ?? t("environment.select")}
      noResultsText={t("common.no-data")}
      disabled={disabled}
      className={className}
      renderValue={(opt) => {
        const env = environments.find(
          (e) => formatEnvironmentName(e.id) === opt.value
        );
        if (!env) return opt.label;
        return <EnvironmentLabel environment={env} />;
      }}
      options={options}
    />
  );
}
