import { computed, unref, watch } from "vue";
import { MaybeRef } from "@/types";
import { type Project, TenantMode } from "@/types/proto/v1/project_service";
import { parseDatabaseLabelValueByTemplate } from "@/utils";

export const useDBNameTemplateInputState = (
  project: MaybeRef<Project>,
  values: {
    databaseName: MaybeRef<string>;
    labels: MaybeRef<Record<string, string>>;
  }
) => {
  const TENANT_LABEL_KEY = "bb.tenant";
  const TENANT_LABEL_REGEXP_GROUP_NAME = "TENANT";

  const tenantValue = computed(() => {
    return unref(values.labels)[TENANT_LABEL_KEY] ?? "";
  });

  const parsedTenantValue = computed(() => {
    return parseDatabaseLabelValueByTemplate(
      unref(project).dbNameTemplate,
      unref(values.databaseName),
      TENANT_LABEL_REGEXP_GROUP_NAME
    );
  });

  watch(
    parsedTenantValue,
    (newValue, oldValue) => {
      const proj = unref(project);
      if (proj.tenantMode !== TenantMode.TENANT_MODE_ENABLED) return;
      if (!proj.dbNameTemplate) return;

      const tenant = tenantValue.value;
      if (tenant && tenant !== oldValue) {
        // Tenant value has been changed by user manually, don't touch it.
        return;
      }
      const labels = unref(values.labels);
      if (newValue) {
        labels[TENANT_LABEL_KEY] = newValue;
      } else {
        delete labels[TENANT_LABEL_KEY];
      }
    },
    { immediate: true }
  );
};
