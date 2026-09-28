import QtQuick
import Qt5Compat.GraphicalEffects

// SVG assets under assets/icons/ are drawn with stroke="#000000" so a
// single file can be recolored per use (nav pill, accent button, muted
// list icon, ...) instead of keeping a copy per color. Uses
// Qt5Compat.GraphicalEffects rather than QtQuick.Effects/MultiEffect so
// it works on Qt 6.2+ instead of requiring 6.5+.
Item {
    id: root

    property url source: ""
    property color color: "#000000"
    property int size: 16

    implicitWidth: size
    implicitHeight: size

    Image {
        id: img
        anchors.fill: parent
        source: root.source
        fillMode: Image.PreserveAspectFit
        smooth: true
        visible: false
    }

    ColorOverlay {
        anchors.fill: parent
        source: img
        color: root.color
    }
}
