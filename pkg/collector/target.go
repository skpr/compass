// Package collector contains the node-local, per-pod trace collector: the parts
// of a DaemonSet deployment that select a single pod on the node and route its
// traces to the subscriber that asked for it.
package collector

import (
	"errors"
	"net/url"

	"github.com/skpr/compass/pkg/collector/pod"
)

// QueryParamUID is the /v1/traces query parameter naming the pod to trace.
const QueryParamUID = "uid"

// ErrMissingUID is returned when a request to the collector omits the pod UID.
// The collector cannot trace "the node", only a named pod on it, so a request
// without a UID is a client error rather than a request for everything.
var ErrMissingUID = errors.New("missing required query parameter: " + QueryParamUID)

// ErrInvalidUID is returned when the pod UID is present but not a pod UID.
var ErrInvalidUID = errors.New("invalid pod UID")

// Target identifies the pod a subscriber wants traces for. It holds the
// normalised UID so two requests naming the same pod in different cgroup-driver
// spellings resolve to one target.
type Target struct {
	// UID is the normalised (lower-case, dash-separated) pod UID.
	UID string
}

// Key returns the value a Target is routed and de-duplicated by.
func (t Target) Key() string {
	return t.UID
}

// ParseTarget extracts and validates the pod target from a request's query
// values. The UID is normalised so the rest of the collector only ever sees the
// canonical form.
func ParseTarget(values url.Values) (Target, error) {
	raw := values.Get(QueryParamUID)
	if raw == "" {
		return Target{}, ErrMissingUID
	}

	uid := pod.NormaliseUID(raw)
	if uid == "" {
		return Target{}, ErrInvalidUID
	}

	return Target{UID: uid}, nil
}
