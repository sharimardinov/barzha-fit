import SwiftUI
import WebKit

struct AuthWebView: UIViewRepresentable {
    let url: URL
    let onAuth: (AuthPayload) -> Void
    let onError: (String) -> Void

    init(url: URL, onAuth: @escaping (AuthPayload) -> Void, onError: @escaping (String) -> Void = { _ in }) {
        self.url = url
        self.onAuth = onAuth
        self.onError = onError
    }

    func makeCoordinator() -> Coordinator {
        Coordinator(onAuth: onAuth, onError: onError)
    }

    func makeUIView(context: Context) -> WKWebView {
        let config = WKWebViewConfiguration()
        config.userContentController.add(context.coordinator, name: "authComplete")
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
        private let onAuth: (AuthPayload) -> Void
        private let onError: (String) -> Void

        init(onAuth: @escaping (AuthPayload) -> Void, onError: @escaping (String) -> Void) {
            self.onAuth = onAuth
            self.onError = onError
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
    }
}
