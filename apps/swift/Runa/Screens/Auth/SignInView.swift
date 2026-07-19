import AuthenticationServices
import SwiftUI

/// Sign-in screen (05). The quiet three-choice design: a glowing moon over the LUNA
/// wordmark and a poetic line, then Apple / Google / メール with いまはスキップ below.
/// "メールでつづける" opens a second, still email step rather than crowding the hero.
struct SignInView: View {
    @Environment(\.runaTheme) private var runaTheme
    let isBusy: Bool
    let errorMessage: String?
    let onApple: (_ idToken: String, _ displayName: String?) -> Void
    let onAppleError: (String) -> Void
    let onGoogle: () -> Void
    let onEmailSubmit: (_ isSignup: Bool, _ email: String, _ password: String, _ displayName: String) -> Void
    let onSkip: () -> Void

    @State private var showEmail = false

    var body: some View {
        ZStack {
            runaTheme.background.ignoresSafeArea()
            if showEmail {
                emailStep
            } else {
                choices
            }
        }
    }

    // MARK: choices

    private var choices: some View {
        ScrollView {
            VStack(spacing: 0) {
                Spacer().frame(height: 44)
                GlowingMoon(diameter: 148)
                Text("LUNA")
                    .font(RunaFonts.logo(44))
                    .tracking(14)
                    .foregroundStyle(runaTheme.heading)
                    .padding(.top, RunaSpacing.md)
                Text("あなたの夜を、はじめましょう。")
                    .font(RunaFonts.heading(22))
                    .foregroundStyle(runaTheme.heading)
                    .multilineTextAlignment(.center)
                    .padding(.top, RunaSpacing.sm)

                Spacer().frame(height: 52)

                SignInWithAppleButton(.signIn) { request in
                    request.requestedScopes = [.fullName, .email]
                } onCompletion: { handleApple($0) }
                    .signInWithAppleButtonStyle(.white)
                    .frame(height: 56)
                    .clipShape(RoundedRectangle(cornerRadius: 16))
                    .disabled(isBusy)

                surfacePill(title: "Googleでつづける", action: onGoogle).padding(.top, 14)
                surfacePill(title: "メールでつづける", action: { showEmail = true }).padding(.top, 14)

                if isBusy { ProgressView().tint(runaTheme.accent).padding(.top, RunaSpacing.md) }
                if let errorMessage { errorLine(errorMessage) }

                Text("いまはスキップ")
                    .font(RunaFonts.body(13))
                    .tracking(4)
                    .foregroundStyle(runaTheme.subtle)
                    .padding(12)
                    .padding(.top, RunaSpacing.md)
                    .onTapGesture { if !isBusy { onSkip() } }
            }
            .padding(.horizontal, 36)
            .padding(.vertical, RunaSpacing.lg)
        }
    }

    // MARK: email step

    @State private var isSignup = false
    @State private var email = ""
    @State private var password = ""
    @State private var displayName = ""
    private var canSubmit: Bool {
        !isBusy && !email.trimmingCharacters(in: .whitespaces).isEmpty && !password.isEmpty
    }

