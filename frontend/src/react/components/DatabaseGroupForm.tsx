import { create } from "@bufbuild/protobuf";
import { cloneDeep, head, isEqual } from "lodash-es";
import { useCallback, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import type { ConditionGroupExpr } from "@/plugins/cel";
import {
  buildCELExpr,
  emptySimpleExpr,
  resolveCELExpr,
  validateSimpleExpr,
  wrapAsGroup,
} from "@/plugins/cel";
import { ExprEditor } from "@/react/components/ExprEditor";
import { MatchedDatabaseView } from "@/react/components/MatchedDatabaseView";
import { ResourceIdField } from "@/react/components/ResourceIdField";
import { Button } from "@/react/components/ui/button";
import { Input } from "@/react/components/ui/input";
import {
  FactorList,
  getDatabaseGroupOptionConfigMap,
} from "@/react/lib/database-group/utils";
import { pushNotification, useDBGroupStore } from "@/store";
import {
  databaseGroupNamePrefix,
  getProjectNameAndDatabaseGroupName,
} from "@/store/modules/v1/common";
import type { ValidatedMessage } from "@/types";
import { isValidDatabaseGroupName } from "@/types";
import type { Expr as CELExpr } from "@/types/proto-es/google/api/expr/v1alpha1/syntax_pb";
import { ExprSchema } from "@/types/proto-es/google/type/expr_pb";
import type { DatabaseGroup } from "@/types/proto-es/v1/database_group_service_pb";
import {
  DatabaseGroupSchema,
  DatabaseGroupView,
} from "@/types/proto-es/v1/database_group_service_pb";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import {
  batchConvertCELStringToParsedExpr,
  batchConvertParsedExprToCELString,
} from "@/utils";

interface DatabaseGroupFormProps {
  readonly: boolean;
  project: Project;
  databaseGroup?: DatabaseGroup;
  className?: string;
  onDismiss: () => void;
  onCreated?: (databaseGroupName: string) => void;
}

export function DatabaseGroupForm({
  readonly,
  project,
  databaseGroup,
  className,
  onDismiss,
  onCreated,
}: DatabaseGroupFormProps) {
  const { t } = useTranslation();
  const dbGroupStore = useDBGroupStore();

  const [title, setTitle] = useState("");
  const [resourceId, setResourceId] = useState("");
  const [resourceIdValid, setResourceIdValid] = useState(false);
  const [expr, setExpr] = useState<ConditionGroupExpr>(
    wrapAsGroup(emptySimpleExpr())
  );

  const isCreating = databaseGroup === undefined;

  const matchedDatabaseNames = useMemo(
    () => databaseGroup?.matchedDatabases.map((db) => db.name) ?? [],
    [databaseGroup]
  );

  // Initialize state from databaseGroup in editing mode
  useEffect(() => {
    if (!databaseGroup || !isValidDatabaseGroupName(databaseGroup.name)) {
      return;
    }

    const [, groupName] = getProjectNameAndDatabaseGroupName(
      databaseGroup.name
    );
    setResourceId(groupName);
    setTitle(databaseGroup.title);

    if (databaseGroup.databaseExpr?.expression) {
      batchConvertCELStringToParsedExpr([
        databaseGroup.databaseExpr.expression,
      ]).then((exprList) => {
        if (exprList.length > 0) {
          const simpleExpr = resolveCELExpr(exprList[0]);
          setExpr(cloneDeep(wrapAsGroup(simpleExpr)));
        }
      });
    }
  }, [databaseGroup]);

  // In edit mode, ResourceIdField is readonly and never fires onValidationChange,
  // so skip the resourceIdValid gate when editing.
  const allowConfirm = useMemo(() => {
    const idOk = isCreating ? resourceIdValid : true;
    return idOk && title.trim() !== "" && validateSimpleExpr(expr);
  }, [isCreating, resourceIdValid, title, expr]);

  // Duplicate-ID validation for creation: try to fetch the group and fail if it exists.
  const validateResourceId = useCallback(
    async (id: string): Promise<ValidatedMessage[]> => {
      try {
        const name = `${project.name}/${databaseGroupNamePrefix}${id}`;
        await dbGroupStore.getOrFetchDBGroupByName(name, {
          silent: true,
          view: DatabaseGroupView.FULL,
        });
        // If fetch succeeded, the group already exists
        return [{ type: "error", message: `${id} already exists` }];
      } catch {
        // Not found — ID is available
        return [];
      }
    },
    [project.name, dbGroupStore]
  );

  const doConfirm = async () => {
    if (!allowConfirm) return;

    let celExpr: CELExpr | undefined;
    try {
      celExpr = await buildCELExpr(expr);
    } catch (error) {
      pushNotification({
        module: "bytebase",
        style: "CRITICAL",
        title: "CEL expression error occurred",
        description: (error as Error).message,
      });
      return;
    }

    const celStrings = await batchConvertParsedExprToCELString([celExpr!]);
    const celString = head(celStrings);
    if (!celString) {
      pushNotification({
        module: "bytebase",
        style: "CRITICAL",
        title: "CEL expression error occurred",
        description: "CEL expression is empty",
      });
      return;
    }

    if (isCreating) {
      await dbGroupStore.createDatabaseGroup({
        projectName: project.name,
        databaseGroup: create(DatabaseGroupSchema, {
          name: `${project.name}/databaseGroups/${resourceId}`,
          title: title,
          databaseExpr: create(ExprSchema, { expression: celString }),
        }),
        databaseGroupId: resourceId,
      });
      onCreated?.(resourceId);
    } else {
      if (!databaseGroup) return;

      const updateMask: string[] = [];
      if (!isEqual(databaseGroup.title, title)) {
        updateMask.push("title");
      }
      if (
        !isEqual(
          databaseGroup.databaseExpr,
          create(ExprSchema, { expression: celString })
        )
      ) {
        updateMask.push("database_expr");
      }
      await dbGroupStore.updateDatabaseGroup(
        {
          ...databaseGroup,
          title,
          databaseExpr: create(ExprSchema, { expression: celString }),
        },
        updateMask
      );
    }

    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: isCreating ? t("common.created") : t("common.updated"),
    });
    onDismiss();
  };

  return (
    <div className={`flex-1 flex flex-col ${className ?? ""}`}>
      <div className="flex-1 mb-6 px-4">
        <div className="w-full">
          <p className="font-medium text-main mb-2">{t("common.name")}</p>
          <Input
            value={title}
            onChange={(e) => setTitle(e.target.value)}
            disabled={readonly}
            maxLength={200}
          />
          <div className="mt-2">
            <ResourceIdField
              suffix
              value={resourceId}
              resourceName={t("common.database-group")}
              resourceTitle={title}
              readonly={!isCreating}
              validate={isCreating ? validateResourceId : undefined}
              onChange={setResourceId}
              onValidationChange={setResourceIdValid}
            />
          </div>
        </div>

        <hr className="my-6" />

        <div className="w-full grid grid-cols-5 gap-x-6">
          <div className="col-span-3">
            <p className="pl-1 font-medium text-main mb-2">
              {t("database-group.condition.self")}
            </p>
            <ExprEditor
              expr={expr}
              readonly={readonly}
              factorList={FactorList}
              optionConfigMap={getDatabaseGroupOptionConfigMap(project.name)}
              onUpdate={setExpr}
            />
          </div>
          <div className="col-span-2">
            <MatchedDatabaseView
              project={project.name}
              expr={expr}
              matchedDatabaseNames={readonly ? matchedDatabaseNames : undefined}
            />
          </div>
        </div>
      </div>

      {!readonly && (
        <div className="sticky bottom-0 z-10">
          <div className="flex justify-end w-full pt-4 pb-2 px-4 border-t border-block-border bg-background gap-x-3">
            <Button variant="outline" onClick={onDismiss}>
              {t("common.cancel")}
            </Button>
            <Button disabled={!allowConfirm} onClick={doConfirm}>
              {isCreating ? t("common.create") : t("common.update")}
            </Button>
          </div>
        </div>
      )}
    </div>
  );
}
