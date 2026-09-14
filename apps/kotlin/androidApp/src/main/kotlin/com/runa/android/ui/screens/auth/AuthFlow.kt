package com.runa.android.ui.screens.auth

import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.saveable.Saver
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.stringResource
import com.runa.android.R
import com.runa.android.auth.AppleWebSignIn
import com.runa.android.auth.rememberGoogleSignIn
import com.runa.shared.feature.auth.AuthState
import com.runa.shared.feature.auth.AuthViewModel
import java.util.UUID

private enum class AuthStep { Onboarding1, Onboarding2, Notifications, SignIn }

/** enum を Bundle に入れるための明示的な Saver。名前で保存して valueOf で戻す。 */
private val AuthStepSaver = Saver<AuthStep, String>(
    save = { it.name },
    restore = { AuthStep.valueOf(it) },
)

/**
 * Unauthenticated flow: onboarding ①② → notification shell → sign-in. Native errors
 * (Google cancelled, provider unconfigured) are surfaced locally beside [AuthState.Error].
 */
@Composable
fun AuthFlow(
    state: AuthState,
    authViewModel: AuthViewModel,
) {
    // 回転で作り直されると、サインインまで進んでいてもオンボーディング1に巻き戻る。
    var step by rememberSaveable(stateSaver = AuthStepSaver) { mutableStateOf(AuthStep.Onboarding1) }
    var localError by remember { mutableStateOf<String?>(null) }
    val context = LocalContext.current
    val appleUnconfigured = stringResource(R.string.signin_provider_unconfigured)

    val launchGoogle = rememberGoogleSignIn(
        onIdToken = { token ->
            localError = null
            authViewModel.clearError()
            authViewModel.loginGoogle(token)
        },
        onError = { localError = it },
    )

    when (step) {
        AuthStep.Onboarding1 -> OnboardingScreen(
            titleRes = R.string.onboarding_1_title,
            onNext = { step = AuthStep.Onboarding2 },
        )

        AuthStep.Onboarding2 -> OnboardingScreen(
            titleRes = R.string.onboarding_2_title,
            onNext = { step = AuthStep.Notifications },
        )

        AuthStep.Notifications -> NotificationPermissionScreen(
            onContinue = { step = AuthStep.SignIn },
            onSkip = { step = AuthStep.SignIn },
        )

        AuthStep.SignIn -> SignInScreen(
            isBusy = state is AuthState.Authenticating,
            errorMessage = localError ?: (state as? AuthState.Error)?.message,
            onAppleClick = {
                localError = null
                authViewModel.clearError()
                if (AppleWebSignIn.isConfigured()) {
                    AppleWebSignIn.launch(
                        context = context,
                        state = UUID.randomUUID().toString(),
                        nonce = UUID.randomUUID().toString(),
                    )
                } else {
                    localError = appleUnconfigured
                }
            },
            onGoogleClick = {
                localError = null
                authViewModel.clearError()
                launchGoogle()
            },
            onEmailSubmit = { mode, email, password, displayName ->
                localError = null
                authViewModel.clearError()
                when (mode) {
                    EmailMode.Login -> authViewModel.loginEmail(email, password)
                    EmailMode.Signup -> authViewModel.signupEmail(email, password, displayName.ifBlank { null })
                }
            },
        )
    }
}
