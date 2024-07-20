/*---------------------------------------------------------------------------------------------
 *  Copyright (c) Microsoft Corporation. All rights reserved.
 *  Licensed under the MIT License. See License.txt in the project root for license information.
 *--------------------------------------------------------------------------------------------*/
import '../../../../base/browser/ui/codicons/codiconStyles.js'; // The codicon symbol styles are defined here and must be loaded
import { Codicon } from '../../../../base/common/codicons.js';
import { CodeActionKind } from '../common/types.js';
import '../../symbolIcons/browser/symbolIcons.js'; // The codicon symbol colors are defined here and must be loaded to get colors
import { localizeWithPath } from '../../../../nls.js';
const uncategorizedCodeActionGroup = Object.freeze({ kind: CodeActionKind.Empty, title: localizeWithPath('vs/editor/contrib/codeAction/browser/codeActionMenu', 'codeAction.widget.id.more', 'More Actions...') });
const codeActionGroups = Object.freeze([
    { kind: CodeActionKind.QuickFix, title: localizeWithPath('vs/editor/contrib/codeAction/browser/codeActionMenu', 'codeAction.widget.id.quickfix', 'Quick Fix') },
    { kind: CodeActionKind.RefactorExtract, title: localizeWithPath('vs/editor/contrib/codeAction/browser/codeActionMenu', 'codeAction.widget.id.extract', 'Extract'), icon: Codicon.wrench },
    { kind: CodeActionKind.RefactorInline, title: localizeWithPath('vs/editor/contrib/codeAction/browser/codeActionMenu', 'codeAction.widget.id.inline', 'Inline'), icon: Codicon.wrench },
    { kind: CodeActionKind.RefactorRewrite, title: localizeWithPath('vs/editor/contrib/codeAction/browser/codeActionMenu', 'codeAction.widget.id.convert', 'Rewrite'), icon: Codicon.wrench },
    { kind: CodeActionKind.RefactorMove, title: localizeWithPath('vs/editor/contrib/codeAction/browser/codeActionMenu', 'codeAction.widget.id.move', 'Move'), icon: Codicon.wrench },
    { kind: CodeActionKind.SurroundWith, title: localizeWithPath('vs/editor/contrib/codeAction/browser/codeActionMenu', 'codeAction.widget.id.surround', 'Surround With'), icon: Codicon.symbolSnippet },
    { kind: CodeActionKind.Source, title: localizeWithPath('vs/editor/contrib/codeAction/browser/codeActionMenu', 'codeAction.widget.id.source', 'Source Action'), icon: Codicon.symbolFile },
    uncategorizedCodeActionGroup,
]);
export function toMenuItems(inputCodeActions, showHeaders, keybindingResolver) {
    if (!showHeaders) {
        return inputCodeActions.map((action) => {
            return {
                kind: "action" /* ActionListItemKind.Action */,
                item: action,
                group: uncategorizedCodeActionGroup,
                disabled: !!action.action.disabled,
                label: action.action.disabled || action.action.title,
                canPreview: !!action.action.edit?.edits.length,
            };
        });
    }
    // Group code actions
    const menuEntries = codeActionGroups.map(group => ({ group, actions: [] }));
    for (const action of inputCodeActions) {
        const kind = action.action.kind ? new CodeActionKind(action.action.kind) : CodeActionKind.None;
        for (const menuEntry of menuEntries) {
            if (menuEntry.group.kind.contains(kind)) {
                menuEntry.actions.push(action);
                break;
            }
        }
    }
    const allMenuItems = [];
    for (const menuEntry of menuEntries) {
        if (menuEntry.actions.length) {
            allMenuItems.push({ kind: "header" /* ActionListItemKind.Header */, group: menuEntry.group });
            for (const action of menuEntry.actions) {
                const group = menuEntry.group;
                allMenuItems.push({
                    kind: "action" /* ActionListItemKind.Action */,
                    item: action,
                    group: action.action.isAI ? { title: group.title, kind: group.kind, icon: Codicon.sparkle } : group,
                    label: action.action.title,
                    disabled: !!action.action.disabled,
                    keybinding: keybindingResolver(action.action),
                });
            }
        }
    }
    return allMenuItems;
}
