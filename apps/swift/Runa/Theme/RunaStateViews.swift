import SwiftUI
import Shared

/// The four shared state screens — LUNA's 空 / オフライン / ローディング / エラー
/// surfaces (confirmed designs 24–27). Every feature renders these instead of
/// re-implementing loading / empty / offline / error, so the world-view (the moon
/// motif, the quiet voice) stays consistent and identical to Android's
/// `RunaStateViews`.
///
/// Observables decode the shared `UiState<T>` into a native Swift enum via [runaDecode]
/// (SKIE bridges the generic sealed type to a bare protocol without `onEnum`), and the
/// views hand each case to one of these components. The emblems (`GlowingMoon` / `NewMoonEmblem` /
/// `CloudedMoon` / `StumbleEmblem`) are the fixed cross-theme motif from
/// `MoonArt.swift`; the surrounding text and CTAs read the theme tokens. The loading
/// indicator is a quiet moon + three dots (never a spinner) and honors reduced
/// motion.

// MARK: - Re-authenticate environment

private struct RunaReauthenticateKey: EnvironmentKey {
    static let defaultValue: () -> Void = {}
}

extension EnvironmentValues {
    /// The app-wide re-authenticate action (clears the session so the shared auth
    /// state drops to sign-in). Provided once at the root; read by `RunaErrorView`'s
    /// auth CTA. Mirrors Android's `LocalReauthenticate`; the global
    /// `TokenStore.sessionExpired` signal remains the primary re-auth path (DoD #3).
    var runaReauthenticate: () -> Void {
        get { self[RunaReauthenticateKey.self] }
        set { self[RunaReauthenticateKey.self] = newValue }
    }
}

// MARK: - UiState decoding (SKIE)

/// A page-level state decoded from the shared, generic `UiState<T>`. SKIE bridges the
/// *generic* sealed `UiState` to a bare Swift protocol without `onEnum` support, so we
/// decode it with `as?` casts to the generated case classes (see [runaDecode]).
enum RunaUi<T> {
    case loading
    case empty
    case content(T, SyncPhase)
    case failure(AppError)
}

/// Decode a SKIE `UiState` emission into a native [RunaUi]. The content payload arrives
/// type-erased from the generic bridge (`UiStateContent<AnyObject>`), so it is downcast
/// to [T] here — a list payload bridges from `NSArray`, an object payload from its class.
func runaDecode<T>(_ value: Any, as type: T.Type) -> RunaUi<T> {
    if value is UiStateLoading { return .loading }
    if value is UiStateEmpty { return .empty }
    if let content = value as? UiStateContent<AnyObject>, let data = content.data as? T {
        return .content(data, content.sync)
    }
    if let failure = value as? UiStateFailure { return .failure(failure.error) }
    return .loading
}

// MARK: - Loading (26)

/// Loading: a quiet glowing moon + three dots. Never a spinner; reduced-motion safe.
struct RunaLoadingView: View {
    @Environment(\.runaTheme) private var runaTheme
    @Environment(\.accessibilityReduceMotion) private var reduceMotion
    var caption: String = "月が、のぼるまで。"

    var body: some View {
        RunaStateScaffold {
            GlowingMoon(diameter: 132)
            Text(caption)
                .font(RunaFonts.heading(22)).foregroundStyle(runaTheme.heading)
                .multilineTextAlignment(.center)
            RunaThreeDotProgress(animate: !reduceMotion)
                .padding(.top, RunaSpacing.xs)
        }
    }
}

// MARK: - Empty (24)

/// Empty: the new-moon emblem over a quiet invitation. Copy is per-feature.
struct RunaEmptyView: View {
    @Environment(\.runaTheme) private var runaTheme
    let title: String
    let message: String
    var ctaLabel: String? = nil
    var onCta: (() -> Void)? = nil

    var body: some View {
        RunaStateScaffold {
            NewMoonEmblem(diameter: 116)
            Text(title)
                .font(RunaFonts.heading(26)).foregroundStyle(runaTheme.heading)
                .multilineTextAlignment(.center)
            Text(message)
                .font(RunaFonts.body(14)).foregroundStyle(runaTheme.subtle)
                .multilineTextAlignment(.center)
            if let ctaLabel, let onCta {
                RunaPillButton(label: ctaLabel, accent: true, action: onCta)
                    .padding(.top, RunaSpacing.sm)
            }
        }
    }
}

// MARK: - Offline (25)

/// Offline: the clouded moon. Only shown when there is nothing cached to render;
/// otherwise offline rides along as a `RunaSyncBanner` over the content.
struct RunaOfflineView: View {
    @Environment(\.runaTheme) private var runaTheme
    let onRetry: () -> Void

    var body: some View {
        RunaStateScaffold {
            CloudedMoon(diameter: 116)
            Text("雲が、月をかくしています。")
                .font(RunaFonts.heading(26)).foregroundStyle(runaTheme.heading)
                .multilineTextAlignment(.center)
            Text("接続がありません。\n綴った言葉は、端末に守られています。")
                .font(RunaFonts.body(14)).foregroundStyle(runaTheme.subtle)
                .multilineTextAlignment(.center)
            RunaPillButton(label: "もう一度ためす", accent: false, action: onRetry)
                .padding(.top, RunaSpacing.sm)
        }
    }
}

