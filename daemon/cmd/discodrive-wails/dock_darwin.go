//go:build darwin

package main

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa
#import <Cocoa/Cocoa.h>

static void setActivationPolicy(int policy) {
    dispatch_async(dispatch_get_main_queue(), ^{
        [NSApp setActivationPolicy:(NSApplicationActivationPolicy)policy];
    });
}
*/
import "C"

// setDockVisible toggles the macOS dock icon: visible → Regular activation policy,
// hidden → Accessory (the app then lives only in the menu-bar tray, no dock icon).
// The change is dispatched to the main thread (callers run off it).
func setDockVisible(visible bool) {
	if visible {
		C.setActivationPolicy(0) // NSApplicationActivationPolicyRegular
	} else {
		C.setActivationPolicy(1) // NSApplicationActivationPolicyAccessory
	}
}
