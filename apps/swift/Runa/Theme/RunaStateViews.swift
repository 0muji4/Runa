import SwiftUI
import Shared

/// The shared empty / offline / loading / error surfaces every feature renders.

// MARK: - Re-authenticate environment

private struct RunaReauthenticateKey: EnvironmentKey {
    static let defaultValue: () -> Void = {}
}

extension EnvironmentValues {
    /// The app-wide re-authenticate action (clears the session so auth drops to sign-in).
    var runaReauthenticate: () -> Void {
        get { self[RunaReauthenticateKey.self] }
        set { self[RunaReauthenticateKey.self] = newValue }
    }
}

// MARK: - UiState decoding (SKIE)

/// A page-level state decoded from the shared, generic `UiState<T>` via [runaDecode].
enum RunaUi<T> {
    case loading
    case empty
    case content(T, SyncPhase)
    case failure(AppError)
}

/// Decode a SKIE `UiState` emission into a native [RunaUi]. SKIE bridges the generic sealed
/// type to a bare protocol without `onEnum`, so cases are matched by `as?` casts.
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

/// Loading: glowing moon + three dots; reduced-motion safe.
struct RunaLoadingView: View {
    @Environment(\.runaTheme) private var runaTheme
    @Environment(\.accessibilityReduceMotion) private var reduceMotion
    var caption: String = L.stateLoadingCaption

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

/// Empty: new-moon emblem over per-feature copy.
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

/// Offline: only shown when there is nothing cached to render; otherwise `RunaSyncBanner`.
struct RunaOfflineView: View {
    @Environment(\.runaTheme) private var runaTheme
    let onRetry: () -> Void

    var body: some View {
        RunaStateScaffold {
            CloudedMoon(diameter: 116)
            Text(L.stateOfflineTitle)
                .font(RunaFonts.heading(26)).foregroundStyle(runaTheme.heading)
                .multilineTextAlignment(.center)
            Text(L.stateOfflineBody)
                .font(RunaFonts.body(14)).foregroundStyle(runaTheme.subtle)
                .multilineTextAlignment(.center)
            RunaPillButton(label: L.stateOfflineCta, accent: false, action: onRetry)
                .padding(.top, RunaSpacing.sm)
        }
    }
}

// MARK: - Error (27)

/// Error: stumble emblem; the auth variant overrides the copy + CTA.
struct RunaErrorView: View {
    @Environment(\.runaTheme) private var runaTheme
    var title: String = L.stateErrorTitle
    var message: String = L.stateErrorBody
    var ctaLabel: String = L.stateErrorCta
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

/// Maps an `AppError` to the right full-screen surface.
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
                title: L.stateAuthTitle,
                message: L.stateAuthBody,
                ctaLabel: L.stateAuthCta,
                onCta: reauthenticate
            )
        } else {
            RunaErrorView(onCta: onRetry) // server / unknown
        }
    }
}

// MARK: - Sync banner

/// Status line shown over cached content; offline/error only, idle/syncing render nothing.
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
        case .offline: return L.stateBannerOffline
        case .error: return L.stateBannerError
        default: return nil
        }
    }
}

// MARK: - Shared pieces

/// Centered column the state surfaces share; fills the space it is given.
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

/// A bordered pill CTA.
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

/// Three dots with a staggered fade unless `animate` is false (reduced motion).
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
