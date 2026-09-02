package com.runa.shared.feature.insight

import com.runa.shared.core.state.SyncPhase
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.emptyFlow
import kotlinx.coroutines.flow.launchIn
import kotlinx.coroutines.test.StandardTestDispatcher
import kotlinx.coroutines.test.advanceUntilIdle
import kotlinx.coroutines.test.resetMain
import kotlinx.coroutines.test.runTest
import kotlinx.coroutines.test.setMain
import kotlinx.datetime.Clock
import kotlinx.datetime.DayOfWeek
import kotlinx.datetime.Instant
import kotlinx.datetime.TimeZone
import kotlin.test.AfterTest
import kotlin.test.BeforeTest
import kotlin.test.Test
import kotlin.test.assertEquals

private class InsightTestClock(private val instant: Instant) : Clock {
    override fun now(): Instant = instant
}

private class FakeInsightRepository : InsightRepository {
    var refreshCount = 0
    override fun observeInsight(period: InsightPeriod, zone: TimeZone): Flow<Insight> = emptyFlow()
    override suspend fun refresh(): Result<Unit> {
        refreshCount += 1
        return Result.success(Unit)
    }
    override val syncStatus: StateFlow<SyncPhase> = MutableStateFlow(SyncPhase.Idle)
}

/**
 * 期間セレクタ（週/月）と見出しラベルの動きを固定する。ラベルは本文が空でも常に出す
 * ため [InsightViewModel.header] が [InsightViewModel.state] と別 Flow になっている。
 */
class InsightViewModelTest {

    @BeforeTest
    fun setUpMain() = Dispatchers.setMain(StandardTestDispatcher())

    @AfterTest
    fun tearDownMain() = Dispatchers.resetMain()

    private val utc = TimeZone.UTC

    // 2026-09-15 は火曜。週起点が日曜なら 9/13〜9/19 の週に入る。
    private fun viewModel(repo: InsightRepository = FakeInsightRepository()) = InsightViewModel(
        repository = repo,
        zone = utc,
        weekStart = DayOfWeek.SUNDAY,
        clock = InsightTestClock(Instant.parse("2026-09-15T12:00:00Z")),
    )

    @Test
    fun opensOnTheMonthContainingToday() = runTest {
        val vm = viewModel()
        val job = vm.header.launchIn(this)
        advanceUntilIdle()

        assertEquals(InsightPeriodType.Monthly, vm.header.value.periodType)
        assertEquals("9月のうつろい", vm.header.value.periodLabel)
        job.cancel()
    }

    @Test
    fun switchingToWeeklyAnchorsOnTheWeekContainingToday() = runTest {
        val vm = viewModel()
        val job = vm.header.launchIn(this)
        advanceUntilIdle()

        vm.setPeriodType(InsightPeriodType.Weekly)
        advanceUntilIdle()

        assertEquals(InsightPeriodType.Weekly, vm.header.value.periodType)
        assertEquals("9月13日〜19日のうつろい", vm.header.value.periodLabel)
        job.cancel()
    }

    @Test
    fun tappingTheAlreadySelectedTypeDoesNothing() = runTest {
        val repo = FakeInsightRepository()
        val vm = viewModel(repo)
        val job = vm.header.launchIn(this)
        advanceUntilIdle()
        val refreshesAfterInit = repo.refreshCount

        vm.setPeriodType(InsightPeriodType.Monthly)
        advanceUntilIdle()

        assertEquals(refreshesAfterInit, repo.refreshCount, "同じ種別の再タップで期間を作り直さない")
        job.cancel()
    }

    @Test
    fun movingBackAndReturningToNowKeepsTheMode() = runTest {
        val vm = viewModel()
        val job = vm.header.launchIn(this)
        advanceUntilIdle()

        vm.showPrevious()
        advanceUntilIdle()
        assertEquals("8月のうつろい", vm.header.value.periodLabel)

        vm.showCurrent()
        advanceUntilIdle()
        assertEquals("9月のうつろい", vm.header.value.periodLabel)
        assertEquals(InsightPeriodType.Monthly, vm.header.value.periodType)
        job.cancel()
    }
}
