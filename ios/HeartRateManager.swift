import Combine
import CoreBluetooth
import Foundation

@MainActor
final class HeartRateManager: NSObject, ObservableObject {
    @Published private(set) var bpm: Int?
    @Published private(set) var status: String = "idle"
    @Published private(set) var isConnected: Bool = false

    private let serviceUUID = CBUUID(string: "180D")
    private let measurementUUID = CBUUID(string: "2A37")

    private var central: CBCentralManager!
    private var peripheral: CBPeripheral?
    private var measurement: CBCharacteristic?
    private var shouldConnect = false
    private var isScanning = false

    override init() {
        super.init()
        central = CBCentralManager(delegate: self, queue: nil)
    }

    func start() {
        shouldConnect = true
        if central.state == .poweredOn {
            beginScan()
        }
    }

    func stop() {
        shouldConnect = false
        isScanning = false
        central.stopScan()
        measurement = nil
        bpm = nil
        status = "stopped"
        if let peripheral {
            central.cancelPeripheralConnection(peripheral)
        }
    }

    private func beginScan() {
        guard !isScanning else { return }
        isScanning = true
        status = "scanning"
        central.scanForPeripherals(withServices: [serviceUUID], options: [
            CBCentralManagerScanOptionAllowDuplicatesKey: false,
        ])
    }

    private func connect(_ peripheral: CBPeripheral) {
        self.peripheral = peripheral
        self.peripheral?.delegate = self
        isScanning = false
        central.stopScan()
        status = "connecting"
        central.connect(peripheral, options: nil)
    }

    private func subscribeIfReady() {
        guard let peripheral, let measurement else { return }
        if measurement.properties.contains(.notify) {
            peripheral.setNotifyValue(true, for: measurement)
        }
    }
}

extension HeartRateManager: CBCentralManagerDelegate {
    func centralManagerDidUpdateState(_ central: CBCentralManager) {
        switch central.state {
        case .poweredOn:
            if shouldConnect {
                beginScan()
            }
        case .unauthorized:
            status = "unauthorized"
        case .poweredOff:
            status = "bluetooth_off"
        default:
            status = "unavailable"
        }
    }

    func centralManager(_ central: CBCentralManager, didDiscover peripheral: CBPeripheral, advertisementData: [String: Any], rssi RSSI: NSNumber) {
        guard shouldConnect else { return }
        if self.peripheral != nil { return }
        connect(peripheral)
    }

    func centralManager(_ central: CBCentralManager, didConnect peripheral: CBPeripheral) {
        isConnected = true
        status = "connected"
        peripheral.discoverServices([serviceUUID])
    }

    func centralManager(_ central: CBCentralManager, didFailToConnect peripheral: CBPeripheral, error: Error?) {
        isConnected = false
        status = "connect_failed"
        if shouldConnect {
            beginScan()
        }
    }

    func centralManager(_ central: CBCentralManager, didDisconnectPeripheral peripheral: CBPeripheral, error: Error?) {
        isConnected = false
        measurement = nil
        if shouldConnect {
            beginScan()
        } else {
            status = "disconnected"
        }
    }
}

extension HeartRateManager: CBPeripheralDelegate {
    func peripheral(_ peripheral: CBPeripheral, didDiscoverServices error: Error?) {
        guard error == nil else { return }
        guard let services = peripheral.services else { return }
        for service in services where service.uuid == serviceUUID {
            peripheral.discoverCharacteristics([measurementUUID], for: service)
        }
    }

    func peripheral(_ peripheral: CBPeripheral, didDiscoverCharacteristicsFor service: CBService, error: Error?) {
        guard error == nil else { return }
        guard let characteristics = service.characteristics else { return }
        for characteristic in characteristics where characteristic.uuid == measurementUUID {
            measurement = characteristic
            subscribeIfReady()
        }
    }

    func peripheral(_ peripheral: CBPeripheral, didUpdateValueFor characteristic: CBCharacteristic, error: Error?) {
        guard error == nil else { return }
        guard characteristic.uuid == measurementUUID else { return }
        guard let data = characteristic.value else { return }
        let bytes = [UInt8](data)
        guard bytes.count >= 2 else { return }
        let flags = bytes[0]
        let isUInt16 = (flags & 0x01) != 0
        let value: Int
        if isUInt16, bytes.count >= 3 {
            value = Int(UInt16(bytes[1]) | (UInt16(bytes[2]) << 8))
        } else {
            value = Int(bytes[1])
        }
        bpm = value
    }
}
