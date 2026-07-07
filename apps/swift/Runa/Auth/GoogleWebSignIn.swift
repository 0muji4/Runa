import AuthenticationServices
import UIKit

/// Sign in with Google on iOS using the system `ASWebAuthenticationSession` — no
/// third-party SDK, so the app stays self-contained.
///
/// It runs Google's OpenID Connect implicit flow, returning an `id_token` in the
/// callback URL fragment. The token's audience is the iOS OAuth **client ID**
/// (`GIDClientID` in Info.plist); the backend must list that client ID in
/// `GOOGLE_CLIENT_IDS`. The shared `AuthRepository.loginGoogle` posts the token.
///
/// Requires `GIDClientID` in Info.plist and its reversed-client-id URL scheme in
/// `CFBundleURLTypes` (see README). Absent config surfaces a friendly error.
final class GoogleWebSignIn: NSObject, ASWebAuthenticationPresentationContextProviding {

    private var session: ASWebAuthenticationSession?

    /// Whether `GIDClientID` is configured.
    static var isConfigured: Bool { clientID?.isEmpty == false }

    private static var clientID: String? {
        Bundle.main.object(forInfoDictionaryKey: "GIDClientID") as? String
    }

    func signIn(onIdToken: @escaping (String) -> Void, onError: @escaping (String) -> Void) {
        guard let clientID = Self.clientID, !clientID.isEmpty else {
            onError("この方法は現在利用できません（設定が必要です）。")
            return
        }

        // Reversed client ID is the callback scheme, e.g.
        // com.googleusercontent.apps.<client-id-without-suffix>.
        let suffix = ".apps.googleusercontent.com"
        let reversed = "com.googleusercontent.apps." + clientID.replacingOccurrences(of: suffix, with: "")
        let redirectURI = reversed + ":/oauth2redirect"
        let nonce = UUID().uuidString

        var components = URLComponents(string: "https://accounts.google.com/o/oauth2/v2/auth")!
        components.queryItems = [
            URLQueryItem(name: "client_id", value: clientID),
            URLQueryItem(name: "redirect_uri", value: redirectURI),
            URLQueryItem(name: "response_type", value: "id_token"),
            URLQueryItem(name: "scope", value: "openid email profile"),
            URLQueryItem(name: "nonce", value: nonce),
        ]

        guard let url = components.url else {
            onError("サインインURLの生成に失敗しました。")
            return
        }

        let session = ASWebAuthenticationSession(url: url, callbackURLScheme: reversed) { callbackURL, error in
            if let error = error {
                onError(error.localizedDescription)
                return
            }
            guard let callbackURL = callbackURL,
                  let idToken = Self.idToken(fromFragment: callbackURL.fragment) else {
                onError("id_token を取得できませんでした。")
                return
            }
            onIdToken(idToken)
        }
        session.presentationContextProvider = self
        session.prefersEphemeralWebBrowserSession = false
        self.session = session
        session.start()
    }

    private static func idToken(fromFragment fragment: String?) -> String? {
        guard let fragment else { return nil }
        for pair in fragment.split(separator: "&") {
            let kv = pair.split(separator: "=", maxSplits: 1)
            if kv.count == 2, kv[0] == "id_token" {
                return String(kv[1]).removingPercentEncoding ?? String(kv[1])
            }
        }
        return nil
    }

    func presentationAnchor(for session: ASWebAuthenticationSession) -> ASPresentationAnchor {
        UIApplication.shared.connectedScenes
            .compactMap { $0 as? UIWindowScene }
            .flatMap { $0.windows }
            .first { $0.isKeyWindow } ?? ASPresentationAnchor()
    }
}
