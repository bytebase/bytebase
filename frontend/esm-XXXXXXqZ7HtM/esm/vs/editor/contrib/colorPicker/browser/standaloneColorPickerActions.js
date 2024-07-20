/*---------------------------------------------------------------------------------------------
 *  Copyright (c) Microsoft Corporation. All rights reserved.
 *  Licensed under the MIT License. See License.txt in the project root for license information.
 *--------------------------------------------------------------------------------------------*/
import { EditorAction, EditorAction2, registerEditorAction } from '../../../browser/editorExtensions.js';
import { localizeWithPath } from '../../../../nls.js';
import { StandaloneColorPickerController } from './standaloneColorPickerWidget.js';
import { EditorContextKeys } from '../../../common/editorContextKeys.js';
import { MenuId, registerAction2 } from '../../../../platform/actions/common/actions.js';
import './colorPicker.css';
export class ShowOrFocusStandaloneColorPicker extends EditorAction2 {
    constructor() {
        super({
            id: 'editor.action.showOrFocusStandaloneColorPicker',
            title: {
                value: localizeWithPath('vs/editor/contrib/colorPicker/browser/standaloneColorPickerActions', 'showOrFocusStandaloneColorPicker', "Show or Focus Standalone Color Picker"),
                mnemonicTitle: localizeWithPath('vs/editor/contrib/colorPicker/browser/standaloneColorPickerActions', { key: 'mishowOrFocusStandaloneColorPicker', comment: ['&& denotes a mnemonic'] }, "&&Show or Focus Standalone Color Picker"),
                original: 'Show or Focus Standalone Color Picker',
            },
            precondition: undefined,
            menu: [
                { id: MenuId.CommandPalette },
            ]
        });
    }
    runEditorCommand(_accessor, editor) {
        StandaloneColorPickerController.get(editor)?.showOrFocus();
    }
}
class HideStandaloneColorPicker extends EditorAction {
    constructor() {
        super({
            id: 'editor.action.hideColorPicker',
            label: localizeWithPath('vs/editor/contrib/colorPicker/browser/standaloneColorPickerActions', {
                key: 'hideColorPicker',
                comment: [
                    'Action that hides the color picker'
                ]
            }, "Hide the Color Picker"),
            alias: 'Hide the Color Picker',
            precondition: EditorContextKeys.standaloneColorPickerVisible.isEqualTo(true),
            kbOpts: {
                primary: 9 /* KeyCode.Escape */,
                weight: 100 /* KeybindingWeight.EditorContrib */
            }
        });
    }
    run(_accessor, editor) {
        StandaloneColorPickerController.get(editor)?.hide();
    }
}
class InsertColorWithStandaloneColorPicker extends EditorAction {
    constructor() {
        super({
            id: 'editor.action.insertColorWithStandaloneColorPicker',
            label: localizeWithPath('vs/editor/contrib/colorPicker/browser/standaloneColorPickerActions', {
                key: 'insertColorWithStandaloneColorPicker',
                comment: [
                    'Action that inserts color with standalone color picker'
                ]
            }, "Insert Color with Standalone Color Picker"),
            alias: 'Insert Color with Standalone Color Picker',
            precondition: EditorContextKeys.standaloneColorPickerFocused.isEqualTo(true),
            kbOpts: {
                primary: 3 /* KeyCode.Enter */,
                weight: 100 /* KeybindingWeight.EditorContrib */
            }
        });
    }
    run(_accessor, editor) {
        StandaloneColorPickerController.get(editor)?.insertColor();
    }
}
registerEditorAction(HideStandaloneColorPicker);
registerEditorAction(InsertColorWithStandaloneColorPicker);
registerAction2(ShowOrFocusStandaloneColorPicker);
