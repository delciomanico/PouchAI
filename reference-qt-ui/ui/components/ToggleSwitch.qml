import QtQuick
import QtQuick.Controls.Basic
import DocumentApp

Switch {
    id: control
    implicitWidth: 44
    implicitHeight: 26

    indicator: Rectangle {
        width: 44; height: 26
        radius: 13
        color: control.checked ? Theme.colors.accent : Theme.colors.border
        Behavior on color { ColorAnimation { duration: Theme.motion.fast } }

        Rectangle {
            width: 20; height: 20
            radius: 10
            color: "#FFFFFF"
            y: 3
            x: control.checked ? parent.width - width - 3 : 3
            Behavior on x { NumberAnimation { duration: Theme.motion.fast; easing.type: Theme.motion.easing } }

            // faux drop shadow
            Rectangle {
                anchors.fill: parent
                radius: parent.radius
                color: "transparent"
                border.width: 0
                z: -1
            }
        }
    }
}
