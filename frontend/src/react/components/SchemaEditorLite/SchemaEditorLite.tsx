import { cloneDeep } from "lodash-es";
import { Loader2 } from "lucide-react";
import {
  forwardRef,
  useCallback,
  useImperativeHandle,
  useMemo,
  useRef,
  useState,
} from "react";
import {
  Panel,
  Group as PanelGroup,
  Separator as PanelResizeHandle,
} from "react-resizable-panels";
import type { DatabaseMetadata } from "@/types/proto-es/v1/database_service_pb";
import { AsideTree } from "./Aside/AsideTree";
import { SchemaEditorProvider } from "./context";
import { EditorPanel } from "./EditorPanel";
import { resizeHandleClass } from "./resize";
import type {
  RebuildMetadataEditReset,
  SchemaEditorContextValue,
  SchemaEditorHandle,
  SchemaEditorProps,
} from "./types";
import { useAlgorithm } from "./useAlgorithm";
import { useEditStatus } from "./useEditStatus";
import { useScrollStatus } from "./useScrollStatus";
import { useSelection } from "./useSelection";
import { useTabs } from "./useTabs";

export const SchemaEditorLite = forwardRef<
  SchemaEditorHandle,
  SchemaEditorProps
>(function SchemaEditorLite(props, ref) {
  const {
    project,
    readonly = false,
    selectedRolloutObjects,
    targets: targetsProp,
    loading = false,
    hidePreview = false,
    options,
    onSelectedRolloutObjectsChange,
  } = props;

  const targets = useMemo(() => targetsProp ?? [], [targetsProp]);
  const ready = targets.length > 0;
  const combinedLoading = loading || !ready;

  // Tree rebuild trigger — version bumps cause AsideTree to re-derive tree data
  const [treeBuildVersion, setTreeBuildVersion] = useState(0);
  const rebuildTreeCallbackRef = useRef<
    ((openFirstChild: boolean) => void) | null
  >(null);

  const rebuildTree = useCallback((openFirstChild: boolean) => {
    setTreeBuildVersion((v) => v + 1);
    rebuildTreeCallbackRef.current?.(openFirstChild);
  }, []);

  // Preview refresh trigger — version bumps cause PreviewPane to re-generate DDL
  const [, setPreviewVersion] = useState(0);
  const refreshPreview = useCallback(() => {
    setPreviewVersion((v) => v + 1);
  }, []);

  // Sub-hooks
  const tabs = useTabs();
  const editStatus = useEditStatus();
  const selection = useSelection(
    selectedRolloutObjects,
    onSelectedRolloutObjectsChange
  );
  const scrollStatus = useScrollStatus();

  // Algorithm layer
  const algorithmCallbacks = useMemo(
    () => ({
      clearTabs: tabs.clearTabs,
      rebuildTree,
    }),
    [tabs.clearTabs, rebuildTree]
  );
  const { rebuildMetadataEdit, applyMetadataEdit } = useAlgorithm(
    editStatus,
    algorithmCallbacks
  );

  // Merge metadata callback (replaces Emittery "merge-metadata" event)
  const mergeMetadata = useCallback(
    (metadatas: DatabaseMetadata[]) => {
      for (const metadata of metadatas) {
        const target = targets.find((t) => t.metadata.name === metadata.name);
        if (!target) continue;

        mergeTableMetadataToTarget({
          metadata,
          mergeTo: target.metadata,
        });
        mergeTableMetadataToTarget({
          metadata,
          mergeTo: target.baselineMetadata,
        });
      }
    },
    [targets]
  );

  // Rebuild edit status callback
  const rebuildEditStatusCallback = useCallback(
    (resets: RebuildMetadataEditReset[]) => {
      if (ready) {
        targets.forEach((target) => {
          rebuildMetadataEdit(target, resets);
        });
      }
    },
    [ready, targets, rebuildMetadataEdit]
  );

  // Compose context value
  const contextValue = useMemo<SchemaEditorContextValue>(
    () => ({
      readonly,
      project,
      targets,
      hidePreview,
      options,
      treeBuildVersion,
      tabs,
      editStatus,
      selection,
      scrollStatus,
      rebuildTree,
      rebuildEditStatus: rebuildEditStatusCallback,
      refreshPreview,
      mergeMetadata,
      applyMetadataEdit,
      rebuildMetadataEdit,
    }),
    [
      readonly,
      project,
      targets,
      hidePreview,
      options,
      treeBuildVersion,
      tabs,
      editStatus,
      selection,
      scrollStatus,
      rebuildTree,
      rebuildEditStatusCallback,
      refreshPreview,
      mergeMetadata,
      applyMetadataEdit,
      rebuildMetadataEdit,
    ]
  );

  // Imperative handle
  useImperativeHandle(
    ref,
    () => ({
      applyMetadataEdit,
      refreshPreview,
      isDirty: editStatus.isDirty,
    }),
    [applyMetadataEdit, refreshPreview, editStatus.isDirty]
  );

  return (
    <SchemaEditorProvider value={contextValue}>
      <div className="bb-schema-editor relative flex size-full flex-col overflow-hidden rounded-sm border">
        {combinedLoading && (
          <div className="absolute inset-0 z-10 flex items-center justify-center bg-background/60">
            <Loader2 className="size-8 animate-spin text-accent" />
          </div>
        )}
        {ready && (
          <PanelGroup id="schema-editor" orientation="horizontal">
            <Panel defaultSize="25%" minSize="15%" maxSize="40%">
              <div className="size-full overflow-hidden">
                <AsideTree />
              </div>
            </Panel>
            <PanelResizeHandle className={resizeHandleClass("vertical")} />
            <Panel defaultSize="75%">
              <div className="size-full overflow-hidden">
                <EditorPanel />
              </div>
            </Panel>
          </PanelGroup>
        )}
      </div>
    </SchemaEditorProvider>
  );
});

function mergeTableMetadataToTarget({
  metadata,
  mergeTo,
}: {
  metadata: DatabaseMetadata;
  mergeTo: DatabaseMetadata;
}) {
  for (const schema of metadata.schemas) {
    if (schema.tables.length === 0) continue;
    const targetSchema = mergeTo.schemas.find((s) => s.name === schema.name);
    if (!targetSchema) continue;

    for (const table of schema.tables) {
      if (!targetSchema.tables.find((t) => t.name === table.name)) {
        targetSchema.tables.push(cloneDeep(table));
      }
    }
  }
}
