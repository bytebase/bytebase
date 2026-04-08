import {
  closestCenter,
  DndContext,
  type DragEndEvent,
  DragOverlay,
  type DragStartEvent,
  KeyboardSensor,
  PointerSensor,
  useSensor,
  useSensors,
} from "@dnd-kit/core";
import {
  arrayMove,
  SortableContext,
  useSortable,
  verticalListSortingStrategy,
} from "@dnd-kit/sortable";
import { CSS } from "@dnd-kit/utilities";
import { GripVertical, Pencil, Plus, Trash2 } from "lucide-react";
import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import {
  PermissionGuard,
  usePermissionCheck,
} from "@/react/components/PermissionGuard";
import { Button } from "@/react/components/ui/button";
import { pushNotification, useWorkspaceApprovalSettingStore } from "@/store";
import type { LocalApprovalRule } from "@/types";
import type { WorkspaceApprovalSetting_Rule_Source } from "@/types/proto-es/v1/setting_service_pb";
import { WorkspaceApprovalSetting_Rule_Source as RuleSource } from "@/types/proto-es/v1/setting_service_pb";
import { RuleEditDialog } from "./RuleEditDialog";
import { approvalSourceText, formatApprovalFlow } from "./utils";

interface RulesSectionProps {
  source: WorkspaceApprovalSetting_Rule_Source;
  rules: LocalApprovalRule[];
  allowAdmin: boolean;
  hasFeature: boolean;
  onShowFeatureModal: () => void;
}

function SortableRow({
  rule,
  index,
  allowAdmin,
  hasFeature,
  onEdit,
  onDelete,
  onShowFeatureModal,
}: {
  rule: LocalApprovalRule;
  index: number;
  allowAdmin: boolean;
  hasFeature: boolean;
  onEdit: (rule: LocalApprovalRule) => void;
  onDelete: (rule: LocalApprovalRule) => void;
  onShowFeatureModal: () => void;
}) {
  const { t } = useTranslation();
  const [confirmingDelete, setConfirmingDelete] = useState(false);

  const {
    attributes,
    listeners,
    setNodeRef,
    transform,
    transition,
    isDragging,
  } = useSortable({ id: rule.uid });

  const style = {
    transform: CSS.Transform.toString(transform),
    transition,
    opacity: isDragging ? 0.5 : undefined,
  };

  const handleEditClick = () => {
    if (!hasFeature) {
      onShowFeatureModal();
      return;
    }
    onEdit(rule);
  };

  const handleDeleteClick = () => {
    if (!hasFeature) {
      onShowFeatureModal();
      return;
    }
    setConfirmingDelete(true);
  };

  const handleConfirmDelete = () => {
    setConfirmingDelete(false);
    onDelete(rule);
  };

  return (
    <div
      ref={setNodeRef}
      style={style}
      className={`grid grid-cols-[40px_200px_1fr_280px_140px] border-b border-gray-100 last:border-b-0 hover:bg-gray-50 ${
        index % 2 === 1 ? "bg-gray-50/50" : ""
      }`}
    >
      <div className="flex items-center justify-center px-2 py-2">
        <GripVertical
          className="h-4 w-4 cursor-grab text-gray-400 active:cursor-grabbing"
          {...attributes}
          {...listeners}
        />
      </div>
      <div className="truncate px-3 py-2" title={rule.title}>
        {rule.title || "-"}
      </div>
      <div className="overflow-hidden text-ellipsis whitespace-nowrap px-3 py-2">
        <code className="font-mono text-sm">{rule.condition || "true"}</code>
      </div>
      <div
        className="overflow-hidden text-ellipsis whitespace-nowrap px-3 py-2"
        title={formatApprovalFlow(rule.flow)}
      >
        {formatApprovalFlow(rule.flow)}
      </div>
      <div className="flex items-center gap-x-1 px-3 py-2">
        {confirmingDelete ? (
          <div className="flex items-center gap-x-1">
            <Button
              variant="destructive"
              size="sm"
              className="h-6 px-2 text-xs"
              onClick={handleConfirmDelete}
            >
              {t("common.delete")}
            </Button>
            <Button
              variant="outline"
              size="sm"
              className="h-6 px-2 text-xs"
              onClick={() => setConfirmingDelete(false)}
            >
              {t("common.cancel")}
            </Button>
          </div>
        ) : (
          <>
            <button
              type="button"
              className="inline-flex h-6 w-6 items-center justify-center rounded-xs hover:bg-gray-200"
              onClick={handleEditClick}
            >
              <Pencil className="h-3 w-3" />
            </button>
            {allowAdmin && (
              <button
                type="button"
                className="inline-flex h-6 w-6 items-center justify-center rounded-xs hover:bg-gray-200"
                onClick={handleDeleteClick}
              >
                <Trash2 className="h-3 w-3" />
              </button>
            )}
          </>
        )}
      </div>
    </div>
  );
}

function RuleRowOverlay({ rule }: { rule: LocalApprovalRule }) {
  return (
    <div className="grid grid-cols-[40px_200px_1fr_280px_140px] border border-gray-200 bg-blue-50 opacity-50">
      <div className="flex items-center justify-center px-2 py-2">
        <GripVertical className="h-4 w-4 text-gray-400" />
      </div>
      <div className="truncate px-3 py-2">{rule.title || "-"}</div>
      <div className="overflow-hidden text-ellipsis whitespace-nowrap px-3 py-2">
        <code className="font-mono text-sm">{rule.condition || "true"}</code>
      </div>
      <div className="overflow-hidden text-ellipsis whitespace-nowrap px-3 py-2">
        {formatApprovalFlow(rule.flow)}
      </div>
      <div className="px-3 py-2" />
    </div>
  );
}

