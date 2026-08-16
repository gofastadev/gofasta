package storage

import (
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"
)

// Deciding where an object goes
//
// Every application storing user uploads has to answer the same question: for
// this kind of file, uploaded by this kind of owner, which bucket and which
// key? The answer is nearly always a template — "avatars/{ownerID}/{timestamp}"
// — with a per-owner exception or two, and it has to be deterministic so the
// object can be found again.
//
// KeyResolver is that, with the vocabulary left to the caller. The framework
// knows nothing about what an "owner" or a "category" is in your domain; it
// knows how to keep a registry of templates, pick the right one, and fill it
// in without letting user-controlled text escape the key.

// KeyPattern is where objects of one category live.
type KeyPattern struct {
	// Bucket is the container the object goes in.
	Bucket string

	// Pattern is the key template. These placeholders are substituted:
	//
	//	{ownerKind}    the owner's path segment (see SetOwnerSegment)
	//	{ownerID}      the owner's identifier, sanitized
	//	{referenceID}  the related record's identifier, sanitized
	//	{timestamp}    Unix milliseconds
	//	{extension}    the file extension, sanitized, without a leading dot
	//
	// An unknown placeholder is left as-is rather than blanked, so a typo
	// shows up in the key instead of silently producing a collision.
	Pattern string
}

// KeyRequest describes one object whose location is being resolved.
type KeyRequest struct {
	// OwnerKind names the sort of actor that uploaded this — whatever your
	// domain calls them. Optional: patterns that do not reference an owner
	// (a course thumbnail, say) can leave it empty.
	OwnerKind string

	// Category names the sort of file. Required: it selects the pattern.
	Category string

	OwnerID     string
	ReferenceID string

	// Extension is the file extension, with or without a leading dot. It
	// typically comes from an uploaded filename, which makes it the one field
	// here an attacker controls directly.
	Extension string

	// Timestamp is the instant recorded in the key. Zero means now.
	Timestamp time.Time
}

// KeyResolver maps a category — and optionally an owner kind — onto a bucket
// and key template.
//
// Safe for concurrent use. Registration is expected at startup, but a resolver
// read while another goroutine registers will not race.
type KeyResolver struct {
	mu        sync.RWMutex
	patterns  map[string]KeyPattern
	overrides map[string]map[string]KeyPattern
	segments  map[string]string
}

// NewKeyResolver returns an empty resolver. Register the categories your
// application stores before using it.
func NewKeyResolver() *KeyResolver {
	return &KeyResolver{
		patterns:  make(map[string]KeyPattern),
		overrides: make(map[string]map[string]KeyPattern),
		segments:  make(map[string]string),
	}
}

// Register sets the pattern used for a category by every owner kind.
func (r *KeyResolver) Register(category string, pattern KeyPattern) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.patterns[category] = pattern
}

// RegisterFor sets the pattern used for one owner kind, overriding whatever
// Register set for that category.
func (r *KeyResolver) RegisterFor(ownerKind, category string, pattern KeyPattern) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.overrides[ownerKind] == nil {
		r.overrides[ownerKind] = make(map[string]KeyPattern)
	}
	r.overrides[ownerKind][category] = pattern
}

// SetOwnerSegment sets the path segment an owner kind renders as.
//
// Without one, {ownerKind} renders the kind verbatim. This exists because the
// natural path segment is usually a pluralization of the domain term —
// "learner" living under "learners/" — and hardcoding a pluralization table in
// a storage library would be both wrong and unextendable. The caller states
// the segment instead.
func (r *KeyResolver) SetOwnerSegment(ownerKind, segment string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.segments[ownerKind] = segment
}

// Unregister removes a category's default pattern.
func (r *KeyResolver) Unregister(category string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.patterns, category)
}

// UnregisterFor removes one owner kind's override.
func (r *KeyResolver) UnregisterFor(ownerKind, category string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.overrides[ownerKind] != nil {
		delete(r.overrides[ownerKind], category)
	}
}

// Categories lists every category with a default pattern, sorted for a stable
// result.
func (r *KeyResolver) Categories() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	out := make([]string, 0, len(r.patterns))
	for c := range r.patterns {
		out = append(out, c)
	}
	sortStrings(out)
	return out
}

// BucketFor returns the bucket an owner kind's category resolves to.
func (r *KeyResolver) BucketFor(ownerKind, category string) (string, error) {
	pattern, err := r.patternFor(ownerKind, category)
	if err != nil {
		return "", err
	}
	return pattern.Bucket, nil
}

// Resolve returns the bucket and key for one object.
func (r *KeyResolver) Resolve(req KeyRequest) (bucket, key string, err error) {
	pattern, err := r.patternFor(req.OwnerKind, req.Category)
	if err != nil {
		return "", "", err
	}

	ts := req.Timestamp
	if ts.IsZero() {
		ts = time.Now()
	}

	r.mu.RLock()
	segment, ok := r.segments[req.OwnerKind]
	r.mu.RUnlock()
	if !ok {
		segment = req.OwnerKind
	}

	replacer := strings.NewReplacer(
		"{ownerKind}", sanitizeSegment(segment, ""),
		"{ownerID}", sanitizeSegment(req.OwnerID, "unknown"),
		"{referenceID}", sanitizeSegment(req.ReferenceID, ""),
		"{timestamp}", fmt.Sprintf("%d", ts.UnixNano()/int64(time.Millisecond)),
		"{extension}", sanitizeSegment(strings.TrimPrefix(req.Extension, "."), ""),
	)
	return pattern.Bucket, replacer.Replace(pattern.Pattern), nil
}

// patternFor picks the override for this owner kind, else the category default.
func (r *KeyResolver) patternFor(ownerKind, category string) (KeyPattern, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if byCategory, ok := r.overrides[ownerKind]; ok {
		if pattern, ok := byCategory[category]; ok {
			return pattern, nil
		}
	}
	if pattern, ok := r.patterns[category]; ok {
		return pattern, nil
	}
	return KeyPattern{}, fmt.Errorf("storage: no key pattern registered for category %q (owner kind %q)", category, ownerKind)
}

// unsafeKeySegment matches everything not allowed in a resolved segment.
var unsafeKeySegment = regexp.MustCompile(`[^a-zA-Z0-9._-]`)

// sanitizeSegment makes one substituted value safe to place in a key.
//
// Two things are being prevented. A separator would let a value invent extra
// path structure — an extension of "jpg/../../secret" rewrites where the
// object lands. And "." or ".." as a whole segment is traversal: object stores
// treat keys as opaque, but the local-filesystem backend joins them with
// filepath.Join, which resolves "..", so an unsanitized value escapes the
// bucket directory entirely.
//
// The extension is the field that matters most here: it usually comes from an
// uploaded filename, which the uploader chooses.
func sanitizeSegment(value, fallback string) string {
	cleaned := unsafeKeySegment.ReplaceAllString(value, "_")

	// Reject dot-only segments outright: they carry no information and are
	// exactly the traversal shape.
	if strings.Trim(cleaned, ".") == "" {
		cleaned = ""
	}
	if cleaned == "" {
		return fallback
	}
	return cleaned
}

// sortStrings is a tiny insertion sort, kept local so this file does not pull
// in sort for one call on a list the size of a category registry.
func sortStrings(s []string) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j] < s[j-1]; j-- {
			s[j], s[j-1] = s[j-1], s[j]
		}
	}
}
