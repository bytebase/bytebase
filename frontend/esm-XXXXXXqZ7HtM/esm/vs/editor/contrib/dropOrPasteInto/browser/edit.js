/*---------------------------------------------------------------------------------------------
 *  Copyright (c) Microsoft Corporation. All rights reserved.
 *  Licensed under the MIT License. See License.txt in the project root for license information.
 *--------------------------------------------------------------------------------------------*/
import { ResourceTextEdit } from '../../../browser/services/bulkEditService.js';
export function createCombinedWorkspaceEdit(uri, ranges, edit) {
    return {
        edits: [
            ...ranges.map(range => new ResourceTextEdit(uri, typeof edit.insertText === 'string'
                ? { range, text: edit.insertText, insertAsSnippet: false }
                : { range, text: edit.insertText.snippet, insertAsSnippet: true })),
            ...(edit.additionalEdit?.edits ?? [])
        ]
    };
}
export function sortEditsByYieldTo(edits) {
    function yieldsTo(yTo, other) {
        return ('providerId' in yTo && yTo.providerId === other.providerId)
            || ('mimeType' in yTo && yTo.mimeType === other.handledMimeType);
    }
    // Build list of nodes each node yields to
    const yieldsToMap = new Map();
    for (const edit of edits) {
        for (const yTo of edit.yieldTo ?? []) {
            for (const other of edits) {
                if (other === edit) {
                    continue;
                }
                if (yieldsTo(yTo, other)) {
                    let arr = yieldsToMap.get(edit);
                    if (!arr) {
                        arr = [];
                        yieldsToMap.set(edit, arr);
                    }
                    arr.push(other);
                }
            }
        }
    }
    if (!yieldsToMap.size) {
        return Array.from(edits);
    }
    // Topological sort
    const visited = new Set();
    const tempStack = [];
    function visit(nodes) {
        if (!nodes.length) {
            return [];
        }
        const node = nodes[0];
        if (tempStack.includes(node)) {
            console.warn(`Yield to cycle detected for ${node.providerId}`);
            return nodes;
        }
        if (visited.has(node)) {
            return visit(nodes.slice(1));
        }
        let pre = [];
        const yTo = yieldsToMap.get(node);
        if (yTo) {
            tempStack.push(node);
            pre = visit(yTo);
            tempStack.pop();
        }
        visited.add(node);
        return [...pre, node, ...visit(nodes.slice(1))];
    }
    return visit(Array.from(edits));
}
