//go:build darwin

package procsetup

/*
#include <pthread/qos.h>

static int kpi_qos_user_interactive(void) {
	return pthread_set_qos_class_self_np(QOS_CLASS_USER_INTERACTIVE, 0);
}
*/
import "C"

// Interactive asks macOS to treat the calling thread as user-interactive,
// which favours the performance cores (research plan §6, native runs): macOS
// cannot pin a thread to a core type, so this is a hint, and the core type is
// still checked with the canary. The caller has locked the goroutine to its
// thread. It reports whether the hint was accepted.
func Interactive() bool { return C.kpi_qos_user_interactive() == 0 }
