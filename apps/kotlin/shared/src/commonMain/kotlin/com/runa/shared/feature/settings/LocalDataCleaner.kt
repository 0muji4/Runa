package com.runa.shared.feature.settings

import com.runa.shared.db.RunaDatabase
import kotlinx.coroutines.CoroutineDispatcher
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext

/** Wipes all locally-persisted user data on account deletion. */
interface LocalDataCleaner {
    suspend fun clearAll()
}

/** [LocalDataCleaner] over the SQLDelight [RunaDatabase]; one transaction so a partial wipe cannot occur. */
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
