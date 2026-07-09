package com.runa.android.ui.screens.diary

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.TopAppBar
import androidx.compose.runtime.Composable
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.TextStyle
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.runa.android.R
import com.runa.android.ui.theme.ShipporiMincho
import com.runa.shared.feature.diary.DiaryListState
import com.runa.shared.feature.diary.DiaryListViewModel
import org.koin.compose.koinInject

/**
 * Diary detail (screen 11) — reading back a record. The body uses the #C8C6CE
 * body colour in Shippori Mincho for calm legibility, with quiet edit/delete
 * affordances. The entry is read from the (singleton) list view model's cache.
 */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun DiaryDetailScreen(
    clientId: String,
    onEdit: (String) -> Unit,
    onDeleted: () -> Unit,
    onBack: () -> Unit,
    viewModel: DiaryListViewModel = koinInject(),
) {
    val state by viewModel.state.collectAsState()
    val entry = (state as? DiaryListState.Content)?.entries?.firstOrNull { it.clientId == clientId }
    var confirmDelete by remember { mutableStateOf(false) }

    Scaffold(
        containerColor = MaterialTheme.colorScheme.background,
        topBar = {
            TopAppBar(
                title = {},
                navigationIcon = {
                    TextButton(onClick = onBack) { Text(stringResource(R.string.action_back)) }
                },
                actions = {
                    TextButton(onClick = { onEdit(clientId) }) { Text(stringResource(R.string.diary_action_edit)) }
                    TextButton(onClick = { confirmDelete = true }) {
                        Text(
                            text = stringResource(R.string.diary_action_delete),
                            color = MaterialTheme.colorScheme.primary,
                        )
                    }
                },
            )
        },
    ) { innerPadding ->
        if (entry == null) return@Scaffold // deleted or not yet loaded

        Column(
            modifier = Modifier
                .fillMaxSize()
                .padding(innerPadding)
                .verticalScroll(rememberScrollState())
                .padding(horizontal = 28.dp, vertical = 12.dp),
            verticalArrangement = Arrangement.spacedBy(20.dp),
        ) {
            Text(
                text = formatDiaryDate(entry.createdAtEpochMs),
                style = MaterialTheme.typography.labelLarge,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
            )
            Text(
                text = entry.bodyText,
                style = TextStyle(fontFamily = ShipporiMincho, fontSize = 18.sp, lineHeight = 32.sp),
                color = MaterialTheme.colorScheme.onSurface,
            )
            DiaryMood.fromValue(entry.mood)?.let { mood ->
                Text(
                    text = stringResource(mood.labelRes),
                    style = MaterialTheme.typography.labelLarge,
                    color = MaterialTheme.colorScheme.secondary,
                )
            }
        }
    }

    if (confirmDelete) {
        AlertDialog(
            onDismissRequest = { confirmDelete = false },
            title = { Text(stringResource(R.string.diary_delete_confirm_title)) },
            text = { Text(stringResource(R.string.diary_delete_confirm_body)) },
            confirmButton = {
                TextButton(onClick = {
                    confirmDelete = false
                    viewModel.delete(clientId)
                    onDeleted()
                }) { Text(stringResource(R.string.action_delete)) }
            },
            dismissButton = {
                TextButton(onClick = { confirmDelete = false }) { Text(stringResource(R.string.action_cancel)) }
            },
        )
    }
}
