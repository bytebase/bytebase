import { create } from "@bufbuild/protobuf";
import { FieldMaskSchema } from "@bufbuild/protobuf/wkt";
import { cloneDeep, isEqual } from "lodash-es";
import { useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { EnvironmentSelect } from "@/react/components/EnvironmentSelect";
import { LabelListEditor } from "@/react/components/LabelListEditor";
import { Button } from "@/react/components/ui/button";
import { pushNotification, useDatabaseV1Store } from "@/store";
import type { Database } from "@/types/proto-es/v1/database_service_pb";
import { UpdateDatabaseRequestSchema } from "@/types/proto-es/v1/database_service_pb";
import {
  convertKVListToLabels,
  convertLabelsToKVList,
  getDatabaseProject,
  getInstanceResource,
  hasProjectPermissionV2,
} from "@/utils";

const EMPTY_LABELS: Record<string, string> = {};

export function DatabaseSettingsPanel({ database }: { database: Database }) {
  const { t } = useTranslation();
  const databaseStore = useDatabaseV1Store();
  const labels = database.labels ?? EMPTY_LABELS;
  const instanceEnvironment = getInstanceResource(database).environment ?? "";
  const allowClearEnvironment = !instanceEnvironment;
  const [kvList, setKVList] = useState(() =>
    convertLabelsToKVList(labels, true)
  );
  const [labelErrors, setLabelErrors] = useState<string[]>([]);
  const [isUpdatingLabels, setIsUpdatingLabels] = useState(false);

  useEffect(() => {
    setKVList(convertLabelsToKVList(labels, true));
  }, [labels]);

  const allowUpdateDatabase = useMemo(() => {
    const project = getDatabaseProject(database);
    return project
      ? hasProjectPermissionV2(project, "bb.databases.update")
      : false;
  }, [database]);

  const originalKVList = useMemo(
    () => convertLabelsToKVList(labels, true),
    [labels]
  );
  const dirty = useMemo(() => {
    return !isEqual(originalKVList, kvList);
  }, [kvList, originalKVList]);
  const allowSave = dirty && labelErrors.length === 0 && !isUpdatingLabels;

  const handleSelectEnvironment = async (name?: string) => {
    const nextEnvironment = name || "";

    if (!nextEnvironment && instanceEnvironment) {
      return;
    }
    if (nextEnvironment === database.effectiveEnvironment) {
      return;
    }

    const databasePatch = cloneDeep(database);
    databasePatch.environment = nextEnvironment;

    await databaseStore.updateDatabase(
      create(UpdateDatabaseRequestSchema, {
        database: databasePatch,
        updateMask: create(FieldMaskSchema, {
          paths: ["environment"],
        }),
      })
    );
    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("common.updated"),
    });
  };

  const handleCancel = () => {
    setKVList(originalKVList);
  };

  const handleSave = async () => {
    if (!allowSave) {
      return;
    }

    setIsUpdatingLabels(true);
    try {
      const labels = convertKVListToLabels(kvList, false);
      await databaseStore.updateDatabase(
        create(UpdateDatabaseRequestSchema, {
          database: {
            ...database,
            labels,
          },
          updateMask: create(FieldMaskSchema, {
            paths: ["labels"],
          }),
        })
      );
      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: t("common.updated"),
      });
    } finally {
      setIsUpdatingLabels(false);
    }
  };

  return (
    <div className="divide-y">
      <div className="flex flex-col gap-y-4 pb-7">
        <div>
          <p className="text-lg font-medium leading-7 text-main">
            {t("common.environment")}
          </p>
          <EnvironmentSelect
            className="mt-1 max-w-md"
            value={database.effectiveEnvironment || ""}
            clearable={allowClearEnvironment}
            disabled={!allowUpdateDatabase}
            renderSuffix={(environment) =>
              instanceEnvironment === environment.name ? (
                <span className="text-xs text-control-placeholder">
                  ({t("common.default")})
                </span>
              ) : null
            }
            onChange={handleSelectEnvironment}
          />
        </div>
      </div>
      <div className="flex flex-col gap-y-4 pt-7">
        <div className="flex items-center">
          <div className="flex-1">
            <div className="flex items-center">
              <p className="flex text-lg font-medium leading-7 text-main">
                {t("database.labels")}
              </p>
            </div>
          </div>
        </div>
        <div className="max-w-120">
          <LabelListEditor
            kvList={kvList}
            onChange={setKVList}
            readonly={!allowUpdateDatabase}
            showErrors={dirty}
            onErrorsChange={setLabelErrors}
          />
        </div>
        {dirty && (
          <div className="flex items-center justify-end gap-x-2">
            <Button variant="outline" onClick={handleCancel}>
              {t("common.revert")}
            </Button>
            <Button disabled={!allowSave} onClick={() => void handleSave()}>
              {t("common.save")}
            </Button>
          </div>
        )}
      </div>
    </div>
  );
}
