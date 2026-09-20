import SwiftUI
import Shared

/// Root auth gate: `Restoring` → splash, `Authenticated` → the tab body, anything else →
/// the onboarding / sign-in flow.
struct RootView: View {
    @StateObject private var auth = AuthObservable()
    @StateObject private var pending = PendingRouteObservable()

    var body: some View {
        Group {
            if let state = auth.state {
                switch onEnum(of: state) {
                case .restoring:
                    SplashView()
                case .authenticated(let authenticated):
                    ContentView(
                        displayName: authenticated.user.displayName,
                        onSignOut: { auth.logout() }
                    )
                    // App-wide re-authenticate action consumed by RunaErrorView's auth CTA.
                    .environment(\.runaReauthenticate, { auth.logout() })
                    .environmentObject(pending)
                default:
                    AuthFlowView(auth: auth, state: state)
                        // A tap that outlived the session must not fire after the next sign-in.
                        .onAppear { pending.consume() }
                }
            } else {
                SplashView()
            }
        }
    }
}
