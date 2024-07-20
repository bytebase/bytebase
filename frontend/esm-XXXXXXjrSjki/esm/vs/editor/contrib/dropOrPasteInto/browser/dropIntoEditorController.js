/*---------------------------------------------------------------------------------------------
 *  Copyright (c) Microsoft Corporation. All rights reserved.
 *  Licensed under the MIT License. See License.txt in the project root for license information.
 *--------------------------------------------------------------------------------------------*/
var __decorate = (this && this.__decorate) || function (decorators, target, key, desc) {
    var c = arguments.length, r = c < 3 ? target : desc === null ? desc = Object.getOwnPropertyDescriptor(target, key) : desc, d;
    if (typeof Reflect === "object" && typeof Reflect.decorate === "function") r = Reflect.decorate(decorators, target, key, desc);
    else for (var i = decorators.length - 1; i >= 0; i--) if (d = decorators[i]) r = (c < 3 ? d(r) : c > 3 ? d(target, key, r) : d(target, key)) || r;
    return c > 3 && r && Object.defineProperty(target, key, r), r;
};
var __param = (this && this.__param) || function (paramIndex, decorator) {
    return function (target, key) { decorator(target, key, paramIndex); }
};
var DropIntoEditorController_1;
import { coalesce } from '../../../../base/common/arrays.js';
import { createCancelablePromise, raceCancellation } from '../../../../base/common/async.js';
import { VSDataTransfer, matchesMimeType } from '../../../../base/common/dataTransfer.js';
import { Disposable } from '../../../../base/common/lifecycle.js';
import { toExternalVSDataTransfer } from '../../../browser/dnd.js';
import { Range } from '../../../common/core/range.js';
import { ILanguageFeaturesService } from '../../../common/services/languageFeatures.js';
import { DraggedTreeItemsIdentifier } from '../../../common/services/treeViewsDnd.js';
import { ITreeViewsDnDService } from '../../../common/services/treeViewsDndService.js';
import { EditorStateCancellationTokenSource } from '../../editorState/browser/editorState.js';
import { InlineProgressManager } from '../../inlineProgress/browser/inlineProgress.js';
import { localizeWithPath } from '../../../../nls.js';
import { IConfigurationService } from '../../../../platform/configuration/common/configuration.js';
import { RawContextKey } from '../../../../platform/contextkey/common/contextkey.js';
import { LocalSelectionTransfer } from '../../../../platform/dnd/browser/dnd.js';
import { IInstantiationService } from '../../../../platform/instantiation/common/instantiation.js';
import { sortEditsByYieldTo } from './edit.js';
import { PostEditWidgetManager } from './postEditWidget.js';
export const defaultProviderConfig = 'editor.experimental.dropIntoEditor.defaultProvider';
export const changeDropTypeCommandId = 'editor.changeDropType';
export const dropWidgetVisibleCtx = new RawContextKey('dropWidgetVisible', false, localizeWithPath('vs/editor/contrib/dropOrPasteInto/browser/dropIntoEditorController', 'dropWidgetVisible', "Whether the drop widget is showing"));
let DropIntoEditorController = DropIntoEditorController_1 = class DropIntoEditorController extends Disposable {
    static get(editor) {
        return editor.getContribution(DropIntoEditorController_1.ID);
    }
    constructor(editor, instantiationService, _configService, _languageFeaturesService, _treeViewsDragAndDropService) {
        super();
        this._configService = _configService;
        this._languageFeaturesService = _languageFeaturesService;
        this._treeViewsDragAndDropService = _treeViewsDragAndDropService;
        this.treeItemsTransfer = LocalSelectionTransfer.getInstance();
        this._dropProgressManager = this._register(instantiationService.createInstance(InlineProgressManager, 'dropIntoEditor', editor));
        this._postDropWidgetManager = this._register(instantiationService.createInstance(PostEditWidgetManager, 'dropIntoEditor', editor, dropWidgetVisibleCtx, { id: changeDropTypeCommandId, label: localizeWithPath('vs/editor/contrib/dropOrPasteInto/browser/dropIntoEditorController', 'postDropWidgetTitle', "Show drop options...") }));
        this._register(editor.onDropIntoEditor(e => this.onDropIntoEditor(editor, e.position, e.event)));
    }
    clearWidgets() {
        this._postDropWidgetManager.clear();
    }
    changeDropType() {
        this._postDropWidgetManager.tryShowSelector();
    }
    async onDropIntoEditor(editor, position, dragEvent) {
        if (!dragEvent.dataTransfer || !editor.hasModel()) {
            return;
        }
        this._currentOperation?.cancel();
        editor.focus();
        editor.setPosition(position);
        const p = createCancelablePromise(async (token) => {
            const tokenSource = new EditorStateCancellationTokenSource(editor, 1 /* CodeEditorStateFlag.Value */, undefined, token);
            try {
                const ourDataTransfer = await this.extractDataTransferData(dragEvent);
                if (ourDataTransfer.size === 0 || tokenSource.token.isCancellationRequested) {
                    return;
                }
                const model = editor.getModel();
                if (!model) {
                    return;
                }
                const providers = this._languageFeaturesService.documentOnDropEditProvider
                    .ordered(model)
                    .filter(provider => {
                    if (!provider.dropMimeTypes) {
                        // Keep all providers that don't specify mime types
                        return true;
                    }
                    return provider.dropMimeTypes.some(mime => ourDataTransfer.matches(mime));
                });
                const edits = await this.getDropEdits(providers, model, position, ourDataTransfer, tokenSource);
                if (tokenSource.token.isCancellationRequested) {
                    return;
                }
                if (edits.length) {
                    const activeEditIndex = this.getInitialActiveEditIndex(model, edits);
                    const canShowWidget = editor.getOption(36 /* EditorOption.dropIntoEditor */).showDropSelector === 'afterDrop';
                    // Pass in the parent token here as it tracks cancelling the entire drop operation
                    await this._postDropWidgetManager.applyEditAndShowIfNeeded([Range.fromPositions(position)], { activeEditIndex, allEdits: edits }, canShowWidget, token);
                }
            }
            finally {
                tokenSource.dispose();
                if (this._currentOperation === p) {
                    this._currentOperation = undefined;
                }
            }
        });
        this._dropProgressManager.showWhile(position, localizeWithPath('vs/editor/contrib/dropOrPasteInto/browser/dropIntoEditorController', 'dropIntoEditorProgress', "Running drop handlers. Click to cancel"), p);
        this._currentOperation = p;
    }
    async getDropEdits(providers, model, position, dataTransfer, tokenSource) {
        const results = await raceCancellation(Promise.all(providers.map(async (provider) => {
            try {
                const edit = await provider.provideDocumentOnDropEdits(model, position, dataTransfer, tokenSource.token);
                if (edit) {
                    return { ...edit, providerId: provider.id };
                }
            }
            catch (err) {
                console.error(err);
            }
            return undefined;
        })), tokenSource.token);
        const edits = coalesce(results ?? []);
        return sortEditsByYieldTo(edits);
    }
    getInitialActiveEditIndex(model, edits) {
        const preferredProviders = this._configService.getValue(defaultProviderConfig, { resource: model.uri });
        for (const [configMime, desiredId] of Object.entries(preferredProviders)) {
            const editIndex = edits.findIndex(edit => desiredId === edit.providerId
                && edit.handledMimeType && matchesMimeType(configMime, [edit.handledMimeType]));
            if (editIndex >= 0) {
                return editIndex;
            }
        }
        return 0;
    }
    async extractDataTransferData(dragEvent) {
        if (!dragEvent.dataTransfer) {
            return new VSDataTransfer();
        }
        const dataTransfer = toExternalVSDataTransfer(dragEvent.dataTransfer);
        if (this.treeItemsTransfer.hasData(DraggedTreeItemsIdentifier.prototype)) {
            const data = this.treeItemsTransfer.getData(DraggedTreeItemsIdentifier.prototype);
            if (Array.isArray(data)) {
                for (const id of data) {
                    const treeDataTransfer = await this._treeViewsDragAndDropService.removeDragOperationTransfer(id.identifier);
                    if (treeDataTransfer) {
                        for (const [type, value] of treeDataTransfer) {
                            dataTransfer.replace(type, value);
                        }
                    }
                }
            }
        }
        return dataTransfer;
    }
};
DropIntoEditorController.ID = 'editor.contrib.dropIntoEditorController';
DropIntoEditorController = DropIntoEditorController_1 = __decorate([
    __param(1, IInstantiationService),
    __param(2, IConfigurationService),
    __param(3, ILanguageFeaturesService),
    __param(4, ITreeViewsDnDService)
], DropIntoEditorController);
export { DropIntoEditorController };