// MARK: - Error (27)

/// Error: the stumble emblem. Does not apologize — says what happened and how to go
/// on, in the world's voice. The auth variant overrides the copy + CTA.
struct RunaErrorView: View {
    @Environment(\.runaTheme) private var runaTheme
    var title: String = "すこし、つまずきました。"
    var message: String = "うまく読み込めませんでした。\nゆっくり、もう一度。"
    var ctaLabel: String = "やりなおす"
    let onCta: () -> Void

    var body: some View {
        RunaStateScaffold {
            StumbleEmblem(diameter: 116)
            Text(title)
                .font(RunaFonts.heading(26)).foregroundStyle(runaTheme.heading)
                .multilineTextAlignment(.center)
            Text(message)
                .font(RunaFonts.body(14)).foregroundStyle(runaTheme.subtle)
                .multilineTextAlignment(.center)
            RunaPillButton(label: ctaLabel, accent: true, action: onCta)
                .padding(.top, RunaSpacing.sm)
        }
    }
}

// MARK: - Failure dispatch (AppError → the right surface)

/// Maps a classified `AppError` to the right full-screen surface: offline → retry,
/// auth → re-authenticate (via the environment), server/unknown → retry.
struct RunaFailureView: View {
    @Environment(\.runaReauthenticate) private var reauthenticate
    let error: AppError
    let onRetry: () -> Void

    var body: some View {
        // SKIE exposes AppError as a bare protocol; dispatch by `as?` on the case classes.
        if error is AppErrorOffline {
            RunaOfflineView(onRetry: onRetry)
        } else if error is AppErrorAuth {
            RunaErrorView(
                title: "また、ここから。",
                message: "サインインの有効期限が切れました。",
                ctaLabel: "サインインし直す",
                onCta: reauthenticate
            )
        } else {
            RunaErrorView(onCta: onRetry) // server / unknown
        }
    }
}

// MARK: - Sync banner

/// The quiet status line shown over cached content (DoD #2): offline/error only —
/// a running sync is signalled by the screen's own refresh, so idle/syncing render
/// nothing.
struct RunaSyncBanner: View {
    @Environment(\.runaTheme) private var runaTheme
    let phase: SyncPhase

    var body: some View {
        if let text = bannerText {
            Text(text)
                .font(RunaFonts.body(13)).foregroundStyle(runaTheme.subtle)
                .multilineTextAlignment(.center)
                .frame(maxWidth: .infinity)
                .padding(.vertical, RunaSpacing.xs)
        }
    }

    private var bannerText: String? {
        switch phase {
        case .offline: return "オフライン。記録は端末に守られています。"
        case .error: return "同期に、すこしつまずいています。"
        default: return nil // idle / syncing stay silent
        }
    }
}

// MARK: - Shared pieces

/// Centered column the state surfaces share. Fills the space it is given (a caller
/// inside a scroll pins a height via `.frame(height:)`).
private struct RunaStateScaffold<Content: View>: View {
    @ViewBuilder var content: () -> Content

    var body: some View {
        VStack(spacing: RunaSpacing.sm) {
            content()
        }
        .frame(maxWidth: .infinity, maxHeight: .infinity)
        .padding(.horizontal, RunaSpacing.lg)
    }
}

/// A bordered pill CTA. `accent` uses the moonlight-pink accent, else a quiet subtle outline.
private struct RunaPillButton: View {
    @Environment(\.runaTheme) private var runaTheme
    let label: String
    var accent: Bool
    let action: () -> Void

    var body: some View {
        let tint = accent ? runaTheme.accent : runaTheme.subtle
        Button(action: action) {
            Text(label)
                .font(RunaFonts.body(16)).foregroundStyle(tint)
                .padding(.horizontal, 32).padding(.vertical, 14)
                .overlay(
                    RoundedRectangle(cornerRadius: 28)
                        .stroke(tint.opacity(0.7), lineWidth: 1)
                )
        }
        .buttonStyle(.plain)
    }
}

/// Three quiet dots. A staggered fade unless `animate` is false (reduced motion).
private struct RunaThreeDotProgress: View {
    @Environment(\.runaTheme) private var runaTheme
    var animate: Bool
    @State private var animating = false

    var body: some View {
        HStack(spacing: 10) {
            ForEach(0..<3, id: \.self) { index in
                Circle()
                    .fill(runaTheme.accent)
                    .frame(width: 8, height: 8)
                    .opacity(opacity(for: index))
                    .animation(
                        animate
                            ? .easeInOut(duration: 0.7).repeatForever(autoreverses: true).delay(Double(index) * 0.2)
                            : .default,
                        value: animating
                    )
            }
        }
        .onAppear { animating = animate }
    }

    private func opacity(for index: Int) -> Double {
        if !animate { return index == 0 ? 1.0 : 0.4 }
        return animating ? 1.0 : 0.3
    }
}
