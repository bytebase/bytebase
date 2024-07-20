/*---------------------------------------------------------------------------------------------
 *  Copyright (c) Microsoft Corporation. All rights reserved.
 *  Licensed under the MIT License. See License.txt in the project root for license information.
 *--------------------------------------------------------------------------------------------*/
import { EditorAction2 } from '../../../browser/editorExtensions.js';
import { localizeWithPath } from '../../../../nls.js';
import { Categories } from '../../../../platform/action/common/actionCommonCategories.js';
import { Action2, MenuId } from '../../../../platform/actions/common/actions.js';
import { IConfigurationService } from '../../../../platform/configuration/common/configuration.js';
import { ContextKeyExpr } from '../../../../platform/contextkey/common/contextkey.js';
import { EditorContextKeys } from '../../../common/editorContextKeys.js';
import { StickyScrollController } from './stickyScrollController.js';
export class ToggleStickyScroll extends Action2 {
    constructor() {
        super({
            id: 'editor.action.toggleStickyScroll',
            title: {
                value: localizeWithPath('vs/editor/contrib/stickyScroll/browser/stickyScrollActions', 'toggleStickyScroll', "Toggle Sticky Scroll"),
                mnemonicTitle: localizeWithPath('vs/editor/contrib/stickyScroll/browser/stickyScrollActions', { key: 'mitoggleStickyScroll', comment: ['&& denotes a mnemonic'] }, "&&Toggle Sticky Scroll"),
                original: 'Toggle Sticky Scroll',
            },
            category: Categories.View,
            toggled: {
                condition: ContextKeyExpr.equals('config.editor.stickyScroll.enabled', true),
                title: localizeWithPath('vs/editor/contrib/stickyScroll/browser/stickyScrollActions', 'stickyScroll', "Sticky Scroll"),
                mnemonicTitle: localizeWithPath('vs/editor/contrib/stickyScroll/browser/stickyScrollActions', { key: 'miStickyScroll', comment: ['&& denotes a mnemonic'] }, "&&Sticky Scroll"),
            },
            menu: [
                { id: MenuId.CommandPalette },
                { id: MenuId.MenubarAppearanceMenu, group: '4_editor', order: 3 },
                { id: MenuId.StickyScrollContext }
            ]
        });
    }
    async run(accessor) {
        const configurationService = accessor.get(IConfigurationService);
        const newValue = !configurationService.getValue('editor.stickyScroll.enabled');
        return configurationService.updateValue('editor.stickyScroll.enabled', newValue);
    }
}
const weight = 100 /* KeybindingWeight.EditorContrib */;
export class FocusStickyScroll extends EditorAction2 {
    constructor() {
        super({
            id: 'editor.action.focusStickyScroll',
            title: {
                value: localizeWithPath('vs/editor/contrib/stickyScroll/browser/stickyScrollActions', 'focusStickyScroll', "Focus Sticky Scroll"),
                mnemonicTitle: localizeWithPath('vs/editor/contrib/stickyScroll/browser/stickyScrollActions', { key: 'mifocusStickyScroll', comment: ['&& denotes a mnemonic'] }, "&&Focus Sticky Scroll"),
                original: 'Focus Sticky Scroll',
            },
            precondition: ContextKeyExpr.and(ContextKeyExpr.has('config.editor.stickyScroll.enabled'), EditorContextKeys.stickyScrollVisible),
            menu: [
                { id: MenuId.CommandPalette },
            ]
        });
    }
    runEditorCommand(_accessor, editor) {
        StickyScrollController.get(editor)?.focus();
    }
}
export class SelectNextStickyScrollLine extends EditorAction2 {
    constructor() {
        super({
            id: 'editor.action.selectNextStickyScrollLine',
            title: {
                value: localizeWithPath('vs/editor/contrib/stickyScroll/browser/stickyScrollActions', 'selectNextStickyScrollLine.title', "Select next sticky scroll line"),
                original: 'Select next sticky scroll line'
            },
            precondition: EditorContextKeys.stickyScrollFocused.isEqualTo(true),
            keybinding: {
                weight,
                primary: 18 /* KeyCode.DownArrow */
            }
        });
    }
    runEditorCommand(_accessor, editor) {
        StickyScrollController.get(editor)?.focusNext();
    }
}
export class SelectPreviousStickyScrollLine extends EditorAction2 {
    constructor() {
        super({
            id: 'editor.action.selectPreviousStickyScrollLine',
            title: {
                value: localizeWithPath('vs/editor/contrib/stickyScroll/browser/stickyScrollActions', 'selectPreviousStickyScrollLine.title', "Select previous sticky scroll line"),
                original: 'Select previous sticky scroll line'
            },
            precondition: EditorContextKeys.stickyScrollFocused.isEqualTo(true),
            keybinding: {
                weight,
                primary: 16 /* KeyCode.UpArrow */
            }
        });
    }
    runEditorCommand(_accessor, editor) {
        StickyScrollController.get(editor)?.focusPrevious();
    }
}
export class GoToStickyScrollLine extends EditorAction2 {
    constructor() {
        super({
            id: 'editor.action.goToFocusedStickyScrollLine',
            title: {
                value: localizeWithPath('vs/editor/contrib/stickyScroll/browser/stickyScrollActions', 'goToFocusedStickyScrollLine.title', "Go to focused sticky scroll line"),
                original: 'Go to focused sticky scroll line'
            },
            precondition: EditorContextKeys.stickyScrollFocused.isEqualTo(true),
            keybinding: {
                weight,
                primary: 3 /* KeyCode.Enter */
            }
        });
    }
    runEditorCommand(_accessor, editor) {
        StickyScrollController.get(editor)?.goToFocused();
    }
}
export class SelectEditor extends EditorAction2 {
    constructor() {
        super({
            id: 'editor.action.selectEditor',
            title: {
                value: localizeWithPath('vs/editor/contrib/stickyScroll/browser/stickyScrollActions', 'selectEditor.title', "Select Editor"),
                original: 'Select Editor'
            },
            precondition: EditorContextKeys.stickyScrollFocused.isEqualTo(true),
            keybinding: {
                weight,
                primary: 9 /* KeyCode.Escape */
            }
        });
    }
    runEditorCommand(_accessor, editor) {
        StickyScrollController.get(editor)?.selectEditor();
    }
}
