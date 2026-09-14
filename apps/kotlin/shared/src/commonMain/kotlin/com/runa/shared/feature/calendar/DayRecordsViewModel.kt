package com.runa.shared.feature.calendar

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.runa.shared.feature.diary.DiaryEntry
import kotlinx.coroutines.flow.SharingStarted
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.stateIn
import kotlinx.datetime.LocalDate
import kotlinx.datetime.TimeZone

/** Streams one day's diary entries; [isoDate] is `yyyy-MM-dd`. */
class DayRecordsViewModel(
    repository: CalendarRepository,
    isoDate: String,
    zone: TimeZone = TimeZone.currentSystemDefault(),
) : ViewModel() {
    private val date = LocalDate.parse(isoDate)

    /** Pre-formatted header label, e.g. "7月4日". */
    val dateLabel: String = "${date.monthNumber}月${date.dayOfMonth}日"

    val state: StateFlow<List<DiaryEntry>> =
        repository.observeEntriesOn(date.year, date.monthNumber, date.dayOfMonth, zone)
            .stateIn(viewModelScope, SharingStarted.WhileSubscribed(5_000L), emptyList())
}
