package com.runa.shared.feature.calendar

import com.runa.shared.core.state.SyncPhase
import com.runa.shared.core.state.UiState
import com.runa.shared.feature.diary.DiaryEntry
import com.runa.shared.feature.today.moon.MoonPhaseKey
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.flowOf
import kotlinx.coroutines.flow.launchIn
import kotlinx.coroutines.test.StandardTestDispatcher
import kotlinx.coroutines.test.advanceUntilIdle
import kotlinx.coroutines.test.resetMain
import kotlinx.coroutines.test.runTest
import kotlinx.coroutines.test.setMain
import kotlinx.datetime.Clock
import kotlinx.datetime.Instant
import kotlinx.datetime.TimeZone
import kotlin.test.AfterTest
import kotlin.test.BeforeTest
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertIs

private class CalendarTestClock(private val instant: Instant) : Clock {
    override fun now(): Instant = instant
}

private class FakeCalendarRepository : CalendarRepository {
    /** observeMonth に渡ってきた (year, month) を記録して、そのまま 1 日分を返す。 */
    val requested = mutableListOf<Pair<Int, Int>>()

    override fun observeMonth(year: Int, month: Int, zone: TimeZone): Flow<List<CalendarDay>> {
        requested += year to month
        return flowOf(
            listOf(
                CalendarDay(
                    year = year,
                    month = month,
                    day = 1,
                    phaseKey = MoonPhaseKey.NEW_MOON,
                    illumination = 0.0,
                    entryCount = 1,
                    isToday = false,
                ),
            ),
        )
    }

    override fun observeEntriesOn(year: Int, month: Int, day: Int, zone: TimeZone): Flow<List<DiaryEntry>> =
        flowOf(emptyList())

    override suspend fun refresh(year: Int, month: Int, zone: TimeZone): Result<Unit> = Result.success(Unit)

    override val syncStatus: StateFlow<SyncPhase> = MutableStateFlow(SyncPhase.Idle)
}

/** 表示中の月の決まり方と、前後移動・今日へ戻るの動きを固定する。 */
class CalendarViewModelTest {

    @BeforeTest
    fun setUpMain() = Dispatchers.setMain(StandardTestDispatcher())

    @AfterTest
    fun tearDownMain() = Dispatchers.resetMain()

    private val utc = TimeZone.UTC
    private val clock = CalendarTestClock(Instant.parse("2026-09-15T12:00:00Z"))

    @Test
    fun opensOnTheMonthContainingToday() = runTest {
        val vm = CalendarViewModel(FakeCalendarRepository(), zone = utc, clock = clock)
        val job = vm.state.launchIn(this)
        advanceUntilIdle()

        val month = assertIs<UiState.Content<CalendarMonth>>(vm.state.value).data
        assertEquals(2026, month.year)
        assertEquals(9, month.month)
        job.cancel()
    }

    @Test
    fun movingBackAndForwardCrossesTheYearBoundary() = runTest {
        val repo = FakeCalendarRepository()
        val vm = CalendarViewModel(repo, zone = utc, clock = clock)
        val job = vm.state.launchIn(this)
        advanceUntilIdle()

        repeat(4) { vm.showPreviousMonth() }
        advanceUntilIdle()
        var month = assertIs<UiState.Content<CalendarMonth>>(vm.state.value).data
        assertEquals(2026 to 5, month.year to month.month)

        repeat(8) { vm.showNextMonth() }
        advanceUntilIdle()
        month = assertIs<UiState.Content<CalendarMonth>>(vm.state.value).data
        assertEquals(2027 to 1, month.year to month.month, "12 月の次は翌年 1 月")

        vm.showToday()
        advanceUntilIdle()
        month = assertIs<UiState.Content<CalendarMonth>>(vm.state.value).data
        assertEquals(2026 to 9, month.year to month.month)
        job.cancel()
    }
}
