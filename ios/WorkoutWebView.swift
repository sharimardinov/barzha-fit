import SwiftUI
import UIKit
import WebKit

struct WorkoutWebView: UIViewRepresentable {
    let url: URL
    @ObservedObject var heartRate: HeartRateManager
    @Binding var activeTab: String
    let onLogout: () -> Void
    let onTabChange: (String) -> Void

    init(url: URL, heartRate: HeartRateManager, activeTab: Binding<String>, onLogout: @escaping () -> Void = {}, onTabChange: @escaping (String) -> Void = { _ in }) {
        self.url = url
        self._heartRate = ObservedObject(wrappedValue: heartRate)
        self._activeTab = activeTab
        self.onLogout = onLogout
        self.onTabChange = onTabChange
    }

    func makeCoordinator() -> Coordinator {
        Coordinator(heartRate: heartRate, onLogout: onLogout, onTabChange: onTabChange)
    }

    func makeUIView(context: Context) -> WKWebView {
        let config = WKWebViewConfiguration()
        config.userContentController.add(context.coordinator, name: "workoutTimer")
        config.userContentController.add(context.coordinator, name: "authState")
        config.userContentController.add(context.coordinator, name: "nativeNav")
        config.defaultWebpagePreferences.allowsContentJavaScript = true
        config.userContentController.addUserScript(makeHideNavScript())

        let webView = WKWebView(frame: .zero, configuration: config)
        webView.navigationDelegate = context.coordinator
        context.coordinator.attach(webView: webView)
        context.coordinator.setInitialURL(url)
        webView.load(URLRequest(url: url))
        return webView
    }

    func updateUIView(_ webView: WKWebView, context: Context) {
        context.coordinator.update(webView: webView, targetURL: url)
        context.coordinator.syncTab(activeTab)
    }

    private func makeHideNavScript() -> WKUserScript {
        let source = """
        (function() {
          if (window.__nativeNavHidden) { return; }
          window.__nativeNavHidden = true;
          var style = document.createElement('style');
          style.textContent = '.nav{display:none !important}';
          document.head && document.head.appendChild(style);
        })();
        """
        return WKUserScript(source: source, injectionTime: .atDocumentEnd, forMainFrameOnly: false)
    }

    final class Coordinator: NSObject, WKScriptMessageHandler, WKNavigationDelegate {
        private let heartRate: HeartRateManager
        private let onLogout: () -> Void
        private let onTabChange: (String) -> Void
        private var lastRequestedURL: URL?
        private weak var webView: WKWebView?
        private var stepsObserver: NSObjectProtocol?
        private var lastTabSent: String?
        private var suppressNextTabSend = false

        init(heartRate: HeartRateManager, onLogout: @escaping () -> Void, onTabChange: @escaping (String) -> Void) {
            self.heartRate = heartRate
            self.onLogout = onLogout
            self.onTabChange = onTabChange
            super.init()
            stepsObserver = NotificationCenter.default.addObserver(forName: .stepsDidUpdate, object: nil, queue: .main) { [weak self] note in
                self?.handleStepsUpdate(note)
            }
        }

        deinit {
            if let stepsObserver {
                NotificationCenter.default.removeObserver(stepsObserver)
            }
        }

        func attach(webView: WKWebView) {
            self.webView = webView
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

        private func handleStepsUpdate(_ notification: Notification) {
            guard let webView else { return }
            guard let info = notification.userInfo else { return }
            let steps = info["steps"] as? Int ?? 0
            let distance = info["distance"] as? Double ?? 0
            let kcal = info["kcal"] as? Double ?? 0
            guard let payload = try? JSONSerialization.data(withJSONObject: [
                "steps": steps,
                "distance": distance,
                "kcal": kcal,
            ]) else { return }
            guard let json = String(data: payload, encoding: .utf8) else { return }
            let js = "window.dispatchEvent(new CustomEvent('nativeSteps', {detail: \(json)}));"
            webView.evaluateJavaScript(js, completionHandler: nil)
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
            if message.name == "nativeNav" {
                handleNativeNav(message)
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

        private func handleNativeNav(_ message: WKScriptMessage) {
            var tab: String?
            if let dict = message.body as? [String: Any] {
                tab = dict["tab"] as? String
            } else if let text = message.body as? String {
                tab = text
            }
            guard let tab, !tab.isEmpty else { return }
            if tab == lastTabSent { return }
            suppressNextTabSend = true
            Task { @MainActor in
                onTabChange(tab)
            }
        }

        func syncTab(_ tab: String) {
            guard let webView else { return }
            if suppressNextTabSend {
                suppressNextTabSend = false
                lastTabSent = tab
                return
            }
            if tab == lastTabSent { return }
            lastTabSent = tab
            let safeTab = tab.replacingOccurrences(of: "'", with: "\\'")
            let js = "window.dispatchEvent(new CustomEvent('nativeTab', {detail: {tab: '\(safeTab)'}}));"
            webView.evaluateJavaScript(js, completionHandler: nil)
        }

        func webView(_ webView: WKWebView, didFail navigation: WKNavigation!, withError error: Error) {
            // Keep web view errors local; heart rate manager should be independent.
        }
    }
}
