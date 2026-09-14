package com.runa.shared.platform

import androidx.lifecycle.ViewModel
import androidx.lifecycle.ViewModelStore

/**
 * iOS 側で view model の寿命を持つ所有者。[ViewModel.clear] は `internal` で Swift から呼べないため
 * [ViewModelStore.clear] を経由する。預けてよいのは Koin で `factory` 束縛の view model だけ
 * （`single` のものを破棄すると他の画面が壊れる）。
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
