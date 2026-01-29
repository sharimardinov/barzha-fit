import SwiftUI

struct SampleContentView: View {
    @StateObject private var auth = AuthState()
    @StateObject private var heartRate = HeartRateManager()
    @State private var profileJSON: String = ""
    @State private var errorText: String = ""
    @State private var debugText: String = ""

    private let loginURL = URL(string: "https://barzhafit.ru/login")!
    private let miniappBaseURL = URL(string: "https://barzhafit.ru/miniapp")!

    var body: some View {
        if let token = auth.token {
            workoutTab(token: token)
            .onChange(of: auth.token) { newValue in
                if newValue == nil {
                    heartRate.stop()
                }
            }
        } else {
            ZStack(alignment: .bottom) {
                AuthWebView(url: loginURL) { payload in
                    errorText = ""
                    auth.save(payload: payload)
                } onError: { error in
                    errorText = error
                } onDebug: { message in
                    debugText = appendDebugLine(debugText, message)
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

                if !debugText.isEmpty {
                    Text(debugText)
                        .font(.caption2)
                        .foregroundColor(.white)
                        .padding(8)
                        .background(Color.black.opacity(0.6))
                        .cornerRadius(8)
                        .padding(.bottom, 80)
                }
            }
            .ignoresSafeArea()
        }
    }

    @ViewBuilder
    private func profileTab(token: String) -> some View {
        VStack(spacing: 12) {
            if let username = auth.username, !username.isEmpty {
                Text("Hi, @\(username)")
                    .font(.headline)
            } else {
                Text("Signed in")
                    .font(.headline)
            }

            if !profileJSON.isEmpty {
                Text(profileJSON)
                    .font(.footnote)
                    .foregroundColor(.secondary)
                    .multilineTextAlignment(.leading)
            }

            if !errorText.isEmpty {
                Text(errorText)
                    .font(.footnote)
                    .foregroundColor(.red)
            }

            Button("Load profile") {
                loadProfile(token: token)
            }

            Button("Log out") {
                auth.clear()
                profileJSON = ""
                heartRate.stop()
            }
        }
        .padding()
    }

    @ViewBuilder
    private func workoutTab(token: String) -> some View {
        WorkoutWebView(url: miniappURL(token: token), heartRate: heartRate, onLogout: {
            errorText = ""
            auth.clear()
        })
            .ignoresSafeArea()
    }

    private func miniappURL(token: String) -> URL {
        var components = URLComponents(url: miniappBaseURL, resolvingAgainstBaseURL: false)
        components?.queryItems = [URLQueryItem(name: "token", value: token)]
        return components?.url ?? miniappBaseURL
    }

    private func appendDebugLine(_ existing: String, _ message: String) -> String {
        let line = message.trimmingCharacters(in: .whitespacesAndNewlines)
        if line.isEmpty {
            return existing
        }
        var lines = existing.split(separator: "\n").map(String.init)
        lines.append(line)
        if lines.count > 8 {
            lines = Array(lines.suffix(8))
        }
        return lines.joined(separator: "\n")
    }

    private func loadProfile(token: String) {
        errorText = ""
        Task {
            do {
                var request = URLRequest(url: URL(string: "https://barzhafit.ru/api/profile/get")!)
                request.httpMethod = "POST"
                request.addValue("application/json", forHTTPHeaderField: "Content-Type")
                request.addValue("Bearer \(token)", forHTTPHeaderField: "Authorization")
                request.httpBody = Data("{}".utf8)

                let (data, _) = try await URLSession.shared.data(for: request)
                profileJSON = String(data: data, encoding: .utf8) ?? ""
            } catch {
                errorText = error.localizedDescription
            }
        }
    }
}
