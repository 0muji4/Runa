package com.runa.android.ui.screens.diary

import androidx.activity.compose.BackHandler
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.imePadding
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.BasicTextField
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.remember
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.SolidColor
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.TextStyle
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.runa.android.R
import com.runa.android.ui.theme.RunaColors
import com.runa.android.ui.theme.ShipporiMincho
import com.runa.shared.feature.diary.DiaryEditorViewModel
import com.runa.shared.feature.diary.SaveStatus
import org.koin.compose.getKoin
import org.koin.core.parameter.parametersOf

/**
 * Diary editor (10) — "書く". A whitespace-first 明朝 canvas: the day's date, a
 * quiet prompt, then the writing surface. The character count is never shown and
 * autosave is durable (the entry persists from the first line) with a whisper of
 * an indicator. "とじる" flushes and leaves. Mood is intentionally absent, matching
 * the confirmed design.
 */
@Composable
fun DiaryEditorScreen(
    clientId: String?,
    onDone: () -> Unit,
) {
    val koin = getKoin()
    val viewModel = remember(clientId) {
        koin.get<DiaryEditorViewModel> { parametersOf(clientId) }
    }
    val state by viewModel.state.collectAsState()
    // No created-at in the editor state; the header shows the day being written.
    val dayMs = remember { System.currentTimeMillis() }

    val leave = {
        viewModel.saveNow()
        onDone()
    }
    BackHandler(onBack = leave)

    Surface(color = RunaColors.Background) {
        Column(
            modifier = Modifier
                .fillMaxSize()
                .imePadding()
                .padding(horizontal = 28.dp, vertical = 20.dp),
        ) {
            Text(
                text = "${formatDiaryDate(dayMs)}　${formatDiaryWeekday(dayMs)}",
                style = MaterialTheme.typography.labelLarge,
                color = RunaColors.Subtle,
            )
            Text(
                text = stringResource(R.string.diary_editor_prompt),
                style = TextStyle(fontFamily = ShipporiMincho, fontSize = 20.sp, lineHeight = 30.sp),
                color = RunaColors.Subtle,
                modifier = Modifier.padding(top = 20.dp, bottom = 16.dp),
            )

            BasicTextField(
                value = state.bodyText,
                onValueChange = viewModel::onBodyChange,
                textStyle = TextStyle(
                    fontFamily = ShipporiMincho,
                    fontSize = 18.sp,
                    lineHeight = 32.sp,
                    color = RunaColors.Body,
                ),
                cursorBrush = SolidColor(RunaColors.Accent),
                modifier = Modifier
                    .fillMaxWidth()
                    .weight(1f),
            )

            Row(
                modifier = Modifier
                    .fillMaxWidth()
                    .padding(top = 16.dp),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                Text(
                    text = stringResource(state.save.autosaveLabelRes()),
                    style = MaterialTheme.typography.labelLarge,
                    color = RunaColors.Subtle,
                    modifier = Modifier.weight(1f),
                )
                Box(
                    modifier = Modifier
                        .border(1.dp, RunaColors.Accent.copy(alpha = 0.7f), RoundedCornerShape(24.dp))
                        .clickable(onClick = leave)
                        .padding(horizontal = 28.dp, vertical = 12.dp),
                ) {
                    Text(
                        text = stringResource(R.string.diary_close),
                        style = MaterialTheme.typography.bodyLarge,
                        color = RunaColors.Accent,
                    )
                }
            }
        }
    }
}

private fun SaveStatus.autosaveLabelRes(): Int = when (this) {
    SaveStatus.Editing -> R.string.diary_editor_autosave
    SaveStatus.Saving -> R.string.diary_editor_saving
    SaveStatus.Saved -> R.string.diary_editor_saved
    SaveStatus.Error -> R.string.diary_editor_error
}
