package ui

/*
#cgo pkg-config: gtk+-3.0
#cgo CFLAGS: -Wno-deprecated-declarations
#include <stdlib.h>
#include <gtk/gtk.h>

static void powerwall_status_icon_popup(GtkMenu *menu, GtkStatusIcon *icon) {
	gtk_menu_popup(menu, NULL, NULL, gtk_status_icon_position_menu, icon, 0, gtk_get_current_event_time());
}
*/
import "C"

import (
	"unsafe"

	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
)

// gotk3 hides GtkStatusIcon unless consumers enable its broad gtk_deprecated
// build tag. Keep this small wrapper local so the optional Linux menu-bar
// equivalent does not change build flags for the rest of the application.
type statusIcon struct {
	*glib.Object
}

func newStatusIcon(iconName string) *statusIcon {
	name := C.CString(iconName)
	defer C.free(unsafe.Pointer(name))
	native := C.gtk_status_icon_new_from_icon_name((*C.gchar)(name))
	if native == nil {
		return nil
	}
	return &statusIcon{Object: glib.Take(unsafe.Pointer(native))}
}

func (s *statusIcon) native() *C.GtkStatusIcon {
	if s == nil || s.GObject == nil {
		return nil
	}
	return (*C.GtkStatusIcon)(unsafe.Pointer(s.GObject))
}

func (s *statusIcon) setVisible(visible bool) {
	value := C.gboolean(0)
	if visible {
		value = 1
	}
	C.gtk_status_icon_set_visible(s.native(), value)
}

func (s *statusIcon) setTitle(title string) {
	value := C.CString(title)
	defer C.free(unsafe.Pointer(value))
	C.gtk_status_icon_set_title(s.native(), (*C.gchar)(value))
}

func (s *statusIcon) setTooltip(text string) {
	value := C.CString(text)
	defer C.free(unsafe.Pointer(value))
	C.gtk_status_icon_set_tooltip_text(s.native(), (*C.gchar)(value))
}

func (s *statusIcon) popup(menu *gtk.Menu) {
	C.powerwall_status_icon_popup((*C.GtkMenu)(unsafe.Pointer(menu.GObject)), s.native())
}
