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
var CopyPasteController_1;
import { addDisposableListener, getActiveDocument } from '../../../../base/browser/dom.js';
import { coalesce } from '../../../../base/common/arrays.js';
import { createCancelablePromise, raceCancellation } from '../../../../base/common/async.js';
import { UriList, createStringDataTransferItem, matchesMimeType } from '../../../../base/common/dataTransfer.js';
import { Disposable } from '../../../../base/common/lifecycle.js';
import { Mimes } from '../../../../base/common/mime.js';
import * as platform from '../../../../base/common/platform.js';
import { generateUuid } from '../../../../base/common/uuid.js';
import { ClipboardEventUtils } from '../../../browser/controller/textAreaInput.js';
import { toExternalVSDataTransfer, toVSDataTransfer } from '../../../browser/dnd.js';
import { IBulkEditService } from '../../../browser/services/bulkEditService.js';
import { Range } from '../../../common/core/range.js';
import { ILanguageFeaturesService } from '../../../common/services/languageFeatures.js';
import { createCombinedWorkspaceEdit, sortEditsByYieldTo } from './edit.js';
import { EditorStateCancellationTokenSource } from '../../editorState/browser/editorState.js';
import { InlineProgressManager } from '../../inlineProgress/browser/inlineProgress.js';
import { localizeWithPath } from '../../../../nls.js';
import { IClipboardService } from '../../../../platform/clipboard/common/clipboardService.js';
import { RawContextKey } from '../../../../platform/contextkey/common/contextkey.js';
import { IInstantiationService } from '../../../../platform/instantiation/common/instantiation.js';
import { IProgressService } from '../../../../platform/progress/common/progress.js';
import { IQuickInputService } from '../../../../platform/quickinput/common/quickInput.js';
import { PostEditWidgetManager } from './postEditWidget.js';
export const changePasteTypeCommandId = 'editor.changePasteType';
export const pasteWidgetVisibleCtx = new RawContextKey('pasteWidgetVisible', false, localizeWithPath('vs/editor/contrib/dropOrPasteInto/browser/copyPasteController', 'pasteWidgetVisible', "Whether the paste widget is showing"));
const vscodeClipboardMime = 'application/vnd.code.copyMetadata';
let CopyPasteController = CopyPasteController_1 = class CopyPasteController extends Disposable {
    static get(editor) {
        return editor.getContribution(CopyPasteController_1.ID);
    }
    constructor(editor, instantiationService, _bulkEditService, _clipboardService, _languageFeaturesService, _quickInputService, _progressService) {
        super();
        this._bulkEditService = _bulkEditService;
        this._clipboardService = _clipboardService;
        this._languageFeaturesService = _languageFeaturesService;
        this._quickInputService = _quickInputService;
        this._progressService = _progressService;
        this._editor = editor;
        const container = editor.getContainerDomNode();
        this._register(addDisposableListener(container, 'copy', e => this.handleCopy(e)));
        this._register(addDisposableListener(container, 'cut', e => this.handleCopy(e)));
        this._register(addDisposableListener(container, 'paste', e => this.handlePaste(e), true));
        this._pasteProgressManager = this._register(new InlineProgressManager('pasteIntoEditor', editor, instantiationService));
        this._postPasteWidgetManager = this._register(instantiationService.createInstance(PostEditWidgetManager, 'pasteIntoEditor', editor, pasteWidgetVisibleCtx, { id: changePasteTypeCommandId, label: localizeWithPath('vs/editor/contrib/dropOrPasteInto/browser/copyPasteController', 'postPasteWidgetTitle', "Show paste options...") }));
    }
    changePasteType() {
        this._postPasteWidgetManager.tryShowSelector();
    }
    pasteAs(preferredId) {
        this._editor.focus();
        try {
            this._pasteAsActionContext = { preferredId };
            getActiveDocument().execCommand('paste');
        }
        finally {
            this._pasteAsActionContext = undefined;
        }
    }
    clearWidgets() {
        this._postPasteWidgetManager.clear();
    }
    isPasteAsEnabled() {
        return this._editor.getOption(84 /* EditorOption.pasteAs */).enabled
            && !this._editor.getOption(90 /* EditorOption.readOnly */);
    }
    handleCopy(e) {
        if (!this._editor.hasTextFocus()) {
            return;
        }
        if (platform.isWeb) {
            // Explicitly clear the web resources clipboard.
            // This is needed because on web, the browser clipboard is faked out using an in-memory store.
            // This means the resources clipboard is not properly updated when copying from the editor.
            this._clipboardService.writeResources([]);
        }
        if (!e.clipboardData || !this.isPasteAsEnabled()) {
            return;
        }
        const model = this._editor.getModel();
        const selections = this._editor.getSelections();
        if (!model || !selections?.length) {
            return;
        }
        const enableEmptySelectionClipboard = this._editor.getOption(37 /* EditorOption.emptySelectionClipboard */);
        let ranges = selections;
        const wasFromEmptySelection = selections.length === 1 && selections[0].isEmpty();
        if (wasFromEmptySelection) {
            if (!enableEmptySelectionClipboard) {
                return;
            }
            ranges = [new Range(ranges[0].startLineNumber, 1, ranges[0].startLineNumber, 1 + model.getLineLength(ranges[0].startLineNumber))];
        }
        const toCopy = this._editor._getViewModel()?.getPlainTextToCopy(selections, enableEmptySelectionClipboard, platform.isWindows);
        const multicursorText = Array.isArray(toCopy) ? toCopy : null;
        const defaultPastePayload = {
            multicursorText,
            pasteOnNewLine: wasFromEmptySelection,
            mode: null
        };
        const providers = this._languageFeaturesService.documentPasteEditProvider
            .ordered(model)
            .filter(x => !!x.prepareDocumentPaste);
        if (!providers.length) {
            this.setCopyMetadata(e.clipboardData, { defaultPastePayload });
            return;
        }
        const dataTransfer = toVSDataTransfer(e.clipboardData);
        const providerCopyMimeTypes = providers.flatMap(x => x.copyMimeTypes ?? []);
        // Save off a handle pointing to data that VS Code maintains.
        const handle = generateUuid();
        this.setCopyMetadata(e.clipboardData, {
            id: handle,
            providerCopyMimeTypes,
            defaultPastePayload
        });
        const promise = createCancelablePromise(async (token) => {
            const results = coalesce(await Promise.all(providers.map(async (provider) => {
                try {
                    return await provider.prepareDocumentPaste(model, ranges, dataTransfer, token);
                }
                catch (err) {
                    console.error(err);
                    return undefined;
                }
            })));
            // Values from higher priority providers should overwrite values from lower priority ones.
            // Reverse the array to so that the calls to `replace` below will do this
            results.reverse();
            for (const result of results) {
                for (const [mime, value] of result) {
                    dataTransfer.replace(mime, value);
                }
            }
            return dataTransfer;
        });
        this._currentCopyOperation?.dataTransferPromise.cancel();
        this._currentCopyOperation = { handle: handle, dataTransferPromise: promise };
    }
    async handlePaste(e) {
        if (!e.clipboardData || !this._editor.hasTextFocus()) {
            return;
        }
        this._currentPasteOperation?.cancel();
        this._currentPasteOperation = undefined;
        const model = this._editor.getModel();
        const selections = this._editor.getSelections();
        if (!selections?.length || !model) {
            return;
        }
        if (!this.isPasteAsEnabled()) {
            return;
        }
        const metadata = this.fetchCopyMetadata(e);
        const dataTransfer = toExternalVSDataTransfer(e.clipboardData);
        dataTransfer.delete(vscodeClipboardMime);
        const allPotentialMimeTypes = [
            ...e.clipboardData.types,
            ...metadata?.providerCopyMimeTypes ?? [],
            // TODO: always adds `uri-list` because this get set if there are resources in the system clipboard.
            // However we can only check the system clipboard async. For this early check, just add it in.
            // We filter providers again once we have the final dataTransfer we will use.
            Mimes.uriList,
        ];
        const allProviders = this._languageFeaturesService.documentPasteEditProvider
            .ordered(model)
            .filter(provider => provider.pasteMimeTypes?.some(type => matchesMimeType(type, allPotentialMimeTypes)));
        if (!allProviders.length) {
            return;
        }
        // Prevent the editor's default paste handler from running.
        // Note that after this point, we are fully responsible for handling paste.
        // If we can't provider a paste for any reason, we need to explicitly delegate pasting back to the editor.
        e.preventDefault();
        e.stopImmediatePropagation();
        if (this._pasteAsActionContext) {
            this.showPasteAsPick(this._pasteAsActionContext.preferredId, allProviders, selections, dataTransfer, metadata);
        }
        else {
            this.doPasteInline(allProviders, selections, dataTransfer, metadata);
        }
    }
    doPasteInline(allProviders, selections, dataTransfer, metadata) {
        const p = createCancelablePromise(async (token) => {
            const editor = this._editor;
            if (!editor.hasModel()) {
                return;
            }
            const model = editor.getModel();
            const tokenSource = new EditorStateCancellationTokenSource(editor, 1 /* CodeEditorStateFlag.Value */ | 2 /* CodeEditorStateFlag.Selection */, undefined, token);
            try {
                await this.mergeInDataFromCopy(dataTransfer, metadata, tokenSource.token);
                if (tokenSource.token.isCancellationRequested) {
                    return;
                }
                // Filter out any providers the don't match the full data transfer we will send them.
                const supportedProviders = allProviders.filter(provider => isSupportedPasteProvider(provider, dataTransfer));
                if (!supportedProviders.length
                    || (supportedProviders.length === 1 && supportedProviders[0].id === 'text') // Only our default text provider is active
                ) {
                    await this.applyDefaultPasteHandler(dataTransfer, metadata, tokenSource.token);
                    return;
                }
                const providerEdits = await this.getPasteEdits(supportedProviders, dataTransfer, model, selections, tokenSource.token);
                if (tokenSource.token.isCancellationRequested) {
                    return;
                }
                // If the only edit returned is a text edit, use the default paste handler
                if (providerEdits.length === 1 && providerEdits[0].providerId === 'text') {
                    await this.applyDefaultPasteHandler(dataTransfer, metadata, tokenSource.token);
                    return;
                }
                if (providerEdits.length) {
                    const canShowWidget = editor.getOption(84 /* EditorOption.pasteAs */).showPasteSelector === 'afterPaste';
                    return this._postPasteWidgetManager.applyEditAndShowIfNeeded(selections, { activeEditIndex: 0, allEdits: providerEdits }, canShowWidget, tokenSource.token);
                }
                await this.applyDefaultPasteHandler(dataTransfer, metadata, tokenSource.token);
            }
            finally {
                tokenSource.dispose();
                if (this._currentPasteOperation === p) {
                    this._currentPasteOperation = undefined;
                }
            }
        });
        this._pasteProgressManager.showWhile(selections[0].getEndPosition(), localizeWithPath('vs/editor/contrib/dropOrPasteInto/browser/copyPasteController', 'pasteIntoEditorProgress', "Running paste handlers. Click to cancel"), p);
        this._currentPasteOperation = p;
    }
    showPasteAsPick(preferredId, allProviders, selections, dataTransfer, metadata) {
        const p = createCancelablePromise(async (token) => {
            const editor = this._editor;
            if (!editor.hasModel()) {
                return;
            }
            const model = editor.getModel();
            const tokenSource = new EditorStateCancellationTokenSource(editor, 1 /* CodeEditorStateFlag.Value */ | 2 /* CodeEditorStateFlag.Selection */, undefined, token);
            try {
                await this.mergeInDataFromCopy(dataTransfer, metadata, tokenSource.token);
                if (tokenSource.token.isCancellationRequested) {
                    return;
                }
                // Filter out any providers the don't match the full data transfer we will send them.
                let supportedProviders = allProviders.filter(provider => isSupportedPasteProvider(provider, dataTransfer));
                if (preferredId) {
                    // We are looking for a specific edit
                    supportedProviders = supportedProviders.filter(edit => edit.id === preferredId);
                }
                const providerEdits = await this.getPasteEdits(supportedProviders, dataTransfer, model, selections, tokenSource.token);
                if (tokenSource.token.isCancellationRequested) {
                    return;
                }
                if (!providerEdits.length) {
                    return;
                }
                let pickedEdit;
                if (preferredId) {
                    pickedEdit = providerEdits.at(0);
                }
                else {
                    const selected = await this._quickInputService.pick(providerEdits.map((edit) => ({
                        label: edit.label,
                        description: edit.providerId,
                        detail: edit.detail,
                        edit,
                    })), {
                        placeHolder: localizeWithPath('vs/editor/contrib/dropOrPasteInto/browser/copyPasteController', 'pasteAsPickerPlaceholder', "Select Paste Action"),
                    });
                    pickedEdit = selected?.edit;
                }
                if (!pickedEdit) {
                    return;
                }
                const combinedWorkspaceEdit = createCombinedWorkspaceEdit(model.uri, selections, pickedEdit);
                await this._bulkEditService.apply(combinedWorkspaceEdit, { editor: this._editor });
            }
            finally {
                tokenSource.dispose();
                if (this._currentPasteOperation === p) {
                    this._currentPasteOperation = undefined;
                }
            }
        });
        this._progressService.withProgress({
            location: 10 /* ProgressLocation.Window */,
            title: localizeWithPath('vs/editor/contrib/dropOrPasteInto/browser/copyPasteController', 'pasteAsProgress', "Running paste handlers"),
        }, () => p);
    }
    setCopyMetadata(dataTransfer, metadata) {
        dataTransfer.setData(vscodeClipboardMime, JSON.stringify(metadata));
    }
    fetchCopyMetadata(e) {
        if (!e.clipboardData) {
            return;
        }
        // Prefer using the clipboard data we saved off
        const rawMetadata = e.clipboardData.getData(vscodeClipboardMime);
        if (rawMetadata) {
            try {
                return JSON.parse(rawMetadata);
            }
            catch {
                return undefined;
            }
        }
        // Otherwise try to extract the generic text editor metadata
        const [_, metadata] = ClipboardEventUtils.getTextData(e.clipboardData);
        if (metadata) {
            return {
                defaultPastePayload: {
                    mode: metadata.mode,
                    multicursorText: metadata.multicursorText ?? null,
                    pasteOnNewLine: !!metadata.isFromEmptySelection,
                },
            };
        }
        return undefined;
    }
    async mergeInDataFromCopy(dataTransfer, metadata, token) {
        if (metadata?.id && this._currentCopyOperation?.handle === metadata.id) {
            const toMergeDataTransfer = await this._currentCopyOperation.dataTransferPromise;
            if (token.isCancellationRequested) {
                return;
            }
            for (const [key, value] of toMergeDataTransfer) {
                dataTransfer.replace(key, value);
            }
        }
        if (!dataTransfer.has(Mimes.uriList)) {
            const resources = await this._clipboardService.readResources();
            if (token.isCancellationRequested) {
                return;
            }
            if (resources.length) {
                dataTransfer.append(Mimes.uriList, createStringDataTransferItem(UriList.create(resources)));
            }
        }
    }
    async getPasteEdits(providers, dataTransfer, model, selections, token) {
        const results = await raceCancellation(Promise.all(providers.map(async (provider) => {
            try {
                const edit = await provider.provideDocumentPasteEdits?.(model, selections, dataTransfer, token);
                if (edit) {
                    return { ...edit, providerId: provider.id };
                }
            }
            catch (err) {
                console.error(err);
            }
            return undefined;
        })), token);
        const edits = coalesce(results ?? []);
        return sortEditsByYieldTo(edits);
    }
    async applyDefaultPasteHandler(dataTransfer, metadata, token) {
        const textDataTransfer = dataTransfer.get(Mimes.text) ?? dataTransfer.get('text');
        if (!textDataTransfer) {
            return;
        }
        const text = await textDataTransfer.asString();
        if (token.isCancellationRequested) {
            return;
        }
        const payload = {
            text,
            pasteOnNewLine: metadata?.defaultPastePayload.pasteOnNewLine ?? false,
            multicursorText: metadata?.defaultPastePayload.multicursorText ?? null,
            mode: null,
        };
        this._editor.trigger('keyboard', "paste" /* Handler.Paste */, payload);
    }
};
CopyPasteController.ID = 'editor.contrib.copyPasteActionController';
CopyPasteController = CopyPasteController_1 = __decorate([
    __param(1, IInstantiationService),
    __param(2, IBulkEditService),
    __param(3, IClipboardService),
    __param(4, ILanguageFeaturesService),
    __param(5, IQuickInputService),
    __param(6, IProgressService)
], CopyPasteController);
export { CopyPasteController };
function isSupportedPasteProvider(provider, dataTransfer) {
    return Boolean(provider.pasteMimeTypes?.some(type => dataTransfer.matches(type)));
}
