package com.runa.android.ui.screens.auth

import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.BasicTextField
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.SolidColor
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.text.input.PasswordVisualTransformation
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.runa.android.R
import com.runa.android.ui.components.GlowingMoon
import com.runa.android.ui.theme.RunaColors

/** Email sub-mode of the sign-in screen. */
enum class EmailMode { Login, Signup }

/**
 * Sign-in screen (05). The quiet three-choice design: a glowing moon over the LUNA
 * wordmark and a poetic line, then Apple / Google / メール, with いまはスキップ below.
 * "メールでつづける" opens a second, still email step rather than crowding the hero.
 */
@Composable
fun SignInScreen(
    isBusy: Boolean,
    errorMessage: String?,
    onAppleClick: () -> Unit,
    onGoogleClick: () -> Unit,
    onEmailSubmit: (mode: EmailMode, email: String, password: String, displayName: String) -> Unit,
    onSkip: () -> Unit,
) {
    var showEmail by remember { mutableStateOf(false) }

    Surface(color = RunaColors.Background) {
        if (showEmail) {
            EmailStep(
                isBusy = isBusy,
                errorMessage = errorMessage,
                onSubmit = onEmailSubmit,
                onBack = { showEmail = false },
            )
        } else {
            SignInChoices(
                isBusy = isBusy,
                errorMessage = errorMessage,
                onAppleClick = onAppleClick,
                onGoogleClick = onGoogleClick,
                onEmailClick = { showEmail = true },
                onSkip = onSkip,
            )
        }
    }
}

@Composable
private fun SignInChoices(
    isBusy: Boolean,
    errorMessage: String?,
    onAppleClick: () -> Unit,
    onGoogleClick: () -> Unit,
    onEmailClick: () -> Unit,
    onSkip: () -> Unit,
) {
    Column(
        modifier = Modifier
            .fillMaxSize()
            .verticalScroll(rememberScrollState())
            .padding(horizontal = 36.dp, vertical = 40.dp),
        horizontalAlignment = Alignment.CenterHorizontally,
    ) {
        Spacer(Modifier.height(48.dp))
        GlowingMoon(diameter = 148.dp)
        Spacer(Modifier.height(20.dp))
        Text(
            text = stringResource(R.string.logo_wordmark),
            style = MaterialTheme.typography.displayLarge.copy(letterSpacing = 14.sp),
            color = RunaColors.Heading,
        )
        Spacer(Modifier.height(18.dp))
        Text(
            text = stringResource(R.string.signin_tagline),
            style = MaterialTheme.typography.titleLarge,
            color = RunaColors.Heading,
            textAlign = TextAlign.Center,
        )

        Spacer(Modifier.height(56.dp))

        FilledPillButton(
            text = stringResource(R.string.signin_apple),
            onClick = onAppleClick,
            enabled = !isBusy,
        )
        Spacer(Modifier.height(14.dp))
        SurfacePillButton(
            text = stringResource(R.string.signin_google),
            onClick = onGoogleClick,
            enabled = !isBusy,
        )
        Spacer(Modifier.height(14.dp))
        SurfacePillButton(
            text = stringResource(R.string.signin_email_continue),
            onClick = onEmailClick,
            enabled = !isBusy,
        )

        Spacer(Modifier.height(28.dp))
        if (isBusy) CircularProgressIndicator(color = RunaColors.Accent, strokeWidth = 2.dp)
        errorMessage?.let { ErrorLine(it) }

        Spacer(Modifier.height(32.dp))
        Text(
            text = stringResource(R.string.signin_skip),
            style = MaterialTheme.typography.labelLarge.copy(letterSpacing = 4.sp),
            color = RunaColors.Subtle,
            modifier = Modifier
                .clickable(enabled = !isBusy, onClick = onSkip)
                .padding(12.dp),
        )
    }
}

