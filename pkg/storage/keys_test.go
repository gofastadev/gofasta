package storage

import (
	"strconv"
	"strings"
	"testing"
	"time"
)

// at is a fixed instant so key assertions can name the exact millisecond.
var at = time.Unix(0, 1_700_000_000_000*int64(time.Millisecond))

func newTestResolver() *KeyResolver {
	r := NewKeyResolver()
	r.SetOwnerSegment("learner", "learners")
	r.Register("profile_picture", KeyPattern{
		Bucket:  "profiles",
		Pattern: "{ownerKind}/{ownerID}/profile/{timestamp}.{extension}",
	})
	return r
}

func TestKeyResolver_Resolve(t *testing.T) {
	bucket, key, err := newTestResolver().Resolve(KeyRequest{
		OwnerKind: "learner",
		Category:  "profile_picture",
		OwnerID:   "abc-123",
		Extension: "jpg",
		Timestamp: at,
	})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if bucket != "profiles" {
		t.Errorf("bucket = %q", bucket)
	}
	if want := "learners/abc-123/profile/1700000000000.jpg"; key != want {
		t.Errorf("key = %q, want %q", key, want)
	}
}

func TestKeyResolver_OwnerSegmentReplacesAPluralizationTable(t *testing.T) {
	// Without a registered segment the kind renders verbatim, so a project
	// that wants "learners/" says so rather than relying on the library to
	// guess an English plural.
	r := NewKeyResolver()
	r.Register("doc", KeyPattern{Bucket: "b", Pattern: "{ownerKind}/x"})

	_, key, err := r.Resolve(KeyRequest{OwnerKind: "internal_admin", Category: "doc"})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if key != "internal_admin/x" {
		t.Errorf("key = %q, want the kind verbatim when no segment is set", key)
	}

	r.SetOwnerSegment("internal_admin", "internal_admins")
	_, key, _ = r.Resolve(KeyRequest{OwnerKind: "internal_admin", Category: "doc"})
	if key != "internal_admins/x" {
		t.Errorf("key = %q, want the registered segment", key)
	}
}

func TestKeyResolver_OverrideBeatsDefault(t *testing.T) {
	r := newTestResolver()
	r.RegisterFor("facilitator", "profile_picture", KeyPattern{
		Bucket:  "staff",
		Pattern: "staff/{ownerID}.{extension}",
	})

	bucket, key, _ := r.Resolve(KeyRequest{
		OwnerKind: "facilitator", Category: "profile_picture",
		OwnerID: "f1", Extension: "png", Timestamp: at,
	})
	if bucket != "staff" || key != "staff/f1.png" {
		t.Errorf("override not applied: %q %q", bucket, key)
	}

	// The default is untouched for everyone else.
	bucket, _, _ = r.Resolve(KeyRequest{
		OwnerKind: "learner", Category: "profile_picture", OwnerID: "l1", Timestamp: at,
	})
	if bucket != "profiles" {
		t.Errorf("default bucket = %q, want profiles", bucket)
	}
}

func TestKeyResolver_UnknownCategoryIsAnError(t *testing.T) {
	_, _, err := newTestResolver().Resolve(KeyRequest{Category: "nope"})
	if err == nil {
		t.Fatal("no error for an unregistered category")
	}
	if !strings.Contains(err.Error(), "nope") {
		t.Errorf("error %q does not name the category", err)
	}
}

