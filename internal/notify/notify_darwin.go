//go:build darwin

package notify

/*
#cgo CFLAGS: -x objective-c -Wno-deprecated-declarations
#cgo LDFLAGS: -framework Foundation -framework UserNotifications -framework AppKit
#import <Foundation/Foundation.h>
#import <UserNotifications/UserNotifications.h>
#import <AppKit/AppKit.h>

static void notify_log(const char *fmt, ...) {
	FILE *f = fopen("/tmp/worktime_notify.log", "a");
	if (!f) return;
	va_list ap;
	va_start(ap, fmt);
	vfprintf(f, fmt, ap);
	va_end(ap);
	fprintf(f, "\n");
	fclose(f);
}

static int  _authGranted = 0;
static int  _authDone   = 0;
static char _errBuf[1024];

static void initAuth(void) {
	notify_log("initAuth start");

	dispatch_semaphore_t sem = dispatch_semaphore_create(0);

	UNUserNotificationCenter *center = [UNUserNotificationCenter currentNotificationCenter];
	notify_log("center=%p thread=%@", (void*)center, [NSThread currentThread]);

	[center getNotificationSettingsWithCompletionHandler:^(UNNotificationSettings *settings) {
		notify_log("getNotificationSettings handler: status=%ld thread=%@",
			(long)settings.authorizationStatus, [NSThread currentThread]);

		if (settings.authorizationStatus == UNAuthorizationStatusAuthorized) {
			notify_log("Authorized");
			_authGranted = 1;
			_authDone = 1;
			dispatch_semaphore_signal(sem);
			return;
		}

		if (settings.authorizationStatus == UNAuthorizationStatusDenied) {
			notify_log("Denied");
			snprintf(_errBuf, sizeof(_errBuf),
				"Denied, 请在 系统设置→通知→worktime 中开启");
			_authDone = 1;
			dispatch_semaphore_signal(sem);
			return;
		}

		notify_log("NotDetermined, requesting auth...");
		[center requestAuthorizationWithOptions:UNAuthorizationOptionAlert | UNAuthorizationOptionSound
						  completionHandler:^(BOOL granted, NSError *err) {
			notify_log("requestAuth completion: granted=%d err=%@ thread=%@",
				granted, err, [NSThread currentThread]);
			_authGranted = granted ? 1 : 0;
			_authDone = 1;
			if (err) {
				snprintf(_errBuf, sizeof(_errBuf), "%s",
					[[err localizedDescription] UTF8String]);
			} else if (!granted) {
				snprintf(_errBuf, sizeof(_errBuf),
					"授权被拒绝, 请在 系统设置→通知→worktime 中开启");
			}
			dispatch_semaphore_signal(sem);
		}];
	}];

	notify_log("waiting on semaphore...");
	dispatch_semaphore_wait(sem, DISPATCH_TIME_FOREVER);
	notify_log("semaphore done, granted=%d errBuf=%s", _authGranted, _errBuf);
}

const char* sendNotification(const char *title, const char *message) {
	notify_log("sendNotification called");
	_errBuf[0] = 0;

	if (!_authDone) {
		initAuth();
	}

	notify_log("authDone=%d authGranted=%d", _authDone, _authGranted);

	if (!_authGranted) {
		return _errBuf;
	}

	notify_log("delivering on main queue...");
	dispatch_async(dispatch_get_main_queue(), ^{
		UNUserNotificationCenter *center = [UNUserNotificationCenter currentNotificationCenter];
		notify_log("deliver center=%p", (void*)center);
		if (!center) return;

		UNMutableNotificationContent *content = [[UNMutableNotificationContent alloc] init];
		content.title = [NSString stringWithUTF8String:title];
		content.body = [NSString stringWithUTF8String:message];
		content.sound = [UNNotificationSound defaultSound];

		NSString *nid = [NSString stringWithFormat:@"wt-%@", [NSUUID UUID].UUIDString];
		UNTimeIntervalNotificationTrigger *trigger = [UNTimeIntervalNotificationTrigger triggerWithTimeInterval:1 repeats:NO];
		UNNotificationRequest *req = [UNNotificationRequest requestWithIdentifier:nid content:content trigger:trigger];

		[center addNotificationRequest:req withCompletionHandler:^(NSError *err) {
			if (err) notify_log("deliver err: %@", err);
			else     notify_log("deliver success");
		}];
	});

	return NULL;
}
*/
import "C"
import "unsafe"

func send(title, message string) error {
	cTitle := C.CString(title)
	cMessage := C.CString(message)
	defer C.free(unsafe.Pointer(cTitle))
	defer C.free(unsafe.Pointer(cMessage))

	res := C.sendNotification(cTitle, cMessage)
	if res != nil {
		return &notifyError{C.GoString(res)}
	}
	return nil
}

type notifyError struct{ msg string }

func (e *notifyError) Error() string { return e.msg }
