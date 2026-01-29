import SwiftUI
import UIKit
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
    }

    func makeUIView(context: Context) -> WKWebView {
        let config = WKWebViewConfiguration()
        config.userContentController.add(context.coordinator, name: "authComplete")
        config.userContentController.add(context.coordinator, name: "authDebug")
        config.defaultWebpagePreferences.allowsContentJavaScript = true
        config.preferences.javaScriptCanOpenWindowsAutomatically = true
        config.userContentController.addUserScript(makeDebugScript())

        let webView = WKWebView(frame: .zero, configuration: config)
        webView.navigationDelegate = context.coordinator
        webView.uiDelegate = context.coordinator
        let tap = UITapGestureRecognizer(target: context.coordinator, action: #selector(Coordinator.handleTap(_:)))
        tap.cancelsTouchesInView = false
        webView.addGestureRecognizer(tap)
        context.coordinator.setInitialURL(url)
        context.coordinator.noteNative("webview created")
        webView.load(URLRequest(url: url))
        return webView
    }

    func updateUIView(_ webView: WKWebView, context: Context) {
        context.coordinator.update(webView: webView, targetURL: url)
    }

    private func makeDebugScript() -> WKUserScript {
        let source = """
        (function() {
          if (window.__authDebugInstalled) { return; }
          window.__authDebugInstalled = true;
          function send(msg) {
            try {
              window.webkit && window.webkit.messageHandlers && window.webkit.messageHandlers.authDebug &&
                window.webkit.messageHandlers.authDebug.postMessage(String(msg));
            } catch (_) {}
          }
          window.addEventListener('error', function(e) {
            if (!e) return;
            send('js error: ' + (e.message || 'unknown'));
          });
          window.addEventListener('unhandledrejection', function(e) {
            var reason = e && e.reason ? (e.reason.message || String(e.reason)) : 'unknown';
            send('js promise: ' + reason);
          });
          document.addEventListener('DOMContentLoaded', function() {
            send('dom ready: ' + window.location.href);
            send('tg widget: ' + (typeof window.onTelegramAuth));
          });
          send('js bridge ready');
        })();
        """
        return WKUserScript(source: source, injectionTime: .atDocumentEnd, forMainFrameOnly: false)
    }

    final class Coordinator: NSObject, WKScriptMessageHandler, WKNavigationDelegate, WKUIDelegate {
        private let onAuth: (AuthPayload) -> Void
        private let onError: (String) -> Void
        private let onDebug: (String) -> Void
        private var didHandleTelegramAuth = false
        private var lastRequestedURL: URL?
        private var nativeLogged = false

        init(onAuth: @escaping (AuthPayload) -> Void, onError: @escaping (String) -> Void, onDebug: @escaping (String) -> Void) {
            self.onAuth = onAuth
            self.onError = onError
            self.onDebug = onDebug
        }

        func noteNative(_ message: String) {
            guard !nativeLogged else { return }
            nativeLogged = true
            onDebug("[native] \(message)")
        }

        @objc func handleTap(_ gesture: UITapGestureRecognizer) {
            let point = gesture.location(in: gesture.view)
            onDebug("[native] tap \(Int(point.x))x\(Int(point.y))")
        }

        func setInitialURL(_ url: URL) {
            lastRequestedURL = url
        }

        func update(webView: WKWebView, targetURL: URL) {
            let targetKey = urlKey(targetURL, includeQuery: true)
            if let last = lastRequestedURL, urlKey(last, includeQuery: true) == targetKey {
                return
            }
            if let current = webView.url {
                let currentBase = urlKey(current, includeQuery: false)
                let targetBase = urlKey(targetURL, includeQuery: false)
                if currentBase != targetBase {
                    // Don't interrupt external OAuth navigation.
                    return
                }
                if urlKey(current, includeQuery: true) == targetKey {
                    lastRequestedURL = targetURL
                    return
                }
            }
            lastRequestedURL = targetURL
            webView.load(URLRequest(url: targetURL))
        }

        private func urlKey(_ url: URL, includeQuery: Bool) -> String {
            guard let comps = URLComponents(url: url, resolvingAgainstBaseURL: false) else {
                return url.absoluteString
            }
            let scheme = (comps.scheme ?? "").lowercased()
            let host = (comps.host ?? "").lowercased()
            let port = comps.port.map { ":\($0)" } ?? ""
            var path = comps.path
            if path.hasSuffix("/") {
                path.removeLast()
            }
            var key = "\(scheme)://\(host)\(port)\(path)"
            if includeQuery {
                let query = comps.percentEncodedQuery ?? ""
                key += "?\(query)"
            }
            return key
        }

        func userContentController(_ userContentController: WKUserContentController, didReceive message: WKScriptMessage) {
            guard message.name == "authComplete" else {
                if message.name == "authDebug" {
                    onDebug("[web] \(message.body)")
                }
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
            if let url = webView.url?.absoluteString {
                onDebug("[nav] finished: \(url)")
            }
            tryHandleTelegramJSON(in: webView)
        }

        func webView(_ webView: WKWebView, createWebViewWith configuration: WKWebViewConfiguration, for navigationAction: WKNavigationAction, windowFeatures: WKWindowFeatures) -> WKWebView? {
            if navigationAction.targetFrame == nil, let url = navigationAction.request.url {
                webView.load(URLRequest(url: url))
            }
            return nil
        }

        func webView(_ webView: WKWebView, decidePolicyFor navigationAction: WKNavigationAction, decisionHandler: @escaping (WKNavigationActionPolicy) -> Void) {
            if let url = navigationAction.request.url {
                let isMain = navigationAction.targetFrame?.isMainFrame ?? false
                onDebug("[nav] action: \(url.absoluteString) main=\(isMain)")
                if let scheme = url.scheme?.lowercased() {
                if scheme == "tg" || scheme == "telegram" {
                    if UIApplication.shared.canOpenURL(url) {
                        UIApplication.shared.open(url, options: [:], completionHandler: nil)
                        onDebug("[nav] open tg://")
                    } else {
                        onDebug("[nav] tg:// not supported")
                    }
                    decisionHandler(.cancel)
                    return
                }
            }
            }
            decisionHandler(.allow)
        }

        func webView(_ webView: WKWebView, didFailProvisionalNavigation navigation: WKNavigation!, withError error: Error) {
            onDebug("[nav] failed: \(error.localizedDescription)")
        }

        func webView(_ webView: WKWebView, didStartProvisionalNavigation navigation: WKNavigation!) {
            if let url = webView.url?.absoluteString {
                onDebug("[nav] start: \(url)")
            }
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
