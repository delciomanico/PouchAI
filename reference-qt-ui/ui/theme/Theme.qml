pragma Singleton
import QtQuick

// ---------------------------------------------------------------------------
// Theme
//
// Single source of truth for every design token used across the app.
// Values are lifted directly from the visual frames (the light,
// warm-neutral theme with an oblique-cut brand motif and 3D-raised
// components) rather than invented — see each group's comment for where
// it maps back.
//
// Usage: `import DocumentApp` then `Theme.colors.accent`, `Theme.spacing.md`, etc.
// Accent is user-tweakable (it was an editable "prop" in the design tool);
// change `accentHex` below to re-theme the whole app.
// ---------------------------------------------------------------------------
QtObject {
    id: root

    // The one truly "themeable" value — everything under colors.accent* is
    // derived from this at runtime via Qt.lighter/Qt.darker, mirroring the
    // color-mix()-based 3D button/chip treatment from the frames.
    readonly property color accentHex: "#1F5D50"

    readonly property QtObject colors: QtObject {
        readonly property color background: "#F5F3EE"
        readonly property color surface: "#FFFFFF"
        readonly property color surfaceSunken: "#F3F0E7"   // recessed inputs / thumbnails
        readonly property color border: "#E7E3D8"
        readonly property color borderStrong: "#DFDAC9"

        readonly property color textPrimary: "#201F1B"
        readonly property color textSecondary: "#3A362D"
        readonly property color textMuted: "#6B6759"
        readonly property color textFaint: "#8C8778"
        readonly property color textFainter: "#9A9584"
        readonly property color textFaintest: "#B4AF9F"

        readonly property color accent: root.accentHex
        readonly property color accentSoft: "#E4EFEC"
        readonly property color accentLight: Qt.lighter(root.accentHex, 1.18)
        readonly property color accentDark: Qt.darker(root.accentHex, 1.28)

        // status / confidence semantics — independent of accent
        readonly property color success: "#1E8F5F"
        readonly property color successSoft: "#E6F4EA"
        readonly property color warning: "#B8860B"
        readonly property color warningSoft: "#FBF1DC"
        readonly property color danger: "#B84B3E"
        readonly property color dangerSoft: "#FBEAE7"

        // per-folder accent gradients (id -> {start,end}), used by
        // DocumentCard's stacked-thumbnail motif and folder-colored icons.
        readonly property var folderGradients: ({
            teal:       { start: "#2E7D6B", end: "#123F35" },
            terracotta: { start: "#C97B4A", end: "#6B3A1F" },
            blue:       { start: "#3E6FB0", end: "#1B3A63" },
            purple:     { start: "#7C5CC4", end: "#3A2766" },
            green:      { start: "#4E9B6B", end: "#1E4A30" }
        })
    }

    readonly property QtObject typography: QtObject {
        // Manrope for UI text, IBM Plex Mono for technical/numeric data
        // (counts, dates, confidence %, build tag) — the frames use this
        // split deliberately to give the app a precise, "native" feel.
        readonly property string uiFamily: "Manrope"
        readonly property string monoFamily: "IBM Plex Mono"

        readonly property int sizeXs: 11
        readonly property int sizeSm: 12
        readonly property int sizeMd: 13
        readonly property int sizeBase: 14
        readonly property int sizeLg: 15
        readonly property int sizeXl: 17
        readonly property int sizeXxl: 22
        readonly property int sizeDisplay: 26

        readonly property int weightRegular: Font.Normal
        readonly property int weightMedium: Font.Medium
        readonly property int weightSemibold: Font.DemiBold
        readonly property int weightBold: Font.Bold
        readonly property int weightExtrabold: Font.ExtraBold
    }

    readonly property QtObject spacing: QtObject {
        readonly property int xxs: 4
        readonly property int xs: 8
        readonly property int sm: 12
        readonly property int md: 16
        readonly property int lg: 22
        readonly property int xl: 32
        readonly property int xxl: 44
    }

    readonly property QtObject radius: QtObject {
        readonly property int sm: 8
        readonly property int md: 10
        readonly property int lg: 16
        readonly property int xl: 18
        readonly property int pill: 9999
    }

    readonly property QtObject elevation: QtObject {
        // Matches the frames' 3D-raised components: a soft ambient shadow
        // plus a 1px light "inner highlight" simulated by a top border on
        // the drawing rectangle (see components/PrimaryButton.qml etc).
        readonly property color shadowColor: "#28211F1B" // ~16% black
        readonly property real cardRadius: 14
        readonly property real cardBlur: 26
        readonly property real cardOffsetY: 10

        readonly property real buttonBlur: 14
        readonly property real buttonOffsetY: 6
    }

    readonly property QtObject motion: QtObject {
        readonly property int fast: 120
        readonly property int base: 180
        readonly property int slow: 260
        readonly property int easing: Easing.OutCubic
    }

    // the oblique-cut inset used by the brand mark and the top nav bar's
    // slanted edges (see components/AppTopBar.qml)
    readonly property int obliqueCut: 22
}
