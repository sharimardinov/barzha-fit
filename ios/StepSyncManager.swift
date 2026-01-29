import Combine
import Foundation
import HealthKit

@MainActor
final class StepSyncManager: NSObject, ObservableObject {
    @Published private(set) var lastSteps: Int = 0
    @Published private(set) var status: String = "idle"

    private let healthStore = HKHealthStore()
    private let stepType = HKQuantityType.quantityType(forIdentifier: .stepCount)!
    private var timer: Timer?
    private var token: String?
    private var isActive = false
    private var lastSyncedSteps: Int?

    func start(token: String) {
        self.token = token
        guard HKHealthStore.isHealthDataAvailable() else {
            status = "unavailable"
            return
        }
        if isActive {
            refreshStepsAndSync()
            return
        }
        isActive = true
        status = "requesting_access"
        healthStore.requestAuthorization(toShare: [], read: [stepType]) { [weak self] ok, error in
            Task { @MainActor in
                guard let self else { return }
                if !ok || error != nil {
                    self.status = "unauthorized"
                    self.isActive = false
                    return
                }
                self.status = "active"
                self.refreshStepsAndSync()
                self.startTimer()
            }
        }
    }

    func stop() {
        timer?.invalidate()
        timer = nil
        token = nil
        isActive = false
        lastSyncedSteps = nil
        status = "stopped"
        lastSteps = 0
    }

    private func startTimer() {
        timer?.invalidate()
        timer = Timer.scheduledTimer(withTimeInterval: 300, repeats: true) { [weak self] _ in
            Task { @MainActor in
                self?.refreshStepsAndSync()
            }
        }
    }

    private func refreshStepsAndSync() {
        let startOfDay = Calendar.current.startOfDay(for: Date())
        let predicate = HKQuery.predicateForSamples(withStart: startOfDay, end: Date(), options: .strictStartDate)
        let query = HKStatisticsQuery(quantityType: stepType, quantitySamplePredicate: predicate, options: .cumulativeSum) { [weak self] _, result, error in
            Task { @MainActor in
                guard let self else { return }
                if error != nil {
                    self.status = "error"
                    return
                }
                let steps = Int(result?.sumQuantity()?.doubleValue(for: .count()) ?? 0)
                self.lastSteps = steps
                self.status = "active"
                self.syncIfNeeded(steps)
            }
        }
        healthStore.execute(query)
    }

    private func syncIfNeeded(_ steps: Int) {
        guard let token else { return }
        if let last = lastSyncedSteps, last == steps {
            return
        }
        lastSyncedSteps = steps
        sendSteps(steps, token: token)
    }

    private func sendSteps(_ steps: Int, token: String) {
        guard let url = URL(string: "https://barzhafit.ru/api/steps/set") else { return }
        var request = URLRequest(url: url)
        request.httpMethod = "POST"
        request.addValue("application/json", forHTTPHeaderField: "Content-Type")
        request.addValue("Bearer \(token)", forHTTPHeaderField: "Authorization")
        request.httpBody = Data("{\"steps\":\(steps)}".utf8)
        URLSession.shared.dataTask(with: request) { _, _, _ in }.resume()
    }
}
