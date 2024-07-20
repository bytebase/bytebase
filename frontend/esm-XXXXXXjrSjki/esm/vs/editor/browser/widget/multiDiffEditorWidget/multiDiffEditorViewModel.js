/*---------------------------------------------------------------------------------------------
 *  Copyright (c) Microsoft Corporation. All rights reserved.
 *  Licensed under the MIT License. See License.txt in the project root for license information.
 *--------------------------------------------------------------------------------------------*/
import { Disposable } from '../../../../base/common/lifecycle.js';
import { derivedWithStore, observableFromEvent, observableValue, transaction } from '../../../../base/common/observable.js';
import { DiffEditorOptions } from '../diffEditor/diffEditorOptions.js';
import { DiffEditorViewModel } from '../diffEditor/diffEditorViewModel.js';
export class MultiDiffEditorViewModel extends Disposable {
    async waitForDiffs() {
        for (const d of this.items.get()) {
            await d.diffEditorViewModel.waitForDiff();
        }
    }
    collapseAll() {
        transaction(tx => {
            for (const d of this.items.get()) {
                d.collapsed.set(true, tx);
            }
        });
    }
    expandAll() {
        transaction(tx => {
            for (const d of this.items.get()) {
                d.collapsed.set(false, tx);
            }
        });
    }
    constructor(_model, _instantiationService) {
        super();
        this._model = _model;
        this._instantiationService = _instantiationService;
        this._documents = observableFromEvent(this._model.onDidChange, /** @description MultiDiffEditorViewModel.documents */ () => this._model.documents);
        this.items = derivedWithStore(this, (reader, store) => this._documents.read(reader).map(d => store.add(new DocumentDiffItemViewModel(d, this._instantiationService)))).recomputeInitiallyAndOnChange(this._store);
        this.activeDiffItem = observableValue(this, undefined);
    }
}
export class DocumentDiffItemViewModel extends Disposable {
    constructor(entry, _instantiationService) {
        super();
        this.entry = entry;
        this._instantiationService = _instantiationService;
        this.collapsed = observableValue(this, false);
        function updateOptions(options) {
            return {
                ...options,
                hideUnchangedRegions: {
                    enabled: true,
                },
            };
        }
        const options = new DiffEditorOptions(updateOptions(this.entry.value.options || {}));
        if (this.entry.value.onOptionsDidChange) {
            this._register(this.entry.value.onOptionsDidChange(() => {
                options.updateOptions(updateOptions(this.entry.value.options || {}));
            }));
        }
        this.diffEditorViewModel = this._register(this._instantiationService.createInstance(DiffEditorViewModel, {
            original: entry.value.original,
            modified: entry.value.modified,
        }, options));
    }
}
