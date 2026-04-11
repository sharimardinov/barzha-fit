import SwiftUI

struct SampleContentView: View {
    @StateObject private var auth = AuthState()
    @StateObject private var heartRate = HeartRateManager()
    @State private var errorText: String = ""

    private let loginURL = AppConfig.loginURL
    private let miniappBaseURL = AppConfig.miniAppURL

    var body: some View {
        Group {
            if let token = auth.token {
                WorkoutWebView(url: miniappURL(token: token), heartRate: heartRate) {
                    errorText = ""
                    auth.clear()
                    heartRate.stop()
                }
                .ignoresSafeArea()
            } else {
                ZStack(alignment: .bottom) {
                    AuthWebView(url: loginURL) { payload in
                        errorText = ""
                        auth.save(payload: payload)
                    } onError: { error in
                        errorText = error
                    }

                    if !errorText.isEmpty {
                        Text(errorText)
                            .font(.footnote)
                            .foregroundColor(.red)
                            .padding(12)
                            .background(.white.opacity(0.9))
                            .cornerRadius(8)
                            .padding(.bottom, 24)
                    }
                }
                .ignoresSafeArea()
            }
        }
        .onChange(of: auth.token) { newValue in
            if newValue?.isEmpty != false {
                heartRate.stop()
            }
        }
    }

    private func miniappURL(token: String) -> URL {
        var components = URLComponents(url: miniappBaseURL, resolvingAgainstBaseURL: false)
        components?.queryItems = [URLQueryItem(name: "token", value: token)]
        return components?.url ?? miniappBaseURL
    }
}
