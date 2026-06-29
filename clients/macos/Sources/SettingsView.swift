import SwiftUI
import DiscoKit

struct SettingsView: View {
    @EnvironmentObject var app: AppState

    var body: some View {
        Form {
            Picker(app.t("settings.language"), selection: Binding(
                get: { app.language },
                set: { newLang in Task { await app.setLanguage(newLang) } }
            )) {
                ForEach(L10n.supported, id: \.self) { code in
                    Text(L10n.displayName[code] ?? code).tag(code)
                }
            }
        }
        .padding(20)
        .frame(width: 320)
        .navigationTitle(app.t("settings.title"))
    }
}
