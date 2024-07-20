/*---------------------------------------------------------------------------------------------
 *  Copyright (c) Microsoft Corporation. All rights reserved.
 *  Licensed under the MIT License. See License.txt in the project root for license information.
 *--------------------------------------------------------------------------------------------*/
import { EditorAction, EditorCommand, registerEditorAction, registerEditorCommand, registerEditorContribution } from '../../../browser/editorExtensions.js';
import { registerEditorFeature } from '../../../common/editorFeatures.js';
import { CopyPasteController, changePasteTypeCommandId, pasteWidgetVisibleCtx } from './copyPasteController.js';
import { DefaultPasteProvidersFeature } from './defaultProviders.js';
import * as nls from '../../../../nls.js';
registerEditorContribution(CopyPasteController.ID, CopyPasteController, 0 /* EditorContributionInstantiation.Eager */); // eager because it listens to events on the container dom node of the editor
registerEditorFeature(DefaultPasteProvidersFeature);
registerEditorCommand(new class extends EditorCommand {
    constructor() {
        super({
            id: changePasteTypeCommandId,
            precondition: pasteWidgetVisibleCtx,
            kbOpts: {
                weight: 100 /* KeybindingWeight.EditorContrib */,
                primary: 2048 /* KeyMod.CtrlCmd */ | 89 /* KeyCode.Period */,
            }
        });
    }
    runEditorCommand(_accessor, editor, _args) {
        return CopyPasteController.get(editor)?.changePasteType();
    }
});
registerEditorAction(class extends EditorAction {
    constructor() {
        super({
            id: 'editor.action.pasteAs',
            label: nls.localizeWithPath('vs/editor/contrib/dropOrPasteInto/browser/copyPasteContribution', 'pasteAs', "Paste As..."),
            alias: 'Paste As...',
            precondition: undefined,
            metadata: {
                description: 'Paste as',
                args: [{
                        name: 'args',
                        schema: {
                            type: 'object',
                            properties: {
                                'id': {
                                    type: 'string',
                                    description: nls.localizeWithPath('vs/editor/contrib/dropOrPasteInto/browser/copyPasteContribution', 'pasteAs.id', "The id of the paste edit to try applying. If not provided, the editor will show a picker."),
                                }
                            },
                        }
                    }]
            }
        });
    }
    run(_accessor, editor, args) {
        const id = typeof args?.id === 'string' ? args.id : undefined;
        return CopyPasteController.get(editor)?.pasteAs(id);
    }
});
