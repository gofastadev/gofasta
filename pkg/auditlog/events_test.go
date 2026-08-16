package auditlog

import "testing"

func TestEventName_UppercasesBothHalves(t *testing.T) {
	// The event_type column is queried by exact string, so a caller passing
	// lowercase must not produce a second spelling of an existing event.
	cases := []struct {
		resource string
		action   string
		want     string
	}{
		{"COURSE", "CREATED", "COURSE_CREATED"},
		{"course", "created", "COURSE_CREATED"},
		{"Live_Class", "Deleted", "LIVE_CLASS_DELETED"},
		{"", "", "_"},
	}

	for _, tc := range cases {
		if got := EventName(tc.resource, tc.action); got != tc.want {
			t.Errorf("EventName(%q, %q) = %q, want %q", tc.resource, tc.action, got, tc.want)
		}
	}
}

func TestParseMutationName(t *testing.T) {
	cases := []struct {
		name         string
		field        string
		wantResource string
		wantAction   string
	}{
		{"create prefix", "createCourse", "COURSE", ActionCreated},
		{"multi-word remainder", "updateCourseVersion", "COURSE_VERSION", ActionUpdated},
		{"delete prefix", "deleteLiveClass", "LIVE_CLASS", ActionDeleted},
		{"add is a create", "addStudent", "STUDENT", ActionCreated},
		{"assign is a create", "assignRole", "ROLE", ActionCreated},
		{"register is a create", "registerDevice", "DEVICE", ActionCreated},
		{"initialize is a create", "initializePayment", "PAYMENT", ActionCreated},
		{"edit is an update", "editProfile", "PROFILE", ActionUpdated},
		{"toggle is an update", "toggleVisibility", "VISIBILITY", ActionUpdated},
		{"publish is an update", "publishCourse", "COURSE", ActionUpdated},
		{"approve is an update", "approveEnrollment", "ENROLLMENT", ActionUpdated},
		{"suspend is an update", "suspendAccount", "ACCOUNT", ActionUpdated},
		{"remove is a delete", "removeMember", "MEMBER", ActionDeleted},
		{"revoke is a delete", "revokeToken", "TOKEN", ActionDeleted},
		{"cancel is a delete", "cancelSubscription", "SUBSCRIPTION", ActionDeleted},

		// A prefix with nothing after it is still the action it names — the
		// mutation `delete(id: …)` is a deletion, not an unclassified event.
		{"bare prefix", "delete", "DELETE", ActionDeleted},

		// The lowercase remainder is what stops `setup` from being read as
		// `set` + `up`: a prefix only matches on a word boundary.
		{"prefix is only a substring", "setup", "SETUP", ""},
		{"unknown verb", "unknownThing", "UNKNOWN_THING", ""},
		{"already snake-ish", "syncAllCourses", "SYNC_ALL_COURSES", ""},

		// Fails closed rather than panicking on the empty remainder.
		{"empty field name", "", "", ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resource, action := ParseMutationName(tc.field)
			if resource != tc.wantResource || action != tc.wantAction {
				t.Errorf("ParseMutationName(%q) = (%q, %q), want (%q, %q)",
					tc.field, resource, action, tc.wantResource, tc.wantAction)
			}
		})
	}
}

func TestCamelToUpperSnake(t *testing.T) {
	cases := []struct{ in, want string }{
		{"CourseVersion", "COURSE_VERSION"},
		{"liveClass", "LIVE_CLASS"},
		{"course", "COURSE"},
		{"URL", "U_R_L"}, // consecutive capitals are each a boundary
		{"", ""},
	}

	for _, tc := range cases {
		if got := camelToUpperSnake(tc.in); got != tc.want {
			t.Errorf("camelToUpperSnake(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
