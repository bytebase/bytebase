import { useCallback, useEffect, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { Combobox, type ComboboxOption } from "@/components/ui/combobox";
import { usePaginatedSelect } from "@/components/usePaginatedSelect";
import { cn } from "@/lib/utils";
import { useAppStore } from "@/stores/app";
import type { WorkloadIdentity } from "@/types/proto-es/v1/workload_identity_service_pb";
import { getDefaultPagination } from "@/utils";

interface WorkloadIdentitySelectProps {
  projectName: string;
  value: string;
  onChange: (
    value: string,
    workloadIdentity: WorkloadIdentity | undefined
  ) => void;
  placeholder?: string;
  disabled?: boolean;
  className?: string;
  portal?: boolean;
}

export function WorkloadIdentitySelect({
  projectName,
  value,
  onChange,
  placeholder,
  disabled,
  className,
  portal,
}: WorkloadIdentitySelectProps) {
  const { t } = useTranslation();
  const listWorkloadIdentities = useAppStore(
    (state) => state.listWorkloadIdentities
  );
  const getWorkloadIdentity = useAppStore((state) => state.getWorkloadIdentity);
  const selectedWorkloadIdentity = useAppStore((state) =>
    value ? state.getWorkloadIdentity(value) : undefined
  );

  const listPage = useCallback(
    async (query: string, pageToken: string) => {
      const response = await listWorkloadIdentities({
        parent: projectName,
        filter: { query },
        pageSize: getDefaultPagination(),
        pageToken,
        showDeleted: false,
      });
      return {
        items: response.workloadIdentities,
        nextPageToken: response.nextPageToken,
      };
    },
    [listWorkloadIdentities, projectName]
  );
  const {
    items: workloadIdentities,
    search,
    hasMore,
    loadingMore,
    loadMore,
  } = usePaginatedSelect({ fetchPage: listPage });

  useEffect(() => {
    search("");
  }, [search]);

  const options = useMemo(() => {
    const toOption = (identity: WorkloadIdentity): ComboboxOption => ({
      value: identity.name,
      label: identity.title || identity.email,
      description: identity.email,
    });
    const options = workloadIdentities.map(toOption);
    if (
      selectedWorkloadIdentity &&
      !workloadIdentities.some(
        (identity) => identity.name === selectedWorkloadIdentity.name
      )
    ) {
      options.push(toOption(selectedWorkloadIdentity));
    }
    return options;
  }, [selectedWorkloadIdentity, workloadIdentities]);

  return (
    <Combobox
      value={value}
      options={options}
      placeholder={placeholder}
      noResultsText={t("common.no-data")}
      onChange={(name) =>
        onChange(name, name ? getWorkloadIdentity(name) : undefined)
      }
      onSearch={search}
      hasMore={hasMore}
      loadingMore={loadingMore}
      onLoadMore={loadMore}
      disabled={disabled}
      className={cn("w-full", className)}
      portal={portal}
    />
  );
}
