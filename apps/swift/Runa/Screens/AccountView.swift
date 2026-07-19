import SwiftUI
import Shared

/// アカウント・データ (23). Profile display + display-name editing, data export (text
/// or JSON via the system share sheet) and account deletion (with confirmation).
/// Sign-out lives here per the confirmed design. On successful deletion the shared
/// auth state drops to unauthenticated, so the app root returns to sign-in on its own.
struct AccountView: View {
    let onSignOut: () -> Void

    @Environment(\.runaTheme) private var runaTheme
    @StateObject private var account = AccountObservable()

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 0) {
                Text("ACCOUNT")
                    .font(RunaFonts.body(13)).tracking(3)
                    .foregroundStyle(runaTheme.subtle)
                    .padding(.top, RunaSpacing.md)
                Text("アカウント・データ")
                    .font(RunaFonts.heading(34))
                    .foregroundStyle(runaTheme.heading)
                    .padding(.top, RunaSpacing.xs)
                    .padding(.bottom, RunaSpacing.lg)

                if let state = account.state {
                    profileSection(state)
                    exportSection(state)
                    divider
                    actionRow(glyph: "⇥", label: "サインアウト", action: onSignOut)
                    Spacer().frame(height: RunaSpacing.xl)
                    deleteSection(state)
                } else {
                    ProgressView().tint(runaTheme.accent)
                }
            }
            .padding(.horizontal, 28)
        }
        .background(runaTheme.background)
        .navigationTitle("")
        .navigationBarTitleDisplayMode(.inline)
        .onAppear { account.loadProfile() }
        .alert("アカウントを削除しますか", isPresented: deleteConfirmBinding) {
            Button("削除する", role: .destructive) { account.confirmDelete() }
            Button("やめる", role: .cancel) { account.cancelDelete() }
        } message: {
            Text("日記・画像を含むすべてのデータが完全に削除され、元に戻せません。")
        }
    }

    // MARK: profile + name editing

    @ViewBuilder
    private func profileSection(_ state: AccountUiState) -> some View {
        HStack(spacing: RunaSpacing.sm) {
            Circle().fill(runaTheme.subAccent).frame(width: 56, height: 56)
            VStack(alignment: .leading, spacing: 4) {
                Text(state.profile?.displayName ?? "")
                    .font(RunaFonts.heading(22)).foregroundStyle(runaTheme.heading)
                if let email = state.profile?.email {
                    Text(email).font(RunaFonts.body(14)).foregroundStyle(runaTheme.subtle)
                }
            }
            Spacer()
        }
        .padding(20)
        .background(runaTheme.surface, in: RoundedRectangle(cornerRadius: 20))

        if state.isEditingName {
            let draft = Binding(
                get: { state.displayNameDraft },
                set: { account.onDisplayNameChange($0) }
            )
            TextField("表示名", text: draft)
                .font(RunaFonts.body(16))
                .foregroundStyle(runaTheme.heading)
                .padding(12)
                .background(runaTheme.surface, in: RoundedRectangle(cornerRadius: 12))
                .padding(.top, RunaSpacing.sm)
            if let error = state.nameError {
                Text(error).font(RunaFonts.body(13)).foregroundStyle(runaTheme.accent)
            }
            HStack(spacing: RunaSpacing.sm) {
                Button("保存") { account.saveName() }
                    .foregroundStyle(runaTheme.accent).disabled(state.isSavingName)
                Button("やめる") { account.cancelEditName() }
                    .foregroundStyle(runaTheme.subtle)
            }
            .padding(.top, RunaSpacing.xs)
        } else {
            Button("表示名を編集") { account.startEditName() }
                .font(RunaFonts.body(15))
                .foregroundStyle(runaTheme.accent)
                .padding(.top, RunaSpacing.sm)
        }

        Spacer().frame(height: RunaSpacing.md)
    }

    // MARK: export

    @ViewBuilder
    private func exportSection(_ state: AccountUiState) -> some View {
        actionRow(glyph: "↥", label: "エクスポート") { account.export() }
        switch onEnum(of: state.export) {
        case .inProgress:
            Text("エクスポートを準備しています…")
                .font(RunaFonts.body(13)).foregroundStyle(runaTheme.subtle)
        case .ready(let ready):
            VStack(alignment: .leading, spacing: RunaSpacing.xs) {
                ShareLink(item: ready.text) {
                    Text("テキストで書き出す").font(RunaFonts.body(15)).foregroundStyle(runaTheme.accent)
                }
                ShareLink(item: ready.json) {
                    Text("JSON で書き出す").font(RunaFonts.body(15)).foregroundStyle(runaTheme.accent)
                }
                Button("閉じる") { account.clearExport() }
                    .font(RunaFonts.body(14)).foregroundStyle(runaTheme.subtle)
            }
            .padding(.leading, 36)
            .padding(.bottom, RunaSpacing.xs)
        case .error(let error):
            Text(error.message).font(RunaFonts.body(13)).foregroundStyle(runaTheme.accent)
        case .idle:
            EmptyView()
        }
    }

    // MARK: deletion

    @ViewBuilder
    private func deleteSection(_ state: AccountUiState) -> some View {
        let deleting = isInProgress(state.deletion)
        Text(deleting ? "削除しています…" : "アカウントを削除")
            .font(RunaFonts.body(15))
            .foregroundStyle(runaTheme.subtle)
            .frame(maxWidth: .infinity)
            .padding(.vertical, 12)
            .contentShape(Rectangle())
            .onTapGesture { if !deleting { account.requestDelete() } }
        if case .error(let error) = onEnum(of: state.deletion) {
            Text(error.message)
                .font(RunaFonts.body(13)).foregroundStyle(runaTheme.accent)
                .frame(maxWidth: .infinity)
        }
    }

    // MARK: helpers

    private var divider: some View {
        Rectangle().fill(runaTheme.subtle.opacity(0.15)).frame(height: 1)
    }

    private func actionRow(glyph: String, label: String, action: @escaping () -> Void) -> some View {
        Button(action: action) {
            HStack {
                Text(glyph).font(RunaFonts.body(18)).foregroundStyle(runaTheme.heading).frame(width: 36, alignment: .leading)
                Text(label).font(RunaFonts.body(17)).foregroundStyle(runaTheme.heading)
                Spacer()
            }
            .padding(.vertical, 20)
            .contentShape(Rectangle())
        }
        .buttonStyle(.plain)
    }

    private var deleteConfirmBinding: Binding<Bool> {
        Binding(
            get: {
                guard let state = account.state else { return false }
                if case .confirming = onEnum(of: state.deletion) { return true }
                return false
            },
            set: { presented in if !presented { account.cancelDelete() } }
        )
    }

    private func isInProgress(_ status: DeletionStatus) -> Bool {
        if case .inProgress = onEnum(of: status) { return true }
        return false
    }
}

#Preview {
    NavigationStack { AccountView(onSignOut: {}) }
}
