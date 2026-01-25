import SwiftUI
import WebKit

struct WorkoutWebView: UIViewRepresentable {
    let url: URL
    @ObservedObject var heartRate: HeartRateManager

    func makeCoordinator() -> Coordinator {
        Coordinator(heartRate: heartRate)
    }

    func makeUIView(context: Context) -> WKWebView {
        let config = WKWebViewConfiguration()
        config.userContentController.add(context.coordinator, name: "workoutTimer")
        config.defaultWebpagePreferences.allowsContentJavaScript = true

        let webView = WKWebView(frame: .zero, configuration: config)
        webView.navigationDelegate = context.coordinator
        webView.load(URLRequest(url: url))
        return webView
    }

    func updateUIView(_ webView: WKWebView, context: Context) {
        if webView.url != url {
            webView.load(URLRequest(url: url))
        }
    }

    final class Coordinator: NSObject, WKScriptMessageHandler, WKNavigationDelegate {
        private let heartRate: HeartRateManager

        init(heartRate: HeartRateManager) {
            self.heartRate = heartRate
        }

        func userContentController(_ userContentController: WKUserContentController, didReceive message: WKScriptMessage) {
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

        func webView(_ webView: WKWebView, didFail navigation: WKNavigation!, withError error: Error) {
            // Keep web view errors local; heart rate manager should be independent.
        }
    }
}