@Composable
private fun EmailStep(
    isBusy: Boolean,
    errorMessage: String?,
    onSubmit: (EmailMode, String, String, String) -> Unit,
    onBack: () -> Unit,
) {
    var mode by remember { mutableStateOf(EmailMode.Login) }
    var email by remember { mutableStateOf("") }
    var password by remember { mutableStateOf("") }
    var displayName by remember { mutableStateOf("") }
    val canSubmit = !isBusy && email.isNotBlank() && password.isNotBlank()

    Column(
        modifier = Modifier
            .fillMaxSize()
            .verticalScroll(rememberScrollState())
            .padding(horizontal = 36.dp, vertical = 40.dp),
    ) {
        Text(
            text = "‹ ${stringResource(R.string.signin_email_continue)}",
            style = MaterialTheme.typography.labelLarge,
            color = RunaColors.Subtle,
            modifier = Modifier
                .clickable(enabled = !isBusy, onClick = onBack)
                .padding(vertical = 8.dp),
        )
        Spacer(Modifier.height(24.dp))
        Text(
            text = stringResource(R.string.signin_email_title),
            style = MaterialTheme.typography.headlineMedium,
            color = RunaColors.Heading,
        )
        Spacer(Modifier.height(32.dp))

        QuietField(
            value = email,
            onValueChange = { email = it },
            placeholder = stringResource(R.string.signin_email_label),
            enabled = !isBusy,
            keyboardType = KeyboardType.Email,
        )
        Spacer(Modifier.height(14.dp))
        QuietField(
            value = password,
            onValueChange = { password = it },
            placeholder = stringResource(R.string.signin_password_label),
            enabled = !isBusy,
            keyboardType = KeyboardType.Password,
            isPassword = true,
        )
        if (mode == EmailMode.Signup) {
            Spacer(Modifier.height(14.dp))
            QuietField(
                value = displayName,
                onValueChange = { displayName = it },
                placeholder = stringResource(R.string.signin_name_label),
                enabled = !isBusy,
            )
        }

        Spacer(Modifier.height(24.dp))
        FilledPillButton(
            text = stringResource(if (mode == EmailMode.Login) R.string.action_login else R.string.action_signup),
            onClick = { onSubmit(mode, email.trim(), password, displayName.trim()) },
            enabled = canSubmit,
        )
        Spacer(Modifier.height(16.dp))
        Text(
            text = stringResource(
                if (mode == EmailMode.Login) R.string.signin_toggle_to_signup
                else R.string.signin_toggle_to_login,
            ),
            style = MaterialTheme.typography.labelLarge,
            color = RunaColors.SubAccent,
            textAlign = TextAlign.Center,
            modifier = Modifier
                .fillMaxWidth()
                .clickable(enabled = !isBusy) {
                    mode = if (mode == EmailMode.Login) EmailMode.Signup else EmailMode.Login
                }
                .padding(8.dp),
        )

        Spacer(Modifier.height(16.dp))
        if (isBusy) CircularProgressIndicator(color = RunaColors.Accent, strokeWidth = 2.dp)
        errorMessage?.let { ErrorLine(it) }
    }
}

/** A near-white filled pill (used for the Apple choice and the email submit). */
@Composable
private fun FilledPillButton(text: String, onClick: () -> Unit, enabled: Boolean) {
    Box(
        modifier = Modifier
            .fillMaxWidth()
            .height(56.dp)
            .background(
                color = if (enabled) RunaColors.Heading else RunaColors.Surface,
                shape = RoundedCornerShape(16.dp),
            )
            .clickable(enabled = enabled, onClick = onClick),
        contentAlignment = Alignment.Center,
    ) {
        Text(
            text = text,
            style = MaterialTheme.typography.bodyLarge,
            color = if (enabled) RunaColors.Background else RunaColors.Subtle,
        )
    }
}

/** A quiet dark-surface pill with a hairline border (Google / メール). */
@Composable
private fun SurfacePillButton(text: String, onClick: () -> Unit, enabled: Boolean) {
    Box(
        modifier = Modifier
            .fillMaxWidth()
            .height(56.dp)
            .background(RunaColors.Surface, RoundedCornerShape(16.dp))
            .border(1.dp, RunaColors.Subtle.copy(alpha = 0.18f), RoundedCornerShape(16.dp))
            .clickable(enabled = enabled, onClick = onClick),
        contentAlignment = Alignment.Center,
    ) {
        Text(
            text = text,
            style = MaterialTheme.typography.bodyLarge,
            color = RunaColors.Heading,
        )
    }
}

@Composable
private fun QuietField(
    value: String,
    onValueChange: (String) -> Unit,
    placeholder: String,
    enabled: Boolean,
    keyboardType: KeyboardType = KeyboardType.Text,
    isPassword: Boolean = false,
) {
    Box(
        modifier = Modifier
            .fillMaxWidth()
            .heightIn(min = 52.dp)
            .background(RunaColors.Surface, RoundedCornerShape(14.dp))
            .padding(horizontal = 16.dp, vertical = 15.dp),
        contentAlignment = Alignment.CenterStart,
    ) {
        if (value.isEmpty()) {
            Text(placeholder, style = MaterialTheme.typography.bodyLarge, color = RunaColors.Subtle)
        }
        BasicTextField(
            value = value,
            onValueChange = onValueChange,
            enabled = enabled,
            singleLine = true,
            textStyle = MaterialTheme.typography.bodyLarge.copy(color = RunaColors.Heading),
            cursorBrush = SolidColor(RunaColors.Accent),
            visualTransformation = if (isPassword) PasswordVisualTransformation() else androidx.compose.ui.text.input.VisualTransformation.None,
            keyboardOptions = KeyboardOptions(keyboardType = keyboardType),
            modifier = Modifier.fillMaxWidth(),
        )
    }
}

@Composable
private fun ErrorLine(message: String) {
    Text(
        text = message,
        style = MaterialTheme.typography.bodyMedium,
        color = RunaColors.Accent,
        textAlign = TextAlign.Center,
        modifier = Modifier
            .fillMaxWidth()
            .padding(top = 12.dp),
    )
}
