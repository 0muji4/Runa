package com.runa.android.ui.screens.diary

import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.TextStyle
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import com.runa.android.R
import com.runa.android.ui.components.MoonPhaseDisc
import com.runa.android.ui.components.RunaScreenHeader
import com.runa.android.ui.theme.RunaColors
import com.runa.android.ui.theme.ShipporiMincho
import com.runa.shared.core.state.UiState
import com.runa.shared.feature.diary.DiaryEntry
import com.runa.shared.feature.diary.DiaryListViewModel
import org.koin.compose.koinInject

/**
 * Diary detail (11) — reading a record back. A moon-led header (phase disc, date,
 * phase · weekday) over the body in #C8C6CE 明朝 for calm legibility, with quiet
 * edit/delete affordances. The entry is read from the (singleton) list cache.
 */
@Composable
fun DiaryDetailScreen(
    clientId: String,
    onEdit: (String) -> Unit,
    onDeleted: () -> Unit,
    onBack: () -> Unit,
    viewModel: DiaryListViewModel = koinInject(),
) {
    val state by viewModel.state.collectAsStateWithLifecycle()
    val entry = (state as? UiState.Content<List<DiaryEntry>>)?.data?.firstOrNull { it.clientId == clientId }
    var confirmDelete by remember { mutableStateOf(false) }

    Surface(color = RunaColors.Background) {
        Column(
            modifier = Modifier
                .fillMaxSize()
                .verticalScroll(rememberScrollState())
                .padding(start = 28.dp, end = 28.dp, bottom = 18.dp),
        ) {
            // The date is the screen title; the moon and weekday sit under it as meta.
            RunaScreenHeader(
                title = entry?.let { formatDiaryDate(it.createdAtEpochMs) },
                onBack = onBack,
            ) {
                Text(
                    text = stringResource(R.string.diary_action_edit),
                    style = MaterialTheme.typography.labelMedium,
                    color = RunaColors.Subtle,
                    modifier = Modifier
                        .clickable { onEdit(clientId) }
                        .padding(8.dp),
                )
                Text(
                    text = stringResource(R.string.diary_action_delete),
                    style = MaterialTheme.typography.labelMedium,
                    color = RunaColors.Accent,
                    modifier = Modifier
                        .clickable { confirmDelete = true }
                        .padding(8.dp),
                )
            }

            if (entry != null) {
                val moon = remember(entry.createdAtEpochMs) { diaryMoonFor(entry.createdAtEpochMs) }
                Row(
                    verticalAlignment = Alignment.CenterVertically,
                    horizontalArrangement = Arrangement.spacedBy(14.dp),
                ) {
                    MoonPhaseDisc(illumination = moon.illumination, waxing = moon.waxing, diameter = 44.dp)
                    Text(
                        text = "${moon.name}　・　${formatDiaryWeekday(entry.createdAtEpochMs)}",
                        style = MaterialTheme.typography.labelMedium,
                        color = RunaColors.Subtle,
                    )
                }
                Spacer(Modifier.height(28.dp))
                Text(
                    text = entry.bodyText,
                    style = TextStyle(fontFamily = ShipporiMincho, fontSize = 18.sp, lineHeight = 32.sp),
                    color = RunaColors.Body,
                    modifier = Modifier.fillMaxWidth(),
                )
            }
        }
    }

    if (confirmDelete) {
        AlertDialog(
            containerColor = RunaColors.Surface,
            onDismissRequest = { confirmDelete = false },
            title = { Text(stringResource(R.string.diary_delete_confirm_title), color = RunaColors.Heading) },
            text = { Text(stringResource(R.string.diary_delete_confirm_body), color = RunaColors.Subtle) },
            confirmButton = {
                TextButton(onClick = {
                    confirmDelete = false
                    viewModel.delete(clientId)
                    onDeleted()
                }) { Text(stringResource(R.string.action_delete), color = RunaColors.Accent) }
            },
            dismissButton = {
                TextButton(onClick = { confirmDelete = false }) {
                    Text(stringResource(R.string.action_cancel), color = RunaColors.Subtle)
                }
            },
        )
    }
}
