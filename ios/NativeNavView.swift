import SwiftUI

struct NativeNavView: View {
    @Binding var activeTab: String
    @Namespace private var highlight

    private let tabs: [TabItem] = [
        .init(id: "today", label: "Сегодня", systemImage: "house.fill"),
        .init(id: "workout", label: "Тренировка", systemImage: "timer"),
        .init(id: "meals", label: "Еда", systemImage: "fork.knife"),
        .init(id: "profile", label: "Профиль", systemImage: "person.fill")
    ]

    var body: some View {
        HStack(spacing: 6) {
            ForEach(tabs) { tab in
                Button {
                    activeTab = tab.id
                } label: {
                    VStack(spacing: 4) {
                        Image(systemName: tab.systemImage)
                            .font(.system(size: 18, weight: .semibold))
                        Text(tab.label)
                            .font(.system(size: 10, weight: .semibold))
                            .lineLimit(1)
                    }
                    .frame(maxWidth: .infinity)
                    .padding(.vertical, 10)
                    .foregroundColor(activeTab == tab.id ? .white : Color.black.opacity(0.6))
                    .background(
                        ZStack {
                            if activeTab == tab.id {
                                Capsule()
                                    .fill(Color(hex: "#ff033e"))
                                    .matchedGeometryEffect(id: "pill", in: highlight)
                            }
                        }
                    )
                }
                .buttonStyle(.plain)
            }
        }
        .padding(6)
        .background(.white.opacity(0.92))
        .clipShape(Capsule())
        .overlay(
            Capsule().stroke(Color.black.opacity(0.08), lineWidth: 1)
        )
        .shadow(color: Color.black.opacity(0.12), radius: 18, x: 0, y: 8)
    }
}

private struct TabItem: Identifiable {
    let id: String
    let label: String
    let systemImage: String
}

private extension Color {
    init(hex: String) {
        var hex = hex.trimmingCharacters(in: CharacterSet.alphanumerics.inverted)
        var int: UInt64 = 0
        Scanner(string: hex).scanHexInt64(&int)
        let r = Double((int >> 16) & 0xff) / 255
        let g = Double((int >> 8) & 0xff) / 255
        let b = Double(int & 0xff) / 255
        self.init(red: r, green: g, blue: b)
    }
}
