package com.runa.shared.platform

import androidx.lifecycle.ViewModel
import androidx.lifecycle.ViewModelStore

/**
 * iOS 側で view model の寿命を持つための小さな所有者。Android では ViewModelStore が
 * 自動でやることを、SwiftUI の観測クラスが手で行うための入口。
 *
 * [ViewModel.clear] は lifecycle 側で `internal` のため Swift から直接は呼べない。
 * 公開されている [ViewModelStore.clear] を通せば `onCleared()` と `viewModelScope` の
 * キャンセルが同じ経路で走るので、そちらを使う。
 *
 * 預けてよいのは Koin で `factory` 束縛している view model だけ。`single` 束縛のものは
 * 画面より長生きさせるのが仕様で（テーマ・認証ゲート・プライバシーロック・再生など）、
 * 破棄すると他の画面が壊れる。
 */
class ViewModelOwner {

    private val store = ViewModelStore()

    /** [viewModel] の寿命をこの所有者に預ける（観測クラスの init から呼ぶ）。 */
    fun own(viewModel: ViewModel) {
        store.put(KEY, viewModel)
    }

    /** 預かった view model を破棄する（観測クラスの deinit から呼ぶ）。冪等。 */
    fun dispose() {
        store.clear()
    }

    private companion object {
        const val KEY = "runa.viewmodel"
    }
}
