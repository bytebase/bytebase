import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { EngineIcon } from "@/components/EngineIcon";
import { EnvironmentLabel } from "@/components/EnvironmentLabel";
import { Combobox, type ComboboxOption } from "@/components/ui/combobox";
import { useAppStore } from "@/stores/app";
import type { Engine } from "@/types/proto-es/v1/common_pb";
import type { Database } from "@/types/proto-es/v1/database_service_pb";
import {
  extractDatabaseResourceName,
  getDatabaseEnvironment,
  getDefaultPagination,
  getInstanceResource,
} from "@/utils";

interface BaseProps {
  placeholder?: string;
  disabled?: boolean;
  className?: string;
  portal?: boolean;
  projectName?: string;
  environmentName?: string;
  allowedEngineTypeList?: Engine[];
}

interface SingleProps {
  multiple?: false;
  value: string;
  onChange: (value: string, database: Database | undefined) => void;
}

interface MultiProps {
  multiple: true;
  value: string[];
  // `databases[i]` corresponds to `values[i]`, resolved best-effort from the
  // current page and the app store's database cache; a value neither has (yet)
  // resolved is `undefined` (the chip still renders from its resource name).
  onChange: (values: string[], databases: (Database | undefined)[]) => void;
}

export type DatabaseSelectProps = BaseProps & (SingleProps | MultiProps);

