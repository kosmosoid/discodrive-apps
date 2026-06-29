import SwiftUI

@main
struct DiscoDriveFastSyncApp: App {
    @StateObject private var model = AppModel()
    @Environment(\.scenePhase) private var scenePhase

    var body: some Scene {
        WindowGroup {
            Group {
                if model.paired { SyncView() } else { SetupView() }
            }
            .environmentObject(model)
        }
        // Opportunistic background sync. iOS runs this when it sees fit (after the requested
        // ~20 min minimum); each run schedules the next. Registration is handled by this modifier.
        .backgroundTask(.appRefresh(AppModel.bgTaskID)) {
            await model.syncNow()
            model.scheduleBackgroundSync()
        }
        .onChange(of: scenePhase) { _, phase in
            if phase == .background { model.scheduleBackgroundSync() }
        }
    }
}
