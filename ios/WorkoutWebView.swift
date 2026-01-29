import SwiftUI
import WebKit

struct WorkoutWebView: UIViewRepresentable {
    let url: URL
    @ObservedObject var heartRate: HeartRateManager
    let onLogout: () -> Void = {}

    func makeCoordinator() -> Coordinator {
        Coordinator(heartRate: heartRate, onLogout: onLogout)
    }

    func makeUIView(context: Context) -> WKWebView {
        let config = WKWebViewConfiguration()
        config.userContentController.add(context.coordinator, name: "workoutTimer")
        config.userContentController.add(context.coordinator, name: "authState")
        config.defaultWebpagePreferences.allowsContentJavaScript = true

        let webView = WKWebView(frame: .zero, configuration: config)
        webView.navigationDelegate = context.coordinator
        context.coordinator.setInitialURL(url)
        webView.load(URLRequest(url: url))
        return webView
    }

    func updateUIView(_ webView: WKWebView, context: Context) {
        context.coordinator.update(webView: webView, targetURL: url)
    }

    final class Coordinator: NSObject, WKScriptMessageHandler, WKNavigationDelegate {
        private let heartRate: HeartRateManager
        private let onLogout: () -> Void
        private var lastRequestedURL: URL?

        init(heartRate: HeartRateManager, onLogout: @escaping () -> Void) {
            self.heartRate = heartRate
            self.onLogout = onLogout
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
                    // Don't interrupt external navigation.
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
            if message.name == "authState" {
                handleAuthState(message)
                return
            }
            guard message.name == "workoutTimer" else { return }
            let action: String?
            if let dict = message.body as? [String: Any] {
                action = dict["action"] as? String
            } else if let text = message.body as? String {
                action = text
            } else {
                action = nil
            }
            guard let action else { return }
            if action == "start" {
                Task { @MainActor in
                    heartRate.start()
                }
            } else if action == "stop" {
                Task { @MainActor in
                    heartRate.stop()
                }
            }
        }

        private func handleAuthState(_ message: WKScriptMessage) {
            let action: String?
            if let dict = message.body as? [String: Any] {
                action = dict["action"] as? String
            } else if let text = message.body as? String {
                action = text
            } else {
                action = nil
            }
            guard let action, action == "logout" || action == "auth_required" else { return }
            Task { @MainActor in
                onLogout()
            }
        }

        func webView(_ webView: WKWebView, didFail navigation: WKNavigation!, withError error: Error) {
            // Keep web view errors local; heart rate manager should be independent.
        }
    }
}
