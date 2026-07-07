package com.runa.android.ui.screens.auth

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.Button
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.OutlinedTextField
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
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.text.input.PasswordVisualTransformation
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import com.runa.android.R

/** Email sub-mode of the sign-in screen. */
enum class EmailMode { Login, Signup }

/**
 * Sign-in screen (screen 05). Three entry points — Apple, Google, and email —
 * with the email form toggling between login and signup. [isBusy] disables input
 * during a sign-in call; [errorMessage] surfaces failures. Callbacks feed the
 * shared [com.runa.shared.feature.auth.AuthViewModel].
 */
@Composable
fun SignInScreen(
    isBusy: Boolean,
    errorMessage: String?,
    onAppleClick: () -> Unit,
    onGoogleClick: () -> Unit,
    onEmailSubmit: (mode: EmailMode, email: String, password: String, displayName: String) -> Unit,
) {
    var mode by remember { mutableStateOf(EmailMode.Login) }
    var email by remember { mutableStateOf("") }
    var password by remember { mutableStateOf("") }
    var displayName by remember { mutableStateOf("") }

    Surface(color = MaterialTheme.colorScheme.background) {
        Column(
            modifier = Modifier
                .fillMaxSize()
                .verticalScroll(rememberScrollState())
                .padding(horizontal = 32.dp, vertical = 48.dp),
            verticalArrangement = Arrangement.spacedBy(16.dp),
            horizontalAlignment = Alignment.CenterHorizontally,
        ) {
            Text(
                text = stringResource(R.string.brand_name),
                style = MaterialTheme.typography.displayLarge,
            )
            Text(
                text = stringResource(R.string.signin_title),
                style = MaterialTheme.typography.headlineMedium,
                modifier = Modifier.padding(bottom = 8.dp),
            )

            OutlinedButton(
                onClick = onAppleClick,
                enabled = !isBusy,
                modifier = Modifier.fillMaxWidth(),
            ) { Text(stringResource(R.string.signin_apple)) }

            OutlinedButton(
                onClick = onGoogleClick,
                enabled = !isBusy,
                modifier = Modifier.fillMaxWidth(),
            ) { Text(stringResource(R.string.signin_google)) }

            Text(
                text = stringResource(R.string.signin_or),
                style = MaterialTheme.typography.labelLarge,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
                modifier = Modifier.padding(vertical = 8.dp),
            )

            OutlinedTextField(
                value = email,
                onValueChange = { email = it },
                label = { Text(stringResource(R.string.signin_email_label)) },
                singleLine = true,
                enabled = !isBusy,
                keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Email),
                modifier = Modifier.fillMaxWidth(),
            )
            OutlinedTextField(
                value = password,
                onValueChange = { password = it },
                label = { Text(stringResource(R.string.signin_password_label)) },
                singleLine = true,
                enabled = !isBusy,
                visualTransformation = PasswordVisualTransformation(),
                keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Password),
                modifier = Modifier.fillMaxWidth(),
            )
            if (mode == EmailMode.Signup) {
                OutlinedTextField(
                    value = displayName,
                    onValueChange = { displayName = it },
                    label = { Text(stringResource(R.string.signin_name_label)) },
                    singleLine = true,
                    enabled = !isBusy,
                    modifier = Modifier.fillMaxWidth(),
                )
            }

            Button(
                onClick = { onEmailSubmit(mode, email.trim(), password, displayName.trim()) },
                enabled = !isBusy && email.isNotBlank() && password.isNotBlank(),
                modifier = Modifier.fillMaxWidth(),
            ) {
                Text(
                    stringResource(
                        if (mode == EmailMode.Login) R.string.action_login else R.string.action_signup,
                    ),
                )
            }

            TextButton(
                onClick = {
                    mode = if (mode == EmailMode.Login) EmailMode.Signup else EmailMode.Login
                },
                enabled = !isBusy,
            ) {
                Text(
                    stringResource(
                        if (mode == EmailMode.Login) R.string.signin_toggle_to_signup
                        else R.string.signin_toggle_to_login,
                    ),
                )
            }

            if (isBusy) {
                CircularProgressIndicator(color = MaterialTheme.colorScheme.primary)
            }
            if (errorMessage != null) {
                Text(
                    text = errorMessage,
                    style = MaterialTheme.typography.bodyMedium,
                    color = MaterialTheme.colorScheme.error,
                    textAlign = TextAlign.Center,
                    modifier = Modifier.fillMaxWidth(),
                )
            }
        }
    }
}
