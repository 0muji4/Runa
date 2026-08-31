package com.runa.android.ui.components

import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.RowScope
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.unit.dp
import com.runa.android.R
import com.runa.android.ui.theme.RunaColors
import com.runa.android.ui.theme.RunaHeader

/**
 * The one screen header. Every screen that carries a title uses this and writes no
 * header of its own, so the title size, the back affordance and the vertical rhythm
 * are identical everywhere (README「画面ヘッダー（全画面共通の型）」is the canon).
 *
 * [onBack] decides the variant: `null` means a bottom-tab root (starts at
 * [RunaHeader.TopTab], no back row), otherwise a pushed screen (starts at
 * [RunaHeader.TopPushed] with the「‹ 戻る」row above the title).
 *
 * [title] is null on the one pushed screen that carries no screen title (the
 * retrospective calendar, whose month stepper is content rather than a heading), so
 * it still gets the same back affordance and the same top offset as its siblings.
 *
 * Horizontal padding is deliberately absent — the caller's container supplies it, so
 * the title always lines up with the body beneath it rather than with a value chosen
 * here.
 */
@Composable
fun RunaScreenHeader(
    title: String? = null,
    modifier: Modifier = Modifier,
    onBack: (() -> Unit)? = null,
    actions: @Composable RowScope.() -> Unit = {},
) {
    Column(modifier.fillMaxWidth()) {
        if (onBack == null) {
            Spacer(Modifier.height(RunaHeader.TopTab))
        } else {
            Spacer(Modifier.height(RunaHeader.TopPushed))
            Text(
                text = "‹ " + stringResource(R.string.action_back),
                style = MaterialTheme.typography.labelMedium,
                color = RunaColors.Subtle,
                modifier = Modifier
                    .clickable(onClick = onBack)
                    // Widens the touch target without moving the glyph off the margin.
                    .padding(vertical = 6.dp, end = 12.dp),
            )
            if (title != null) Spacer(Modifier.height(RunaHeader.BackGap))
        }

        if (title != null) {
            Row(
                modifier = Modifier.fillMaxWidth(),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                Text(
                    text = title,
                    style = MaterialTheme.typography.headlineLarge,
                    color = RunaColors.Heading,
                    modifier = Modifier.weight(1f),
                )
                actions()
            }
        }

        Spacer(Modifier.height(RunaHeader.Bottom))
    }
}
