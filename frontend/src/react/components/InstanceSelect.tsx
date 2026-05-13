import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { EngineIcon } from "@/react/components/EngineIcon";
import { EnvironmentLabel } from "@/react/components/EnvironmentLabel";
import { Combobox } from "@/react/components/ui/combobox";
import { useInstanceV1Store } from "@/store";
import type { Engine } from "@/types/proto-es/v1/common_pb";
import type { Instance } from "@/types/proto-es/v1/instance_service_pb";
import { extractInstanceResourceName, getDefaultPagination } from "@/utils";

export interface InstanceSelectProps {
  value: string;
  onChange: (value: string, instance: Instance | undefined) => void;
  placeholder?: string;
  disabled?: boolean;
  className?: string;
  portal?: boolean;
  /** Only show instances with these engines */
  engines?: Engine[];
}

export function InstanceSelect({
  value,
  onChange,
  placeholder,
  disabled,
  className,
  portal,
  engines,
}: InstanceSelectProps) {
  const { t } = useTranslation();
  const instanceStore = useInstanceV1Store();
  const [instances, setInstances] = useState<Instance[]>([]);

  // Stabilize engines array to avoid re-fetching on every render
  const enginesRef = useRef(engines);
  const stableEngines = useMemo(() => {
    const prev = enginesRef.current;
    if (
      prev &&
      engines &&
      prev.length === engines.length &&
      prev.every((e, i) => e === engines[i])
    ) {
      return prev;
    }
    enginesRef.current = engines;
    return engines;
  }, [engines]);

  const fetchInstances = useCallback(
    (query: string) => {
      instanceStore
        .fetchInstanceList({
          pageSize: getDefaultPagination(),
          filter: { query, engines: stableEngines },
        })
        .then((result) => setInstances(result.instances));
    },
    [instanceStore, stableEngines]
  );

  useEffect(() => {
    fetchInstances("");
  }, [fetchInstances]);

  const handleChange = useCallback(
    (name: string) => {
      const inst = instances.find((i) => i.name === name);
      onChange(name, inst);
    },
    [instances, onChange]
  );

  return (
    <Combobox
      value={value}
      onChange={handleChange}
      placeholder={placeholder ?? t("common.instance")}
      noResultsText={t("common.no-data")}
      onSearch={fetchInstances}
      disabled={disabled}
      className={className}
      portal={portal}
      renderValue={(opt) => {
        const inst = instances.find((i) => i.name === opt.value);
        if (!inst) return opt.label;
        return (
          <span className="flex items-center gap-1.5 truncate">
            <EngineIcon engine={inst.engine} className="h-4 w-4" />
            {inst.title}
          </span>
        );
      }}
      options={instances.map((inst) => ({
        value: inst.name,
        label: inst.title,
        description: extractInstanceResourceName(inst.name),
        render: () => (
          <div className="flex flex-col gap-0.5">
            <div className="flex items-center gap-1.5">
              {inst.environment && (
                <>
                  <EnvironmentLabel environmentName={inst.environment} />
                  <span className="text-control-placeholder">&gt;</span>
                </>
              )}
              <EngineIcon engine={inst.engine} className="h-4 w-4" />
              <span>{inst.title}</span>
            </div>
            <span className="text-xs text-control-placeholder">
              {extractInstanceResourceName(inst.name)}
            </span>
          </div>
        ),
      }))}
    />
  );
}
