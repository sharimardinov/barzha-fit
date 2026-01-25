import SwiftUI

struct SampleContentView: View {
    @StateObject private var auth = AuthState()
    @StateObject private var heartRate = HeartRateManager()
    @State private var profileJSON: String = ""
    @State private var errorText: String = ""

    private let loginURL = URL(string: "https://barzhafit.ru/login")!
    private let miniappBaseURL = URL(string: "https://barzhafit.ru/miniapp")!

    var body: some View {
        if let token = auth.token {
            TabView {
                profileTab(token: token)
                    .tabItem { Label("Profile", systemImage: "person") }

                workoutTab(token: token)
                    .tabItem { Label("Workout", systemImage: "timer") }
            }
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
        VStack(spacing: 12) {
            HStack(spacing: 8) {
                Text("Heart rate")
                    .font(.headline)
                Text(heartRate.bpm.map(String.init) ?? "n/a")
                    .font(.headline)
            }

            Text(heartRate.status)
                .font(.footnote)
                .foregroundColor(.secondary)

            WorkoutWebView(url: miniappURL(token: token), heartRate: heartRate)
                .frame(maxWidth: .infinity, maxHeight: .infinity)
                .layoutPriority(1)
        }
        .padding(.top, 12)
        .frame(maxWidth: .infinity, maxHeight: .infinity)
    }

    private func miniappURL(token: String) -> URL {
        var components = URLComponents(url: miniappBaseURL, resolvingAgainstBaseURL: false)
        components?.queryItems = [URLQueryItem(name: "token", value: token)]
        return components?.url ?? miniappBaseURL
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