    private var emailStep: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 0) {
                Text("‹ メールでつづける")
                    .font(RunaFonts.body(14))
                    .foregroundStyle(runaTheme.subtle)
                    .padding(.vertical, 8)
                    .onTapGesture { if !isBusy { showEmail = false } }

                Text("メールではじめる")
                    .font(RunaFonts.heading(26))
                    .foregroundStyle(runaTheme.heading)
                    .padding(.top, RunaSpacing.md)

                quietField("メールアドレス", text: $email, keyboard: .emailAddress).padding(.top, RunaSpacing.md)
                quietSecureField("パスワード", text: $password).padding(.top, 14)
                if isSignup {
                    quietField("表示名（任意）", text: $displayName, keyboard: .default).padding(.top, 14)
                }

                filledPill(
                    title: isSignup ? "新規登録" : "ログイン",
                    enabled: canSubmit
                ) {
                    onEmailSubmit(
                        isSignup,
                        email.trimmingCharacters(in: .whitespaces),
                        password,
                        displayName.trimmingCharacters(in: .whitespaces)
                    )
                }
                .padding(.top, RunaSpacing.md)

                Text(isSignup ? "すでにアカウントをお持ちの方はこちら" : "アカウントをお持ちでない方はこちら")
                    .font(RunaFonts.body(13))
                    .foregroundStyle(runaTheme.subAccent)
                    .frame(maxWidth: .infinity, alignment: .center)
                    .padding(.top, RunaSpacing.sm)
                    .onTapGesture { if !isBusy { isSignup.toggle() } }

                if isBusy { ProgressView().tint(runaTheme.accent).frame(maxWidth: .infinity).padding(.top, RunaSpacing.sm) }
                if let errorMessage { errorLine(errorMessage) }
            }
            .padding(.horizontal, 36)
            .padding(.vertical, RunaSpacing.lg)
        }
    }

    // MARK: pieces

    private func handleApple(_ result: Result<ASAuthorization, Error>) {
        switch result {
        case .success(let authorization):
            guard let credential = authorization.credential as? ASAuthorizationAppleIDCredential,
                  let tokenData = credential.identityToken,
                  let idToken = String(data: tokenData, encoding: .utf8) else {
                onAppleError("Apple IDトークンを取得できませんでした。")
                return
            }
            let name = [credential.fullName?.givenName, credential.fullName?.familyName]
                .compactMap { $0 }
                .joined(separator: " ")
            onApple(idToken, name.isEmpty ? nil : name)
        case .failure(let error):
            onAppleError(error.localizedDescription)
        }
    }

    private func filledPill(title: String, enabled: Bool, action: @escaping () -> Void) -> some View {
        Button(action: action) {
            Text(title)
                .font(RunaFonts.body(16))
                .frame(maxWidth: .infinity)
                .frame(height: 56)
                .background(enabled ? runaTheme.heading : runaTheme.surface)
                .foregroundStyle(enabled ? runaTheme.background : runaTheme.subtle)
                .clipShape(RoundedRectangle(cornerRadius: 16))
        }
        .disabled(!enabled)
    }

    private func surfacePill(title: String, action: @escaping () -> Void) -> some View {
        Button(action: action) {
            Text(title)
                .font(RunaFonts.body(16))
                .frame(maxWidth: .infinity)
                .frame(height: 56)
                .background(runaTheme.surface)
                .foregroundStyle(runaTheme.heading)
                .clipShape(RoundedRectangle(cornerRadius: 16))
                .overlay(
                    RoundedRectangle(cornerRadius: 16)
                        .stroke(runaTheme.subtle.opacity(0.18), lineWidth: 1)
                )
        }
        .disabled(isBusy)
    }

    private func quietField(_ label: String, text: Binding<String>, keyboard: UIKeyboardType) -> some View {
        TextField("", text: text, prompt: Text(label).foregroundColor(runaTheme.subtle))
            .keyboardType(keyboard)
            .textInputAutocapitalization(.never)
            .autocorrectionDisabled()
            .foregroundStyle(runaTheme.heading)
            .padding(.horizontal, RunaSpacing.sm)
            .frame(height: 52)
            .background(runaTheme.surface)
            .clipShape(RoundedRectangle(cornerRadius: 14))
            .disabled(isBusy)
    }

    private func quietSecureField(_ label: String, text: Binding<String>) -> some View {
        SecureField("", text: text, prompt: Text(label).foregroundColor(runaTheme.subtle))
            .foregroundStyle(runaTheme.heading)
            .padding(.horizontal, RunaSpacing.sm)
            .frame(height: 52)
            .background(runaTheme.surface)
            .clipShape(RoundedRectangle(cornerRadius: 14))
            .disabled(isBusy)
    }

    private func errorLine(_ message: String) -> some View {
        Text(message)
            .font(RunaFonts.body(13))
            .foregroundStyle(runaTheme.accent)
            .multilineTextAlignment(.center)
            .frame(maxWidth: .infinity)
            .padding(.top, RunaSpacing.sm)
    }
}
