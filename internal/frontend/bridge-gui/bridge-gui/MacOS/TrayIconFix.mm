// Copyright (c) 2026 Proton AG
//
// This file is part of Proton Mail Bridge.
//
// Proton Mail Bridge is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Proton Mail Bridge is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with Proton Mail Bridge.  If not, see <https://www.gnu.org/licenses/>.

#pragma clang diagnostic push
#pragma clang diagnostic ignored "-Wavailability"
#pragma clang diagnostic ignored "-Wdeprecated-declarations"
#pragma clang diagnostic ignored "-Wnullability-completeness"
#pragma clang diagnostic ignored "-Wdeprecated-anon-enum-enum-conversion"
#include <Cocoa/Cocoa.h>
#pragma clang diagnostic pop
#include <QString>
#include "TrayIconFix.h"
#include "QMLBackend.h"

using namespace bridgepp;

#import <objc/runtime.h>

#ifdef Q_OS_MACOS


//****************************************************************************************************************************************************
/// \brief Qt's QCocoaSystemTrayIcon::emitActivated() calls - [NSEvent clickCount] and [NSEvent buttonNumber] on NSApp.currentEvent unconditionally.
/// On macOS 27 GoldenGate, status items are scene-based, so the event backing a tray-menu open is not a mouse event, and these accessors raise an exception
/// crashing the app. Both accessors return 0 for non-mouse events instead of throwing.
//****************************************************************************************************************************************************

static bool isMouseEventType(NSEventType type) {
    switch (type) {
    case NSEventTypeLeftMouseDown:
    case NSEventTypeLeftMouseUp:
    case NSEventTypeRightMouseDown:
    case NSEventTypeRightMouseUp:
    case NSEventTypeOtherMouseDown:
    case NSEventTypeOtherMouseUp:
    case NSEventTypeLeftMouseDragged:
    case NSEventTypeRightMouseDragged:
    case NSEventTypeOtherMouseDragged:
        return true;
    default:
        return false;
    }
}


static void guardNSEventMouseAccessor(SEL selector) {
    Method method = class_getInstanceMethod([NSEvent class], selector);
    if (!method) {
        app().log().warn(QString("No method found for selector %1").arg(sel_getName(selector)));
        return;
    }
    IMP imp = method_getImplementation(method);
    if (!imp)
    {
        app().log().warn(QString("No implementation found for selector %1").arg(sel_getName(selector)));
        return;
    }
    auto const original = reinterpret_cast<NSInteger (*)(id, SEL)>(imp);

    IMP guarded = imp_implementationWithBlock(^NSInteger(NSEvent *event) {
        return isMouseEventType(event.type) ? original(event, selector) : 0;
    });
    if (!guarded)
    {
        app().log().error(QString("imp_implementationWithBlock failed for selector %1").arg(sel_getName(selector)));
        return;
    }
    method_setImplementation(method, guarded);
}


void installMacOsGoldenGateTrayIconFix() {
    if (![NSProcessInfo.processInfo isOperatingSystemAtLeastVersion:(NSOperatingSystemVersion){27, 0, 0}]) {
        return;
    }
    guardNSEventMouseAccessor(@selector(clickCount));
    guardNSEventMouseAccessor(@selector(buttonNumber));
}


#endif // #ifdef Q_OS_MACOS
