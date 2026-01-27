import SwiftUI
import WebKit

struct AuthWebView: UIViewRepresentable {
    let url: URL
    let onAuth: (AuthPayload) -> Void
    let onError: (String) -> Void
    let onDebug: (String) -> Void

    init(url: URL, onAuth: @escaping (AuthPayload) -> Void, onError: @escaping (String) -> Void = { _ in }, onDebug: @escaping (String) -> Void = { _ in }) {
        self.url = url
        self.onAuth = onAuth
        self.onError = onError
        self.onDebug = onDebug
    }

    func makeCoordinator() -> Coordinator {
        Coordinator(onAuth: onAuth, onError: onError, onDebug: onDebug)
    }

    func makeUIView(context: Context) -> WKWebView {
        let config = WKWebViewConfiguration()
        config.userContentController.add(context.coordinator, name: "authComplete")
        config.defaultWebpagePreferences.allowsContentJavaScript = true
        config.preferences.javaScriptCanOpenWindowsAutomatically = true

        let webView = WKWebView(frame: .zero, configuration: config)
        webView.navigationDelegate = context.coordinator
        webView.uiDelegate = context.coordinator
        webView.load(URLRequest(url: url))
        return webView
    }

    func updateUIView(_ webView: WKWebView, context: Context) {
        if webView.url != url {
            webView.load(URLRequest(url: url))
        }
    }

    final class Coordinator: NSObject, WKScriptMessageHandler, WKNavigationDelegate, WKUIDelegate {
        private let onAuth: (AuthPayload) -> Void
        private let onError: (String) -> Void
        private let onDebug: (String) -> Void
        private var didHandleTelegramAuth = false

        init(onAuth: @escaping (AuthPayload) -> Void, onError: @escaping (String) -> Void, onDebug: @escaping (String) -> Void) {
            self.onAuth = onAuth
            self.onError = onError
            self.onDebug = onDebug
        }

        func userContentController(_ userContentController: WKUserContentController, didReceive message: WKScriptMessage) {
            guard message.name == "authComplete" else {
                return
            }
            guard let payload = AuthPayload.fromMessageBody(message.body) else {
                onError("invalid_auth_payload")
                return
            }
            DispatchQueue.main.async {
                self.onAuth(payload)
            }
        }

        func webView(_ webView: WKWebView, didFail navigation: WKNavigation!, withError error: Error) {
            onError(error.localizedDescription)
        }

        func webView(_ webView: WKWebView, didFinish navigation: WKNavigation!) {
            tryHandleTelegramJSON(in: webView)
        }

        func webView(_ webView: WKWebView, createWebViewWith configuration: WKWebViewConfiguration, for navigationAction: WKNavigationAction, windowFeatures: WKWindowFeatures) -> WKWebView? {
            if navigationAction.targetFrame == nil, let url = navigationAction.request.url {
                webView.load(URLRequest(url: url))
            }
            return nil
        }

        private func tryHandleTelegramJSON(in webView: WKWebView) {
            guard !didHandleTelegramAuth else { return }
            guard let host = webView.url?.host?.lowercased(), host == "oauth.telegram.org" else { return }

            webView.evaluateJavaScript("window.location.hash") { hashResult, _ in
                if let hash = hashResult as? String, let user = Self.userFromHash(hash) {
                    self.didHandleTelegramAuth = true
                    self.onDebug("hash ok")
                    self.submitTelegramAuth(user)
                    return
                }

                webView.evaluateJavaScript("document.body && document.body.innerText") { result, _ in
                    guard let text = result as? String else { return }
                    self.onDebug("hash miss; body len=\(text.count)")
                    guard let user = Self.userFromBody(text) else {
                        self.onDebug("body json parse miss")
                        return
                    }
                    self.didHandleTelegramAuth = true
                    self.onDebug("body json ok")
                    self.submitTelegramAuth(user)
                }
            }
        }

        private static func userFromHash(_ hash: String) -> [String: Any]? {
            let prefix = "#tgAuthResult="
            guard hash.hasPrefix(prefix) else { return nil }
            let encoded = String(hash.dropFirst(prefix.count))
            guard let data = Data(base64Encoded: encoded) else { return nil }
            return (try? JSONSerialization.jsonObject(with: data)) as? [String: Any]
        }

        private static func userFromBody(_ text: String) -> [String: Any]? {
            let trimmed = text.trimmingCharacters(in: .whitespacesAndNewlines)
            if trimmed.first == "{", trimmed.last == "}" {
                if let data = trimmed.data(using: .utf8),
                   let json = (try? JSONSerialization.jsonObject(with: data)) as? [String: Any] {
                    return json
                }
            }

            guard let start = trimmed.firstIndex(of: "{"),
                  let end = trimmed.lastIndex(of: "}") else {
                return nil
            }
            let candidate = String(trimmed[start...end])
            guard let data = candidate.data(using: .utf8) else { return nil }
            return (try? JSONSerialization.jsonObject(with: data)) as? [String: Any]
        }

        private func submitTelegramAuth(_ user: [String: Any]) {
            guard let url = URL(string: "https://barzhafit.ru/auth/telegram") else { return }
            var request = URLRequest(url: url)
            request.httpMethod = "POST"
            request.addValue("application/json", forHTTPHeaderField: "Content-Type")
            guard let body = try? JSONSerialization.data(withJSONObject: user, options: []) else { return }
            request.httpBody = body

            URLSession.shared.dataTask(with: request) { data, _, error in
                if let error = error {
                    self.onDebug("auth error: \(error.localizedDescription)")
                    self.onError(error.localizedDescription)
                    return
                }
                guard let data,
                      let json = (try? JSONSerialization.jsonObject(with: data)) as? [String: Any],
                      let ok = json["ok"] as? Bool,
                      ok,
                      let payloadData = json["data"] as? [String: Any],
                      let token = payloadData["token"] as? String,
                      !token.isEmpty else {
                    self.onDebug("auth failed resp")
                    self.onError("telegram_auth_failed")
                    return
                }

                let userId = (payloadData["user_id"] as? NSNumber)?.int64Value ?? 0
                let username = payloadData["username"] as? String
                var expiresAt: Date?
                if let ts = (payloadData["expires_at"] as? NSNumber)?.doubleValue {
                    expiresAt = Date(timeIntervalSince1970: ts)
                }

                let payload = AuthPayload(token: token, userId: userId, username: username, expiresAt: expiresAt)
                DispatchQueue.main.async {
                    self.onDebug("auth ok")
                    self.onAuth(payload)
                }
            }.resume()
        }
    }
}