export function RulesSection({
  source,
  rules,
  allowAdmin,
  hasFeature,
  onShowFeatureModal,
}: RulesSectionProps) {
  const { t } = useTranslation();
  const store = useWorkspaceApprovalSettingStore();
  const [canEdit] = usePermissionCheck(["bb.settings.set"]);

  const [localRules, setLocalRules] = useState(rules);
  useEffect(() => setLocalRules(rules), [rules]);

  const [activeId, setActiveId] = useState<string | null>(null);
  const [dialogOpen, setDialogOpen] = useState(false);
  const [dialogMode, setDialogMode] = useState<"create" | "edit">("create");
  const [editingRule, setEditingRule] = useState<
    LocalApprovalRule | undefined
  >();

  const isFallback = source === RuleSource.SOURCE_UNSPECIFIED;

  const sensors = useSensors(
    useSensor(PointerSensor),
    useSensor(KeyboardSensor)
  );

  const handleAddRule = () => {
    if (!hasFeature) {
      onShowFeatureModal();
      return;
    }
    setDialogMode("create");
    setEditingRule(undefined);
    setDialogOpen(true);
  };

  const handleEditRule = (rule: LocalApprovalRule) => {
    setDialogMode("edit");
    setEditingRule(rule);
    setDialogOpen(true);
  };

  const handleDeleteRule = async (rule: LocalApprovalRule) => {
    try {
      await store.deleteRule(rule.uid);
      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: t("common.deleted"),
      });
    } catch {
      // Error handled by store
    }
  };

  const handleDragStart = (event: DragStartEvent) => {
    setActiveId(event.active.id as string);
  };

  const handleDragEnd = async (event: DragEndEvent) => {
    const { active, over } = event;
    setActiveId(null);

    if (!over || active.id === over.id) return;

    const oldIndex = localRules.findIndex((r) => r.uid === active.id);
    const newIndex = localRules.findIndex((r) => r.uid === over.id);

    if (oldIndex === newIndex) return;

    if (!hasFeature) {
      onShowFeatureModal();
      return;
    }

    // Optimistic local reorder
    setLocalRules(arrayMove(localRules, oldIndex, newIndex));

    try {
      await store.reorderRules(source, oldIndex, newIndex);
      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: t("common.updated"),
      });
    } catch {
      // Revert on error
      setLocalRules([...rules]);
    }
  };

  const handleSaveRule = async (
    ruleData: Omit<LocalApprovalRule, "uid"> & { uid?: string }
  ) => {
    try {
      if (dialogMode === "create") {
        await store.addRule(ruleData as Omit<LocalApprovalRule, "uid">);
      } else if (editingRule && ruleData.uid) {
        await store.updateRule(ruleData.uid, ruleData);
      }
      setDialogOpen(false);
      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: t("common.updated"),
      });
    } catch {
      // Error handled by store
    }
  };

  const activeRule = activeId
    ? localRules.find((r) => r.uid === activeId)
    : undefined;

  return (
    <div className="flex flex-col gap-y-2">
      <div className="flex items-center justify-between">
        <div className="text-base font-medium">
          {approvalSourceText(source)}
        </div>
        <PermissionGuard permissions={["bb.settings.set"]}>
          <Button variant="outline" disabled={!canEdit} onClick={handleAddRule}>
            <Plus className="h-4 w-4" />
            {t("common.create")}
          </Button>
        </PermissionGuard>
      </div>

      <div className="rounded-sm border border-gray-200 text-sm">
        {/* Table Header */}
        <div className="grid grid-cols-[40px_200px_1fr_280px_140px] border-b border-gray-200 bg-gray-50 font-medium text-gray-600">
          <div className="w-10 px-2 py-2" />
          <div className="px-3 py-2">{t("common.title")}</div>
          <div className="px-3 py-2">{t("cel.condition.self")}</div>
          <div className="px-3 py-2">
            {t("custom-approval.approval-flow.self")}
          </div>
          <div className="w-[140px] px-3 py-2">{t("common.operations")}</div>
        </div>

        {/* Draggable Body */}
        <DndContext
          sensors={sensors}
          collisionDetection={closestCenter}
          onDragStart={handleDragStart}
          onDragEnd={handleDragEnd}
        >
          <SortableContext
            items={localRules.map((r) => r.uid)}
            strategy={verticalListSortingStrategy}
          >
            {localRules.map((rule, index) => (
              <SortableRow
                key={rule.uid}
                rule={rule}
                index={index}
                allowAdmin={allowAdmin}
                hasFeature={hasFeature}
                onEdit={handleEditRule}
                onDelete={handleDeleteRule}
                onShowFeatureModal={onShowFeatureModal}
              />
            ))}
          </SortableContext>
          <DragOverlay>
            {activeRule ? <RuleRowOverlay rule={activeRule} /> : null}
          </DragOverlay>
        </DndContext>

        {/* Empty State */}
        {localRules.length === 0 && (
          <div className="px-3 py-4 text-center text-gray-400">
            {t("common.no-data")}
          </div>
        )}
      </div>

      <RuleEditDialog
        open={dialogOpen}
        onOpenChange={setDialogOpen}
        mode={dialogMode}
        source={source}
        rule={editingRule}
        isFallback={isFallback}
        allowAdmin={allowAdmin}
        hasFeature={hasFeature}
        onShowFeatureModal={onShowFeatureModal}
        onSave={handleSaveRule}
      />
    </div>
  );
}
