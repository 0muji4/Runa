package com.runa.shared.feature.today

import com.runa.shared.core.state.AppError
import com.runa.shared.core.state.SyncPhase
import com.runa.shared.core.state.UiState
import com.runa.shared.feature.today.moon.MoonPhase
import com.runa.shared.feature.today.moon.MoonPhaseKey
import com.runa.shared.network.ApiException
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.test.StandardTestDispatcher
import kotlinx.coroutines.test.advanceUntilIdle
import kotlinx.coroutines.test.resetMain
import kotlinx.coroutines.test.runTest
import kotlinx.coroutines.test.setMain
import kotlinx.datetime.LocalDate
import kotlinx.datetime.TimeZone
import kotlin.test.AfterTest
import kotlin.test.BeforeTest
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertIs

/** ホームは「ネットワーク優先＋失敗時キャッシュ」なので、リポジトリの戻り方を差し替える。 */
private class StubTodayRepository(
    var result: () -> Today,
) : TodayRepository {
    override suspend fun getToday(localDate: LocalDate, zone: TimeZone): Today = result()
}

private fun today(isOffline: Boolean) = Today(
    dateLabel = "9月1日",
    quote = null,
    song = null,
    moon = MoonPhase(MoonPhaseKey.FULL_MOON, illumination = 1.0, ageDays = 14.8),
    isOffline = isOffline,
)

/**
 * ホームの状態の出し分けを固定する。オフラインは本文を隠す [UiState.Failure] ではなく
 * [UiState.Content] に [SyncPhase.Offline] を載せる、というのがこのアプリの約束。
 */
class HomeViewModelTest {

    @BeforeTest
    fun setUpMain() = Dispatchers.setMain(StandardTestDispatcher())

    @AfterTest
    fun tearDownMain() = Dispatchers.resetMain()

    @Test
    fun onlineResultBecomesContentWithIdleSync() = runTest {
        val vm = HomeViewModel(StubTodayRepository { today(isOffline = false) })
        advanceUntilIdle()

        val content = assertIs<UiState.Content<Today>>(vm.state.value)
        assertEquals(SyncPhase.Idle, content.sync)
        assertEquals("9月1日", content.data.dateLabel)
    }

    @Test
    fun offlineKeepsContentAndOnlyFlagsTheSyncPhase() = runTest {
        val vm = HomeViewModel(StubTodayRepository { today(isOffline = true) })
        advanceUntilIdle()

        val content = assertIs<UiState.Content<Today>>(vm.state.value)
        assertEquals(SyncPhase.Offline, content.sync, "オフラインでも本文は隠さず帯だけ切り替える")
    }

    @Test
    fun repositoryFailureBecomesClassifiedFailure() = runTest {
        val vm = HomeViewModel(StubTodayRepository { throw ApiException(500, null, "boom") })
        advanceUntilIdle()

        val failure = assertIs<UiState.Failure>(vm.state.value)
        assertIs<AppError.Server>(failure.error)
    }
}
