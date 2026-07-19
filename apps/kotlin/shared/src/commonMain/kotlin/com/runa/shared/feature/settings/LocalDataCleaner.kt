package com.runa.shared.feature.settings

import com.runa.shared.db.RunaDatabase
import kotlinx.coroutines.CoroutineDispatcher
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext

/**
 * Wipes all locally-persisted user data. Used on account deletion so no diary,
 * gallery or cached "today" content survives after the account is gone. An
 * interface so callers (and tests) do not depend on the concrete database.
 */
interface LocalDataCleaner {
    suspend fun clearAll()
}

/**
 * [LocalDataCleaner] over the SQLDelight [RunaDatabase]. Every user-scoped table is
 * truncated inside one transaction so a partial wipe cannot leave inconsistent
 * local state.
 */
class DefaultLocalDataCleaner(
    private val database: RunaDatabase,
    private val dispatcher: CoroutineDispatcher = Dispatchers.Default,
) : LocalDataCleaner {

    override suspend fun clearAll() = withContext(dispatcher) {
        database.transaction {
            database.diaryQueries.deleteAllEntries()
            database.diaryQueries.deleteAllSyncMeta()
            database.galleryQueries.deleteAllImages()
            database.galleryQueries.deleteAllSyncMeta()
            database.todayQueries.deleteAllQuotes()
            database.todayQueries.deleteAllSongs()
            database.todayQueries.deleteAllHistory()
        }
    }
}
