package com.runa.android.ui.screens

import android.content.Context
import android.content.Intent
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.OutlinedTextFieldDefaults
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.ImeAction
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.runa.android.R
import com.runa.android.ui.theme.RunaColors
import com.runa.shared.feature.settings.AccountViewModel
import com.runa.shared.feature.settings.DeletionStatus
import com.runa.shared.feature.settings.ExportStatus
import org.koin.compose.koinInject

/**
 * アカウント・データ (23). Profile display + display-name editing, data export
 * (text or JSON via the system share sheet) and account deletion (with a
 * confirmation step). Sign-out lives here per the confirmed design. On successful
 * deletion the shared auth state drops to unauthenticated, so the app root returns
 * to sign-in on its own — no navigation needed here.
 */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun AccountScreen(
    onBack: () -> Unit,
    onSignOut: () -> Unit,
    viewModel: AccountViewModel = koinInject(),
) {
    val state by viewModel.state.collectAsState()
    val context = LocalContext.current

    // When an export is ready, offer the two formats and hand off to the OS share sheet.
    (state.export as? ExportStatus.Ready)?.let { ready ->
        val shareTitle = stringResource(R.string.account_export_share_title)
        val jsonLabel = stringResource(R.string.account_export_json)
        val textLabel = stringResource(R.string.account_export_text)
        AlertDialog(
            onDismissRequest = viewModel::clearExport,
            containerColor = RunaColors.Surface,
            title = { Text(stringResource(R.string.account_export), color = RunaColors.Heading) },
            text = {
                Column {
                    TextButton(onClick = {
                        shareText(context, ready.text, shareTitle)
                        viewModel.clearExport()
                    }) { Text(textLabel, color = RunaColors.Accent) }
                    TextButton(onClick = {
                        shareText(context, ready.json, shareTitle)
                        viewModel.clearExport()
                    }) { Text(jsonLabel, color = RunaColors.Accent) }
                }
            },
            confirmButton = {
                TextButton(onClick = viewModel::clearExport) {
                    Text(stringResource(R.string.action_cancel), color = RunaColors.Subtle)
                }
            },
        )
    }

    // Deletion confirmation.
    if (state.deletion == DeletionStatus.Confirming) {
        AlertDialog(
            onDismissRequest = viewModel::cancelDelete,
            containerColor = RunaColors.Surface,
            title = { Text(stringResource(R.string.account_delete_confirm_title), color = RunaColors.Heading) },
            text = { Text(stringResource(R.string.account_delete_confirm_body), color = RunaColors.Body) },
            confirmButton = {
                TextButton(onClick = viewModel::confirmDelete) {
                    Text(stringResource(R.string.action_delete), color = RunaColors.Accent)
                }
            },
            dismissButton = {
                TextButton(onClick = viewModel::cancelDelete) {
                    Text(stringResource(R.string.action_cancel), color = RunaColors.Subtle)
                }
            },
        )
    }

    Column(
        modifier = Modifier
            .fillMaxSize()
            .background(RunaColors.Background)
            .verticalScroll(rememberScrollState())
            .padding(horizontal = 28.dp),
    ) {
        Spacer(Modifier.height(24.dp))
        Text(
            text = stringResource(R.string.action_back),
            color = RunaColors.Subtle,
            fontSize = 15.sp,
            modifier = Modifier.clickable(onClick = onBack),
        )
        Spacer(Modifier.height(24.dp))
        Text(stringResource(R.string.account_eyebrow), color = RunaColors.Subtle, fontSize = 13.sp, letterSpacing = 3.sp)
        Spacer(Modifier.height(8.dp))
        Text(
            text = stringResource(R.string.account_title),
            color = RunaColors.Heading,
            fontWeight = FontWeight.Medium,
            fontSize = 36.sp,
        )
        Spacer(Modifier.height(28.dp))

        when {
            state.isLoadingProfile && state.profile == null ->
                CircularProgressIndicator(color = RunaColors.Accent)
            state.loadError != null && state.profile == null ->
                Text(stringResource(R.string.account_load_error), color = RunaColors.Subtle)
            else -> ProfileSection(state, viewModel)
        }

        Spacer(Modifier.height(36.dp))
        AccountActionRow("↥", stringResource(R.string.account_export), enabled = state.export != ExportStatus.InProgress) {
            viewModel.export()
        }
        if (state.export == ExportStatus.InProgress) {
            Text(stringResource(R.string.account_export_preparing), color = RunaColors.Subtle, fontSize = 13.sp)
        }
        (state.export as? ExportStatus.Error)?.let {
            Text(it.message, color = RunaColors.Accent, fontSize = 13.sp)
        }
        ActionDivider()
        AccountActionRow("⇥", stringResource(R.string.action_sign_out), onClick = onSignOut)

        Spacer(Modifier.height(56.dp))
        val deleting = state.deletion == DeletionStatus.InProgress
        Text(
            text = if (deleting) stringResource(R.string.account_deleting) else stringResource(R.string.account_delete),
            color = RunaColors.Subtle,
            fontSize = 15.sp,
            textAlign = TextAlign.Center,
            modifier = Modifier
                .fillMaxWidth()
                .then(if (deleting) Modifier else Modifier.clickable(onClick = viewModel::requestDelete))
                .padding(vertical = 12.dp),
        )
        (state.deletion as? DeletionStatus.Error)?.let {
            Text(it.message, color = RunaColors.Accent, fontSize = 13.sp, modifier = Modifier.fillMaxWidth(), textAlign = TextAlign.Center)
        }
        Spacer(Modifier.height(32.dp))
    }
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
private fun ProfileSection(
    state: com.runa.shared.feature.settings.AccountUiState,
    viewModel: AccountViewModel,
) {
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .background(RunaColors.Surface, RoundedCornerShape(20.dp))
            .padding(20.dp),
        verticalAlignment = Alignment.CenterVertically,
        horizontalArrangement = Arrangement.spacedBy(16.dp),
    ) {
        Box(
            Modifier
                .size(56.dp)
                .background(RunaColors.SubAccent, CircleShape),
        )
        Column(Modifier.weight(1f)) {
            Text(state.profile?.displayName.orEmpty(), color = RunaColors.Heading, fontSize = 22.sp)
            state.profile?.email?.let {
                Spacer(Modifier.height(4.dp))
                Text(it, color = RunaColors.Subtle, fontSize = 14.sp)
            }
        }
    }

    Spacer(Modifier.height(16.dp))

    if (state.isEditingName) {
        OutlinedTextField(
            value = state.displayNameDraft,
            onValueChange = viewModel::onDisplayNameChange,
            label = { Text(stringResource(R.string.account_name_label)) },
            singleLine = true,
            isError = state.nameError != null,
            keyboardOptions = KeyboardOptions(imeAction = ImeAction.Done),
            colors = OutlinedTextFieldDefaults.colors(
                focusedTextColor = RunaColors.Heading,
                unfocusedTextColor = RunaColors.Heading,
                focusedBorderColor = RunaColors.Accent,
                unfocusedBorderColor = RunaColors.Subtle,
                focusedLabelColor = RunaColors.Accent,
                unfocusedLabelColor = RunaColors.Subtle,
                cursorColor = RunaColors.Accent,
            ),
            modifier = Modifier.fillMaxWidth(),
        )
        state.nameError?.let { Text(it, color = RunaColors.Accent, fontSize = 13.sp) }
        Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
            TextButton(onClick = viewModel::saveName, enabled = !state.isSavingName) {
                Text(stringResource(R.string.account_save), color = RunaColors.Accent)
            }
            TextButton(onClick = viewModel::cancelEditName) {
                Text(stringResource(R.string.action_cancel), color = RunaColors.Subtle)
            }
        }
    } else {
        Text(
            text = stringResource(R.string.account_edit_name),
            color = RunaColors.Accent,
            fontSize = 15.sp,
            modifier = Modifier.clickable(onClick = viewModel::startEditName),
        )
    }
}

@Composable
private fun AccountActionRow(
    glyph: String,
    label: String,
    enabled: Boolean = true,
    onClick: () -> Unit,
) {
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .then(if (enabled) Modifier.clickable(onClick = onClick) else Modifier)
            .padding(vertical = 20.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Text(glyph, color = RunaColors.Heading, fontSize = 18.sp, modifier = Modifier.width(36.dp))
        Text(label, color = RunaColors.Heading, fontSize = 17.sp, modifier = Modifier.weight(1f))
    }
}

@Composable
private fun ActionDivider() {
    Box(
        Modifier
            .fillMaxWidth()
            .height(1.dp)
            .background(RunaColors.Subtle.copy(alpha = 0.15f)),
    )
}

private fun shareText(context: Context, text: String, title: String) {
    val send = Intent(Intent.ACTION_SEND).apply {
        type = "text/plain"
        putExtra(Intent.EXTRA_TITLE, title)
        putExtra(Intent.EXTRA_TEXT, text)
    }
    context.startActivity(Intent.createChooser(send, title))
}
