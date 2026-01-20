import SwiftUI

struct SampleContentView: View {
    @StateObject private var auth = AuthState()
    @State private var profileJSON: String = ""
    @State private var errorText: String = ""

    private let loginURL = URL(string: "https://barzhafit.ru/login")!

    var body: some View {
        if let token = auth.token {
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
                }
            }
            .padding()
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
