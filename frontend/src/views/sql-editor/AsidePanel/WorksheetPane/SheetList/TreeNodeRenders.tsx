import {
  FileCodeIcon,
  FolderCodeIcon,
  FolderMinusIcon,
  FolderOpenIcon,
  FolderSyncIcon,
  FolderPenIcon,
  MoreHorizontalIcon,
  StarIcon,
  UsersIcon,
  XIcon,
} from "lucide-vue-next";
import { NTooltip } from "naive-ui";
import { defineComponent, type PropType } from "vue";
import { t } from "@/plugins/i18n";
import { useUserStore, useWorkSheetStore, useCurrentUserV1, useSQLEditorTabStore, useTabViewStateStore } from "@/store";
import {
  type Worksheet,
  Worksheet_Visibility,
} from "@/types/proto-es/v1/worksheet_service_pb";
import type { WorsheetFolderNode, SheetViewMode } from "@/views/sql-editor/Sheet";

// Prefix icon component
export const TreeNodePrefix = defineComponent({
  name: "TreeNodePrefix",
  props: {
    node: {
      type: Object as PropType<WorsheetFolderNode>,
      required: true,
    },
    expandedKeys: {
      type: Object as PropType<Set<string>>,
      required: true,
    },
    rootPath: {
      type: String,
      required: true,
    },
    view: {
      type: String as PropType<SheetViewMode>,
      required: true,
    },
  },
  setup(props) {
    return () => {
      if (props.node.worksheet) {
        // the node is file
        return <FileCodeIcon class="w-4 h-auto text-gray-600" />;
      }
      if (props.expandedKeys.has(props.node.key)) {
        // is opened folder
        return <FolderOpenIcon class="w-4 h-auto text-gray-600" />;
      }
      if (props.node.key === props.rootPath) {
        // root folder icon
        switch (props.view) {
          case "draft":
            return <FolderPenIcon class="w-4 h-auto text-gray-600" />;
          case "shared":
            return <FolderSyncIcon class="w-4 h-auto text-gray-600" />;
          default:
            return <FolderCodeIcon class="w-4 h-auto text-gray-600" />;
        }
      }
      if (props.node.empty) {
        // empty folder icon
        return <FolderMinusIcon class="w-4 h-auto text-gray-600" />;
      }
      // fallback to normal folder icon
      return <FolderCodeIcon class="w-4 h-auto text-gray-600" />;
    };
  },
});

// Suffix icons component
export const TreeNodeSuffix = defineComponent({
  name: "TreeNodeSuffix",
  props: {
    node: {
      type: Object as PropType<WorsheetFolderNode>,
      required: true,
    },
    view: {
      type: String as PropType<SheetViewMode>,
      required: true,
    },
  },
  emits: ["contextMenuShow", "sharePanelShow", "toggleStar", "delete"],
  setup(props, { emit }) {
    const userStore = useUserStore();
    const worksheetStore = useWorkSheetStore();
    const me = useCurrentUserV1();
    const { removeViewState } = useTabViewStateStore();
    const tabStore = useSQLEditorTabStore();

    const visibilityDisplayName = (visibility: Worksheet_Visibility) => {
      switch (visibility) {
        case Worksheet_Visibility.PRIVATE:
          return t("sql-editor.private");
        case Worksheet_Visibility.PROJECT_READ:
          return t("sql-editor.project-read");
        case Worksheet_Visibility.PROJECT_WRITE:
          return t("sql-editor.project-write");
        default:
          return "";
      }
    };

    const creatorForSheet = (sheet: Worksheet) => {
      return (
        userStore.getUserByIdentifier(sheet.creator)?.title ?? sheet.creator
      );
    };

    const isWorksheetCreator = (worksheet: Worksheet) => {
      return worksheet.creator === `users/${me.value.email}`;
    };

    const renderShareIcon = (worksheet: Worksheet) => {
      if (
        worksheet.visibility !== Worksheet_Visibility.PROJECT_READ &&
        worksheet.visibility !== Worksheet_Visibility.PROJECT_WRITE
      ) {
        return null;
      }

      return (
        <NTooltip>
          {{
            trigger: () => (
              <UsersIcon
                class="w-4 text-gray-400"
                onClick={(e) => emit("sharePanelShow", e, props.node)}
              />
            ),
            default: () => {
              return (
                <div>
                  <div>
                    {t("common.visibility")}
                    {": "}
                    {visibilityDisplayName(worksheet.visibility)}
                  </div>
                  {isWorksheetCreator(worksheet) ? null : (
                    <div>
                      {t("common.creator")}
                      {": "}
                      {creatorForSheet(worksheet)}
                    </div>
                  )}
                </div>
              );
            },
          }}
        </NTooltip>
      );
    };

    return () => {
      if (props.view === "draft") {
        if (!props.node.worksheet) {
          return null
        }
        return <XIcon class="w-4 h-auto text-gray-600" onClick={() => {
          if (props.node.worksheet && props.node.worksheet.name) {
            const draft = tabStore.tabList.find((t) => t.id === props.node.worksheet!.name);
            if (draft) {
              tabStore.removeTab(draft);
            }
            removeViewState(props.node.worksheet.name);
          }
        }} />
      }

      if (!props.node.worksheet) {
        return (
          <MoreHorizontalIcon
            class="w-4 h-auto text-gray-600"
            onClick={(e: MouseEvent) => emit("contextMenuShow", e, props.node)}
          />
        );
      }

      const worksheet = worksheetStore.getWorksheetByName(props.node.worksheet.name)
      if (!worksheet) {
        return null
      }

      return (
        <div class="inline-flex gap-1">
          {renderShareIcon(worksheet)}
          <StarIcon
            class={`w-4 h-auto text-gray-400 ${worksheet.starred ? "text-yellow-400" : ""}`}
            onClick={() =>
              props.node.worksheet && emit("toggleStar", props.node.worksheet)
            }
          />
          <MoreHorizontalIcon
            class="w-4 h-auto text-gray-600"
            onClick={(e: MouseEvent) => emit("contextMenuShow", e, props.node)}
          />
        </div>
      );
    };
  },
});
