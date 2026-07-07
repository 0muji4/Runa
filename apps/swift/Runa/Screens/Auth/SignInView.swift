import AuthenticationServices
import SwiftUI

/// Sign-in screen (screen 05). Apple (native button), Google, and email — email
/// toggling between login and signup. Matches the design system's quiet tone.
struct SignInView: View {
    let isBusy: Bool
    let errorMessage: String?
    let onApple: (_ idToken: String, _ displayName: String?) -> Void
    let onAppleError: (String) -> Void
    let onGoogle: () -> Void
    let onEmailSubmit: (_ isSignup: Bool, _ email: String, _ password: String, _ displayName: String) -> Void

    @State private var isSignup = false
    @State private var email = ""
    @State private var password = ""
    @State private var displayName = ""

    private var canSubmit: Bool {
        !isBusy && !email.trimmingCharacters(in: .whitespaces).isEmpty && !password.isEmpty
    }

    var body: some View {
        ZStack {
            RunaColors.background.ignoresSafeArea()

            ScrollView {
                VStack(spacing: RunaSpacing.sm) {
                    Text("Runa")
                        .font(RunaFonts.logo(40))
                        .foregroundStyle(RunaColors.heading)
                    Text("サインイン")
                        .font(RunaFonts.heading(24))
                        .foregroundStyle(RunaColors.heading)
                        .padding(.bottom, RunaSpacing.xs)

                    SignInWithAppleButton(.signIn) { request in
                        request.requestedScopes = [.fullName, .email]
                    } onCompletion: { result in
                        handleApple(result)
                    }
                    .signInWithAppleButtonStyle(.white)
                    .frame(height: 48)
                    .disabled(isBusy)

                    socialButton(title: "Googleでサインイン", action: onGoogle)

                    Text("または")
                        .font(RunaFonts.body(13))
                        .foregroundStyle(RunaColors.subtle)
                        .padding(.vertical, RunaSpacing.xs)

                    field("メールアドレス", text: $email, keyboard: .emailAddress)
                    secureField("パスワード", text: $password)
                    if isSignup {
                        field("表示名（任意）", text: $displayName, keyboard: .default)
                    }

                    Button {
                        onEmailSubmit(
                            isSignup,
                            email.trimmingCharacters(in: .whitespaces),
                            password,
                            displayName.trimmingCharacters(in: .whitespaces)
                        )
                    } label: {
                        Text(isSignup ? "新規登録" : "ログイン")
                            .font(RunaFonts.body(16))
                            .frame(maxWidth: .infinity)
                            .padding(.vertical, RunaSpacing.sm)
                            .background(canSubmit ? RunaColors.accent : RunaColors.surface)
                            .foregroundStyle(canSubmit ? RunaColors.background : RunaColors.subtle)
                            .clipShape(RoundedRectangle(cornerRadius: 12))
                    }
                    .disabled(!canSubmit)

                    Button(isSignup ? "すでにアカウントをお持ちの方はこちら" : "アカウントをお持ちでない方はこちら") {
                        isSignup.toggle()
                    }
                    .font(RunaFonts.body(13))
                    .foregroundStyle(RunaColors.subAccent)
                    .disabled(isBusy)

                    if isBusy {
                        ProgressView().tint(RunaColors.accent).padding(.top, RunaSpacing.xs)
                    }
                    if let errorMessage {
                        Text(errorMessage)
                            .font(RunaFonts.body(13))
                            .foregroundStyle(RunaColors.accent)
                            .multilineTextAlignment(.center)
                    }
                }
                .padding(.horizontal, RunaSpacing.md)
                .padding(.vertical, RunaSpacing.lg)
            }
        }
    }

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

    private func socialButton(title: String, action: @escaping () -> Void) -> some View {
        Button(action: action) {
            Text(title)
                .font(RunaFonts.body(16))
                .frame(maxWidth: .infinity)
                .padding(.vertical, RunaSpacing.sm)
                .background(RunaColors.surface)
                .foregroundStyle(RunaColors.heading)
                .clipShape(RoundedRectangle(cornerRadius: 12))
        }
        .disabled(isBusy)
    }

    private func field(_ label: String, text: Binding<String>, keyboard: UIKeyboardType) -> some View {
        TextField("", text: text, prompt: Text(label).foregroundColor(RunaColors.subtle))
            .keyboardType(keyboard)
            .textInputAutocapitalization(.never)
            .autocorrectionDisabled()
            .foregroundStyle(RunaColors.heading)
            .padding(RunaSpacing.sm)
            .background(RunaColors.surface)
            .clipShape(RoundedRectangle(cornerRadius: 12))
    }

    private func secureField(_ label: String, text: Binding<String>) -> some View {
        SecureField("", text: text, prompt: Text(label).foregroundColor(RunaColors.subtle))
            .foregroundStyle(RunaColors.heading)
            .padding(RunaSpacing.sm)
            .background(RunaColors.surface)
            .clipShape(RoundedRectangle(cornerRadius: 12))
    }
}
