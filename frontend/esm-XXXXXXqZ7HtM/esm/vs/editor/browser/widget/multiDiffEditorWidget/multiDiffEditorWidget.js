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
import { Disposable } from '../../../../base/common/lifecycle.js';
import { derived, derivedWithStore, observableValue, recomputeInitiallyAndOnChange } from '../../../../base/common/observable.js';
import { readHotReloadableExport } from '../diffEditor/utils.js';
import { MultiDiffEditorWidgetImpl } from './multiDiffEditorWidgetImpl.js';
import { MultiDiffEditorViewModel } from './multiDiffEditorViewModel.js';
import { IInstantiationService } from '../../../../platform/instantiation/common/instantiation.js';
import './colors.js';
import { DiffEditorItemTemplate } from './diffEditorItemTemplate.js';
import { Event } from '../../../../base/common/event.js';
let MultiDiffEditorWidget = class MultiDiffEditorWidget extends Disposable {
    constructor(_element, _workbenchUIElementFactory, _instantiationService) {
        super();
        this._element = _element;
        this._workbenchUIElementFactory = _workbenchUIElementFactory;
        this._instantiationService = _instantiationService;
        this._dimension = observableValue(this, undefined);
        this._viewModel = observableValue(this, undefined);
        this._widgetImpl = derivedWithStore(this, (reader, store) => {
            readHotReloadableExport(DiffEditorItemTemplate, reader);
            return store.add(this._instantiationService.createInstance((readHotReloadableExport(MultiDiffEditorWidgetImpl, reader)), this._element, this._dimension, this._viewModel, this._workbenchUIElementFactory));
        });
        this._activeControl = derived(this, (reader) => this._widgetImpl.read(reader).activeControl.read(reader));
        this.onDidChangeActiveControl = Event.fromObservableLight(this._activeControl);
        this._scrollState = derived(this, (reader) => {
            const w = this._widgetImpl.read(reader);
            const top = w.scrollTop.read(reader);
            const left = w.scrollLeft.read(reader);
            return { top, left };
        });
        this.onDidChangeScrollState = Event.fromObservableLight(this._scrollState);
        this._register(recomputeInitiallyAndOnChange(this._widgetImpl));
    }
    createViewModel(model) {
        return new MultiDiffEditorViewModel(model, this._instantiationService);
    }
    setViewModel(viewModel) {
        this._viewModel.set(viewModel, undefined);
    }
    layout(dimension) {
        this._dimension.set(dimension, undefined);
    }
    getActiveControl() {
        return this._activeControl.get();
    }
    getScrollState() {
        return this._scrollState.get();
    }
    setScrollState(scrollState) {
        const w = this._widgetImpl.get();
        w.setScrollState(scrollState);
    }
};
MultiDiffEditorWidget = __decorate([
    __param(2, IInstantiationService)
], MultiDiffEditorWidget);
export { MultiDiffEditorWidget };
