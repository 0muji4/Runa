import SwiftUI
import Shared

/// Root auth gate. Subscribes to the shared `AuthObservable` and switches the
/// whole app between the startup splash, the unauthenticated flow, and the tab
/// body:
///  - `AuthState.Restoring`     → splash (checking the stored session)
///  - `AuthState.Authenticated` → the tabbed app, greeting the /me display name
///  - anything else             → onboarding → sign-in
///
/// Signing out from Settings flips the shared state back to unauthenticated, so
/// this gate returns to the sign-in flow automatically.
struct RootView: View {
    @StateObject private var auth = AuthObservable()

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
                default:
                    AuthFlowView(auth: auth, state: state)
                }
            } else {
                SplashView()
            }
        }
        // The color scheme now follows the selected theme, applied by ThemedRoot at
        // the app root (see RunaApp).
    }
}
