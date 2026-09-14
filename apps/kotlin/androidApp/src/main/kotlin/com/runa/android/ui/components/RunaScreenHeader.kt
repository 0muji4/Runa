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
 * The one screen header (offsets shared with iOS/README, drift-guarded). `onBack == null` means a
 * bottom-tab root; a null [title] keeps the back row and top offset. No horizontal padding:
 * the caller's container supplies it.
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
                    .padding(top = 6.dp, bottom = 6.dp, end = 12.dp),
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
