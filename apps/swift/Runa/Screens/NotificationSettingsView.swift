import SwiftUI
import UserNotifications
import Shared

/// 通知設定 (21) — 夜のリマインド. A quiet toggle, a large time display, three preset
/// chips (21:00 / 22:00 / 23:00) plus a free time picker, over a poetic footer.
/// Turning the reminder on asks for notification authorization; a denial doesn't
/// break the screen — the preference is still saved (DoD#3).
struct NotificationSettingsView: View {
    @Environment(\.runaTheme) private var runaTheme
    @StateObject private var obs = NotificationSettingsObservable()
    @State private var showPicker = false

    var body: some View {
        VStack(alignment: .leading, spacing: 0) {
            Text("NOTIFICATION")
                .font(RunaFonts.body(13)).tracking(3)
                .foregroundStyle(runaTheme.subtle)
                .padding(.top, RunaSpacing.md)
            Text("夜のリマインド")
                .font(RunaFonts.heading(40))
                .foregroundStyle(runaTheme.heading)
                .padding(.top, RunaSpacing.xs)

            Toggle(isOn: Binding(
                get: { obs.enabled },
                set: { on in if on { enableReminder() } else { obs.toggle(false) } }
            )) {
                Text("リマインドをうけとる")
                    .font(RunaFonts.body(17))
                    .foregroundStyle(runaTheme.heading)
            }
            .tint(runaTheme.accent)
            .padding(.top, RunaSpacing.lg)

            Text("おやすみ前の、しずかな時刻")
                .font(RunaFonts.body(14))
                .foregroundStyle(runaTheme.subtle)
                .frame(maxWidth: .infinity)
                .padding(.top, RunaSpacing.xl)

            Text(obs.time.label)
                .font(.system(size: 72, weight: .light))
                .foregroundStyle(runaTheme.heading)
                .frame(maxWidth: .infinity)
                .padding(.top, RunaSpacing.sm)
                .contentShape(Rectangle())
                .onTapGesture { showPicker = true }

            HStack(spacing: 12) {
                ForEach(obs.presets, id: \.label) { preset in
                    presetChip(preset)
                }
            }
            .frame(maxWidth: .infinity)
            .padding(.top, RunaSpacing.md)

            Spacer()
            Text("その時刻に、月がそっと\n今日をふりかえるひとことを。")
                .font(RunaFonts.body(14))
                .foregroundStyle(runaTheme.subtle)
                .multilineTextAlignment(.center)
                .frame(maxWidth: .infinity)
                .padding(.bottom, RunaSpacing.lg)
        }
        .padding(.horizontal, 28)
        .frame(maxWidth: .infinity, maxHeight: .infinity, alignment: .topLeading)
        .background(runaTheme.background)
        .navigationTitle("")
        .navigationBarTitleDisplayMode(.inline)
        .sheet(isPresented: $showPicker) {
            TimePickerSheet(initial: obs.time) { picked in
                obs.selectTime(picked)
                showPicker = false
            } onCancel: {
                showPicker = false
            }
        }
    }

    private func presetChip(_ preset: ReminderTime) -> some View {
        let selected = preset.hour == obs.time.hour && preset.minute == obs.time.minute
        return Text(preset.label)
            .font(RunaFonts.body(16))
            .foregroundStyle(selected ? runaTheme.accent : runaTheme.subtle)
            .padding(.horizontal, 22)
            .padding(.vertical, 12)
            .overlay(
                Capsule().stroke(selected ? runaTheme.accent : runaTheme.subtle.opacity(0.4), lineWidth: 1)
            )
            .contentShape(Capsule())
            .onTapGesture { obs.selectTime(preset) }
    }

    private func enableReminder() {
        UNUserNotificationCenter.current().requestAuthorization(options: [.alert, .sound]) { _, _ in }
        obs.toggle(true)
    }
}

/// A modal hour+minute picker for the free reminder time.
private struct TimePickerSheet: View {
    @Environment(\.runaTheme) private var runaTheme
    let initial: ReminderTime
    let onConfirm: (ReminderTime) -> Void
    let onCancel: () -> Void

    @State private var date: Date

    init(initial: ReminderTime, onConfirm: @escaping (ReminderTime) -> Void, onCancel: @escaping () -> Void) {
        self.initial = initial
        self.onConfirm = onConfirm
        self.onCancel = onCancel
        var components = DateComponents()
        components.hour = Int(initial.hour)
        components.minute = Int(initial.minute)
        _date = State(initialValue: Calendar.current.date(from: components) ?? Date())
    }

    var body: some View {
        VStack(spacing: RunaSpacing.lg) {
            DatePicker("", selection: $date, displayedComponents: .hourAndMinute)
                .datePickerStyle(.wheel)
                .labelsHidden()
            HStack {
                Button("やめる", action: onCancel)
                    .foregroundStyle(runaTheme.subtle)
                Spacer()
                Button("決定") {
                    let c = Calendar.current.dateComponents([.hour, .minute], from: date)
                    onConfirm(ReminderTime(hour: Int32(c.hour ?? 22), minute: Int32(c.minute ?? 0)))
                }
                .foregroundStyle(runaTheme.accent)
            }
            .padding(.horizontal, 28)
        }
        .padding(.top, RunaSpacing.xl)
        .frame(maxWidth: .infinity, maxHeight: .infinity, alignment: .top)
        .background(runaTheme.background)
        .presentationDetents([.height(320)])
    }
}
