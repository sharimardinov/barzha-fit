import Foundation

@MainActor
final class AuthState: ObservableObject {
    @Published var token: String?
    @Published var userId: Int64?
    @Published var username: String?

    private let userIdKey = "auth.userId"
    private let usernameKey = "auth.username"

    init() {
        token = KeychainStore.loadToken()
        if let stored = UserDefaults.standard.object(forKey: userIdKey) as? NSNumber {
            userId = stored.int64Value
        }
        username = UserDefaults.standard.string(forKey: usernameKey)
    }

    func save(payload: AuthPayload) {
        token = payload.token
        userId = payload.userId
        username = payload.username
        KeychainStore.saveToken(payload.token)
        UserDefaults.standard.set(NSNumber(value: payload.userId), forKey: userIdKey)
        if let username = payload.username {
            UserDefaults.standard.set(username, forKey: usernameKey)
        }
    }

    func clear() {
        token = nil
        userId = nil
        username = nil
        KeychainStore.deleteToken()
        UserDefaults.standard.removeObject(forKey: userIdKey)
        UserDefaults.standard.removeObject(forKey: usernameKey)
    }
}
