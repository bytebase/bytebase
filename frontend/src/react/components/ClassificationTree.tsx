import { ChevronRight } from "lucide-react";
import { type JSX, useCallback, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { SearchInput } from "@/react/components/ui/search-input";
import type { TreeDataNode } from "@/react/components/ui/tree";
import { Tree } from "@/react/components/ui/tree";
import type { DataClassificationSetting_DataClassificationConfig } from "@/types/proto-es/v1/setting_service_pb";

interface TreeNode {
  key: string;
  label: string;
  level?: number;
  children: TreeNode[];
}

interface ClassificationMap {
  [key: string]: {
    id: string;
    label: string;
    level?: number;
    children: ClassificationMap;
  };
}

function sortClassification(a: { id: string }, b: { id: string }): number {
  const id1s = a.id.split("-");
  const id2s = b.id.split("-");
  if (id1s.length !== id2s.length) return id1s.length - id2s.length;
  for (let i = 0; i < id1s.length; i++) {
    if (id1s[i] === id2s[i]) continue;
    if (Number.isNaN(Number(id1s[i])) || Number.isNaN(Number(id2s[i]))) {
      return id1s[i].localeCompare(id2s[i]);
    }
    return Number(id1s[i]) - Number(id2s[i]);
  }
  return 0;
}

function buildTreeData(
  config: DataClassificationSetting_DataClassificationConfig
): TreeNode[] {
  const classifications = Object.values(config.classification).sort(
    sortClassification
  );
  const map: ClassificationMap = {};
  for (const c of classifications) {
    const ids = c.id.split("-");
    let tmp = map;
    for (let i = 0; i < ids.length - 1; i++) {
      const parentKey = ids.slice(0, i + 1).join("-");
      if (!tmp[parentKey]) break;
      tmp = tmp[parentKey].children;
    }
    tmp[c.id] = {
      id: c.id,
      label: c.title,
      level: c.level,
      children: {},
    };
  }

  function toNodes(m: ClassificationMap): TreeNode[] {
    return Object.values(m)
      .sort(sortClassification)
      .map((item) => ({
        key: item.id,
        label: `${item.id} ${item.label}`,
        level: item.level,
        children: toNodes(item.children),
      }));
  }
  return toNodes(map);
}

const bgColorList = [
  "bg-green-200",
  "bg-yellow-200",
  "bg-orange-300",
  "bg-amber-500",
  "bg-red-500",
];

function LevelBadge({
  level,
  config,
}: {
  level: number;
  config: DataClassificationSetting_DataClassificationConfig;
}) {
  const levelObj = config.levels.find((l) => l.level === level);
  if (!levelObj) return null;
  const color = bgColorList[level - 1] ?? "bg-control-bg-hover";
  return (
    <span className={`ml-1 px-1 py-0.5 rounded-xs text-xs ${color}`}>
      {levelObj.title}
    </span>
  );
}

function highlightText(text: string, keyword: string) {
  if (!keyword) return text;
  const lower = text.toLowerCase();
  const kw = keyword.toLowerCase();
  const parts: (string | JSX.Element)[] = [];
  let cursor = 0;
  let idx = lower.indexOf(kw, cursor);
  while (idx !== -1) {
    if (idx > cursor) parts.push(text.slice(cursor, idx));
    parts.push(
      <b key={idx} className="text-accent">
        {text.slice(idx, idx + keyword.length)}
      </b>
    );
    cursor = idx + keyword.length;
    idx = lower.indexOf(kw, cursor);
  }
  if (cursor < text.length) parts.push(text.slice(cursor));
  return <>{parts}</>;
}

function toTreeData(node: TreeNode): TreeDataNode<TreeNode> {
  return {
    id: node.key,
    data: node,
    children: node.children.map(toTreeData),
  };
}

function collectAllIds(nodes: TreeDataNode<TreeNode>[]): string[] {
  const ids: string[] = [];
  for (const n of nodes) {
    ids.push(n.id);
    if (n.children)
      ids.push(...collectAllIds(n.children as TreeDataNode<TreeNode>[]));
  }
  return ids;
}

function collectMatchingIds(nodes: TreeNode[], keyword: string): Set<string> {
  const ids = new Set<string>();
  function walk(node: TreeNode): boolean {
    const selfMatches = node.label
      .toLowerCase()
      .includes(keyword.toLowerCase());
    const childMatches = node.children.some((c) => walk(c));
    if (selfMatches || childMatches) {
      ids.add(node.key);
      return true;
    }
    return false;
  }
  for (const n of nodes) walk(n);
  return ids;
}

export function ClassificationTree({
  config,
  onApply,
}: {
  readonly config: DataClassificationSetting_DataClassificationConfig;
  readonly onApply?: (classificationId: string) => void;
}) {
  const { t } = useTranslation();
  const [searchText, setSearchText] = useState("");
  const treeData = useMemo(
    () => buildTreeData(config).map(toTreeData),
    [config]
  );
  const expandedIds = useMemo(() => collectAllIds(treeData), [treeData]);
  const matchingIds = useMemo(() => {
    if (!searchText) return null;
    return collectMatchingIds(
      treeData.map((n) => n.data),
      searchText
    );
  }, [treeData, searchText]);

  const searchMatch = useCallback(
    (node: TreeDataNode<TreeNode>) => {
      if (!matchingIds) return true;
      return matchingIds.has(node.data.key);
    },
    [matchingIds]
  );

  return (
    <div className="flex flex-col gap-4">
      <div>
        <SearchInput
          value={searchText}
          onChange={(e) => setSearchText(e.target.value)}
          placeholder={t("schema-template.classification.search")}
        />
      </div>
      <Tree<TreeNode>
        data={treeData}
        expandedIds={expandedIds}
        searchTerm={searchText || undefined}
        searchMatch={searchMatch}
        height={400}
        rowHeight={32}
        renderNode={({ node, style }) => {
          const treeNode = node.data.data;
          const hasChildren = (node.data.children?.length ?? 0) > 0;
          return (
            <div
              style={style}
              className="flex items-center gap-1 py-1 px-1 rounded-xs hover:bg-control-bg cursor-pointer select-none text-sm"
              onClick={() => {
                if (!hasChildren && onApply) {
                  onApply(treeNode.key);
                } else {
                  node.toggle();
                }
              }}
            >
              {hasChildren ? (
                <ChevronRight
                  className={`size-4 shrink-0 transition-transform ${node.isOpen ? "rotate-90" : ""}`}
                />
              ) : (
                <span className="size-4 shrink-0" />
              )}
              <span>{highlightText(treeNode.label, searchText)}</span>
              {treeNode.level != null && treeNode.level !== 0 && (
                <LevelBadge level={treeNode.level} config={config} />
              )}
            </div>
          );
        }}
      />
    </div>
  );
}
