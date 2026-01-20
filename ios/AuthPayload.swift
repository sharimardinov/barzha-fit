import Foundation

struct AuthPayload {
    let token: String
    let userId: Int64
    let username: String?
    let expiresAt: Date?

    static func fromMessageBody(_ body: Any) -> AuthPayload? {
        guard let dict = body as? [String: Any] else {
            return nil
        }
        guard let token = dict["token"] as? String, !token.isEmpty else {
            return nil
        }

        let userId = (dict["userId"] as? NSNumber)?.int64Value ?? 0
        let username = dict["username"] as? String

        var expiresAt: Date?
        if let ts = (dict["expiresAt"] as? NSNumber)?.doubleValue {
            expiresAt = Date(timeIntervalSince1970: ts)
        }

        return AuthPayload(token: token, userId: userId, username: username, expiresAt: expiresAt)
    }
}
