package com.runa.android.ui.screens.diary

import androidx.activity.compose.BackHandler
import androidx.compose.foundation.horizontalScroll
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.text.BasicTextField
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.FilterChip
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.TopAppBar
import androidx.compose.runtime.Composable
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.remember
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.SolidColor
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.TextStyle
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.runa.android.R
import com.runa.android.ui.theme.ShipporiMincho
import com.runa.shared.feature.diary.DiaryEditorViewModel
import com.runa.shared.feature.diary.SaveStatus
import org.koin.compose.getKoin
import org.koin.core.parameter.parametersOf

/**
 * Diary editor (screen 10) — "書く". A whitespace-first, Shippori Mincho canvas
 * where long-form entries read beautifully and the character count is never
 * shown. Autosave is durable (the entry persists locally from the first line)
 * and its indicator stays deliberately quiet. Mood is an optional, still choice.
 */
@OptIn(ExperimentalMaterial3Api::class)
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

    val leave = {
        viewModel.saveNow()
        onDone()
    }
    BackHandler(onBack = leave)

    Scaffold(
        containerColor = MaterialTheme.colorScheme.background,
        topBar = {
            TopAppBar(
                title = {},
                navigationIcon = {
                    TextButton(onClick = leave) { Text(stringResource(R.string.action_back)) }
                },
                actions = {
                    Text(
                        text = stringResource(state.save.labelRes()),
                        style = MaterialTheme.typography.labelLarge,
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                        modifier = Modifier.padding(end = 16.dp),
                    )
                },
            )
        },
    ) { innerPadding ->
        Column(
            modifier = Modifier
                .fillMaxSize()
                .padding(innerPadding)
                .padding(horizontal = 28.dp),
            verticalArrangement = Arrangement.spacedBy(20.dp),
        ) {
            BasicTextField(
                value = state.bodyText,
                onValueChange = viewModel::onBodyChange,
                textStyle = TextStyle(
                    fontFamily = ShipporiMincho,
                    fontSize = 18.sp,
                    lineHeight = 32.sp,
                    color = MaterialTheme.colorScheme.onSurface,
                ),
                cursorBrush = SolidColor(MaterialTheme.colorScheme.primary),
                modifier = Modifier
                    .fillMaxWidth()
                    .weight(1f)
                    .padding(top = 12.dp),
                decorationBox = { inner ->
                    if (state.bodyText.isEmpty()) {
                        Text(
                            text = stringResource(R.string.diary_editor_placeholder),
                            style = TextStyle(fontFamily = ShipporiMincho, fontSize = 18.sp, lineHeight = 32.sp),
                            color = MaterialTheme.colorScheme.onSurfaceVariant,
                        )
                    }
                    inner()
                },
            )

            MoodRow(
                selected = state.mood,
                onSelect = { picked -> viewModel.onMoodChange(if (picked == state.mood) null else picked) },
            )
        }
    }
}

@Composable
private fun MoodRow(selected: String?, onSelect: (String) -> Unit) {
    Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
        Text(
            text = stringResource(R.string.diary_mood_label),
            style = MaterialTheme.typography.labelLarge,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
        )
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .horizontalScroll(rememberScrollState())
                .padding(bottom = 12.dp),
            horizontalArrangement = Arrangement.spacedBy(8.dp),
        ) {
            DiaryMood.entries.forEach { mood ->
                FilterChip(
                    selected = selected == mood.value,
                    onClick = { onSelect(mood.value) },
                    label = { Text(stringResource(mood.labelRes)) },
                )
            }
        }
    }
}

private fun SaveStatus.labelRes(): Int = when (this) {
    SaveStatus.Editing -> R.string.diary_editor_editing
    SaveStatus.Saving -> R.string.diary_editor_saving
    SaveStatus.Saved -> R.string.diary_editor_saved
    SaveStatus.Error -> R.string.diary_editor_error
}
