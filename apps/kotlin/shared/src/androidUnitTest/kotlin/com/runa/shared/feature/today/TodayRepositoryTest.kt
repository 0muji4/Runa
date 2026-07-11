package com.runa.shared.feature.today

import com.runa.shared.feature.today.moon.MoonPhaseKey
import io.ktor.client.engine.mock.MockEngine
import io.ktor.client.engine.mock.respondError
import io.ktor.http.HttpStatusCode
import kotlinx.coroutines.test.runTest
import kotlinx.datetime.LocalDate
import kotlinx.datetime.TimeZone
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertFalse
import kotlin.test.assertNotNull
import kotlin.test.assertNull
import kotlin.test.assertTrue

class TodayRepositoryTest {

    private val date = LocalDate.parse("2024-12-15")

    @Test
    fun composesQuoteSongAndMoonWhenOnline() = runTest {
        val repo = DefaultTodayRepository(mockApiClient(jsonEngine { todayJson }), inMemoryDatabase())

        val today = repo.getToday(date, TimeZone.UTC)

        assertFalse(today.isOffline)
        assertEquals("月あかり", today.quote?.bodyText)
        assertEquals("夜想曲", today.song?.title)
        // 2024-12-15 is a full moon — proves the moon is composed in, not fetched.
        assertEquals(MoonPhaseKey.FULL_MOON, today.moon.phaseKey)
        assertTrue(today.moon.illumination > 0.95)
    }

    @Test
    fun fallsBackToCachedQuoteAndComputesMoonWhenOffline() = runTest {
        val db = inMemoryDatabase()

        // First, an online fetch populates the cache.
        DefaultTodayRepository(mockApiClient(jsonEngine { todayJson }), db).getToday(date, TimeZone.UTC)

        // Then a failing backend must fall back to the cache, moon still computed.
        val failing = MockEngine { respondError(HttpStatusCode.InternalServerError) }
        val offline = DefaultTodayRepository(mockApiClient(failing), db).getToday(date, TimeZone.UTC)

        assertTrue(offline.isOffline)
        assertEquals("月あかり", offline.quote?.bodyText)
        assertEquals("夜想曲", offline.song?.title)
        assertEquals(MoonPhaseKey.FULL_MOON, offline.moon.phaseKey)
    }

    @Test
    fun offlineWithNoCacheStillReturnsMoon() = runTest {
        val failing = MockEngine { respondError(HttpStatusCode.ServiceUnavailable) }
        val repo = DefaultTodayRepository(mockApiClient(failing), inMemoryDatabase())

        val today = repo.getToday(date, TimeZone.UTC)

        assertTrue(today.isOffline)
        assertNull(today.quote)
        assertNull(today.song)
        assertNotNull(today.moon)
        assertEquals(MoonPhaseKey.FULL_MOON, today.moon.phaseKey)
    }
}