export function DatabaseSelect(props: DatabaseSelectProps) {
  const {
    placeholder,
    disabled,
    className,
    portal,
    projectName,
    environmentName,
    allowedEngineTypeList,
  } = props;
  const { t } = useTranslation();
  const workspaceResourceName = useAppStore((s) => s.workspaceResourceName());
  // The app store's global database cache — populated by every fetch below
  // (upserted fresh) plus databases fetched elsewhere (e.g. Sync Schema's
  // deep-link preload) and not evicted by this picker's own project-parent
  // fetches. It resolves a selected value that is not on the current page, so
  // chips stay labeled across searches and `onChange` payloads carry the real,
  // fresh `Database`. Read non-reactively on purpose: `onChange` reads it live
  // at click time (payload always current), and every flow that sets a selected
  // value either populates the cache first (deep-link) or triggers a local
  // re-render (search -> setDatabases), so a selected option never lingers as a
  // synthesized label in practice.
  const getDatabaseByName = useAppStore((s) => s.getDatabaseByName);

  // Current page / latest search results (drives the dropdown rows).
  const [databases, setDatabases] = useState<Database[]>([]);
  // Monotonic fetch id: only the latest in-flight fetch may apply its result,
  // so an out-of-order (slow mount / stale search) response can't clobber newer
  // results. Mirrors DatabaseResourceSelector / the plan-detail DatabaseSelector.
  const fetchIdRef = useRef(0);

  // Stabilize engines array to avoid re-fetching on every render.
  const enginesRef = useRef(allowedEngineTypeList);
  const stableEngines = useMemo(() => {
    const prev = enginesRef.current;
    if (
      prev &&
      allowedEngineTypeList &&
      prev.length === allowedEngineTypeList.length &&
      prev.every((e, i) => e === allowedEngineTypeList[i])
    ) {
      return prev;
    }
    enginesRef.current = allowedEngineTypeList;
    return allowedEngineTypeList;
  }, [allowedEngineTypeList]);

  const fetchDatabases = useCallback(
    (query: string) => {
      const fetchId = ++fetchIdRef.current;
      useAppStore
        .getState()
        .fetchDatabases({
          parent: projectName ?? workspaceResourceName,
          filter: {
            environment: environmentName,
            engines: stableEngines,
            query,
          },
          pageSize: getDefaultPagination(),
          silent: true,
        })
        .then((result) => {
          // Drop stale responses so a slower earlier fetch can't overwrite the
          // results of a newer query.
          if (fetchId !== fetchIdRef.current) return;
          setDatabases(result.databases);
        })
        .catch(() => {
          /* keep existing options on error — never wipe the selection */
        });
    },
    [projectName, environmentName, stableEngines, workspaceResourceName]
  );

  useEffect(() => {
    fetchDatabases("");
  }, [fetchDatabases]);

  const selectedValues: string[] = useMemo(() => {
    if (props.multiple === true) return props.value;
    return props.value ? [props.value] : [];
  }, [props.multiple, props.value]);

  // Resolve a database name to its real `Database`: current page first, then
  // the store cache. Returns undefined when neither has it (a never-fetched
  // value) so callers can synthesize a label / omit it from the payload.
  // The store upserts every fetched database into its global cache, so the
  // current page is always a subset of the cache — resolve straight from it.
  // getDatabaseByName returns an "unknown" placeholder on a miss (its name won't
  // equal what we asked for), which we treat as unresolved.
  const resolveDatabase = useCallback(
    (name: string): Database | undefined => {
      const cached = getDatabaseByName(name);
      return cached.name === name ? cached : undefined;
    },
    [getDatabaseByName]
  );

  const buildOption = useCallback((database: Database): ComboboxOption => {
    const inst = getInstanceResource(database);
    const { databaseName, instance } = extractDatabaseResourceName(
      database.name
    );
    return {
      value: database.name,
      label: databaseName,
      description: instance,
      render: () => (
        <div className="flex flex-col gap-0.5">
          <div className="flex items-center gap-1.5">
            {inst.title && (
              <>
                <EngineIcon engine={inst.engine} className="h-4 w-4" />
                <span>{inst.title}</span>
                <span className="text-control-placeholder">&gt;</span>
              </>
            )}
            <EnvironmentLabel
              environmentName={getDatabaseEnvironment(database).name}
            />
            <span className="text-control-placeholder">&gt;</span>
            <span>{databaseName}</span>
          </div>
          <span className="text-xs text-control-placeholder">
            {database.name}
          </span>
        </div>
      ),
    };
  }, []);

  // Deduped union: current results + any selected value not on the page
  // (resolved from the cache, else a label-only synthesized option from the
  // name) — so a selected chip never loses its label after a search narrows
  // the page or for a pre-filled value that was never fetched.
  const options: ComboboxOption[] = useMemo(() => {
    const present = new Set(databases.map((d) => d.name));
    const opts = databases.map(buildOption);
    for (const name of selectedValues) {
      if (!name || present.has(name)) continue;
      present.add(name);
      const resolved = resolveDatabase(name);
      if (resolved) {
        opts.push(buildOption(resolved));
      } else {
        const { databaseName, instance } = extractDatabaseResourceName(name);
        opts.push({ value: name, label: databaseName, description: instance });
      }
    }
    return opts;
  }, [databases, selectedValues, buildOption, resolveDatabase]);

  const commonProps = {
    placeholder: placeholder ?? t("database.select"),
    noResultsText: t("common.no-data"),
    onSearch: fetchDatabases,
    disabled,
    className,
    portal,
    options,
  };

  if (props.multiple === true) {
    const onChange = props.onChange;
    return (
      <Combobox
        {...commonProps}
        multiple
        value={props.value}
        onChange={(values) => {
          // Index-aligned with `values`; unresolved entries are undefined.
          onChange(
            values,
            values.map((v) => resolveDatabase(v))
          );
        }}
      />
    );
  }

  const onChange = props.onChange;
  return (
    <Combobox
      {...commonProps}
      value={props.value}
      onChange={(name) => onChange(name, resolveDatabase(name))}
      renderValue={(opt) => {
        const database = resolveDatabase(opt.value);
        if (!database) return opt.label;
        const inst = getInstanceResource(database);
        return (
          <span className="flex items-center gap-1.5 truncate">
            <EngineIcon engine={inst.engine} className="h-4 w-4" />
            {extractDatabaseResourceName(database.name).databaseName}
          </span>
        );
      }}
    />
  );
}