// The uploader chooses the filename, so the extension is the field an attacker
// controls. None of these may add path structure or escape the bucket.
func TestKeyResolver_SanitizesUserControlledSegments(t *testing.T) {
	r := NewKeyResolver()
	r.Register("doc", KeyPattern{Bucket: "b", Pattern: "{ownerID}/{referenceID}/f.{extension}"})

	cases := []struct{ name, owner, ref, ext string }{
		{"slash in extension", "o", "r", "jpg/../../etc/passwd"},
		{"traversal in reference", "o", "../../..", "jpg"},
		{"traversal in owner", "../..", "r", "jpg"},
		{"backslash", "o", "r", `jpg\..\..`},
		{"absolute", "/etc/passwd", "r", "jpg"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, key, err := r.Resolve(KeyRequest{
				Category: "doc", OwnerID: tc.owner, ReferenceID: tc.ref, Extension: tc.ext,
			})
			if err != nil {
				t.Fatalf("Resolve: %v", err)
			}
			// The safety property is not "the characters .. never appear" —
			// "jpg_.._etc" is a single harmless segment. It is that no
			// substituted value introduces path structure, and that no
			// segment is exactly "." or "..", which is what filepath.Join
			// resolves upward.
			if strings.Count(key, "/") != 2 {
				t.Errorf("key %q gained path structure from a substituted value", key)
			}
			for _, segment := range strings.Split(key, "/") {
				if segment == "." || segment == ".." {
					t.Errorf("key %q contains a traversal segment", key)
				}
			}
			if strings.ContainsAny(key, `\`) {
				t.Errorf("key %q contains a backslash", key)
			}
		})
	}
}

func TestKeyResolver_LeadingDotOnExtensionIsAccepted(t *testing.T) {
	// Callers pass whatever filepath.Ext returned, which includes the dot.
	r := NewKeyResolver()
	r.Register("doc", KeyPattern{Bucket: "b", Pattern: "f.{extension}"})

	for _, ext := range []string{"jpg", ".jpg"} {
		_, key, _ := r.Resolve(KeyRequest{Category: "doc", Extension: ext})
		if key != "f.jpg" {
			t.Errorf("extension %q produced %q, want f.jpg", ext, key)
		}
	}
}

func TestKeyResolver_EmptyOwnerIDFallsBack(t *testing.T) {
	// An empty segment would collapse two slashes and make the key ambiguous.
	r := NewKeyResolver()
	r.Register("doc", KeyPattern{Bucket: "b", Pattern: "{ownerID}/f"})

	_, key, _ := r.Resolve(KeyRequest{Category: "doc", OwnerID: ""})
	if key != "unknown/f" {
		t.Errorf("key = %q, want the unknown fallback", key)
	}
}

func TestKeyResolver_ZeroTimestampIsNow(t *testing.T) {
	r := NewKeyResolver()
	r.Register("doc", KeyPattern{Bucket: "b", Pattern: "{timestamp}"})

	before := time.Now().UnixMilli()
	_, key, _ := r.Resolve(KeyRequest{Category: "doc"})
	after := time.Now().UnixMilli()

	got, err := strconv.ParseInt(key, 10, 64)
	if err != nil {
		t.Fatalf("timestamp %q is not a number: %v", key, err)
	}
	if got < before || got > after {
		t.Errorf("timestamp %d outside [%d, %d]", got, before, after)
	}
}

func TestKeyResolver_UnknownPlaceholderSurvives(t *testing.T) {
	// Blanking it would silently collide two different objects onto one key.
	r := NewKeyResolver()
	r.Register("doc", KeyPattern{Bucket: "b", Pattern: "{ownerID}/{typo}"})

	_, key, _ := r.Resolve(KeyRequest{Category: "doc", OwnerID: "o"})
	if !strings.Contains(key, "{typo}") {
		t.Errorf("key = %q, want the unknown placeholder left visible", key)
	}
}

func TestKeyResolver_BucketForAndCategories(t *testing.T) {
	r := newTestResolver()
	r.Register("logo", KeyPattern{Bucket: "logos", Pattern: "l"})

	if got, err := r.BucketFor("learner", "profile_picture"); err != nil || got != "profiles" {
		t.Errorf("BucketFor = %q, %v", got, err)
	}
	if _, err := r.BucketFor("learner", "nope"); err == nil {
		t.Error("BucketFor accepted an unregistered category")
	}

	got := r.Categories()
	if len(got) != 2 || got[0] != "logo" || got[1] != "profile_picture" {
		t.Errorf("Categories = %v, want a sorted pair", got)
	}
}

func TestKeyResolver_Unregister(t *testing.T) {
	r := newTestResolver()
	r.Unregister("profile_picture")
	if _, _, err := r.Resolve(KeyRequest{Category: "profile_picture"}); err == nil {
		t.Error("category still resolves after Unregister")
	}

	r.Register("doc", KeyPattern{Bucket: "b", Pattern: "d"})
	r.RegisterFor("learner", "doc", KeyPattern{Bucket: "over", Pattern: "o"})
	r.UnregisterFor("learner", "doc")
	if bucket, _, _ := r.Resolve(KeyRequest{OwnerKind: "learner", Category: "doc"}); bucket != "b" {
		t.Errorf("bucket = %q, want the default back after UnregisterFor", bucket)
	}
}
